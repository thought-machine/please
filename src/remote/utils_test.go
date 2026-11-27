package remote

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestCredProvider returns a commandCredProvider whose command prints the given token
// and records each invocation, along with a function returning the number of invocations so far.
func newTestCredProvider(t *testing.T, token string) (*commandCredProvider, func() int) {
	t.Helper()
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	countFile := filepath.Join(dir, "count")
	require.NoError(t, os.WriteFile(tokenFile, []byte(token+"\n"), 0644))
	cred := &commandCredProvider{
		command: fmt.Sprintf("echo x >> '%s' && cat '%s'", countFile, tokenFile),
	}
	return cred, func() int {
		b, err := os.ReadFile(countFile)
		if os.IsNotExist(err) {
			return 0
		}
		require.NoError(t, err)
		return strings.Count(string(b), "x")
	}
}

func makeJWT(t *testing.T, expiry time.Time) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(expiry),
	}).SignedString([]byte("test-signing-key"))
	require.NoError(t, err)
	return token
}

func TestCommandCredProviderMetadata(t *testing.T) {
	cred, _ := newTestCredProvider(t, "  opaque-token  ")
	md, err := cred.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"authorization": "Bearer opaque-token"}, md)
	assert.False(t, cred.RequireTransportSecurity())
}

func TestCommandCredProviderOpaqueTokenIsCached(t *testing.T) {
	cred, count := newTestCredProvider(t, "opaque-token")
	for range 3 {
		token, err := cred.getToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "opaque-token", token)
	}
	assert.Equal(t, 1, count(), "non-JWT tokens should be treated as never expiring")
}

func TestCommandCredProviderValidJWTIsCached(t *testing.T) {
	expiry := time.Now().Add(time.Hour).Truncate(time.Second)
	jwtToken := makeJWT(t, expiry)
	cred, count := newTestCredProvider(t, jwtToken)
	for range 3 {
		token, err := cred.getToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, jwtToken, token)
	}
	assert.Equal(t, 1, count())
	expectedExpiry := expiry.Add(-tokenExpiryMargin)
	assert.True(t, expectedExpiry.Equal(cred.expiry), "expected expiry %s, was %s", expectedExpiry, cred.expiry)
}

func TestCommandCredProviderExpiredJWTIsRefreshed(t *testing.T) {
	jwtToken := makeJWT(t, time.Now().Add(-time.Minute))
	cred, count := newTestCredProvider(t, jwtToken)
	for range 3 {
		token, err := cred.getToken(context.Background())
		require.NoError(t, err)
		assert.Equal(t, jwtToken, token)
	}
	assert.Equal(t, 3, count(), "expired tokens should be re-fetched on every call")
}

func TestCommandCredProviderRefreshesAfterExpiry(t *testing.T) {
	cred, count := newTestCredProvider(t, "opaque-token")
	_, err := cred.getToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, count())
	// Simulate the token expiring
	cred.expiry = time.Now().Add(-time.Second)
	_, err = cred.getToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 2, count())
}

func TestCommandCredProviderCommandFails(t *testing.T) {
	cred := &commandCredProvider{command: "echo 'not logged in' >&2; exit 1"}
	_, err := cred.GetRequestMetadata(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not logged in")
	assert.Contains(t, err.Error(), "exit status 1")
}
