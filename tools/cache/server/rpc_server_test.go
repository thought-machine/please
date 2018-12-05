// Test for the rpc server.
package server

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	pb "github.com/thought-machine/please/src/cache/proto/rpc_cache"
)

const (
	testDir   = "src/cache/test_data"
	testKey   = "src/cache/test_data/key.pem"
	testCert  = "src/cache/test_data/cert_signed.pem"
	testCert2 = "src/cache/test_data/cert.pem"
	testCa    = "src/cache/test_data/ca.pem"
)

func startServer(port int, auth bool, readonlyCerts, writableCerts string) *grpc.Server {
	cache := NewCache(testDir, 20*time.Hour, 100, 1000000, 1000000)
	if !auth {
		s, lis := BuildGrpcServer(port, cache, nil, "", "", "", readonlyCerts, writableCerts)
		go s.Serve(lis)
		return s
	}
	s, lis := BuildGrpcServer(port, cache, nil, testKey, testCert, testCa, readonlyCerts, writableCerts)
	go s.Serve(lis)
	return s
}

func buildClient(t *testing.T, port int, auth bool) pb.RpcCacheClient {
	const maxSize = 10 * 1024 * 1024
	sizeOpts := grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxSize), grpc.MaxCallSendMsgSize(maxSize))
	url := fmt.Sprintf("localhost:%d", port)
	if !auth {
		conn, err := grpc.Dial(url, grpc.WithInsecure(), grpc.WithTimeout(5*time.Second), sizeOpts)
		assert.NoError(t, err)
		return pb.NewRpcCacheClient(conn)
	}
	cert, err := tls.LoadX509KeyPair(testCert, testKey)
	assert.NoError(t, err)
	ca, err := ioutil.ReadFile(testCa)
	assert.NoError(t, err)
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      x509.NewCertPool(),
	}
	assert.True(t, config.RootCAs.AppendCertsFromPEM(ca))
	conn, err := grpc.Dial(url, grpc.WithTransportCredentials(credentials.NewTLS(&config)), grpc.WithTimeout(5*time.Second), sizeOpts)
	assert.NoError(t, err)
	return pb.NewRpcCacheClient(conn)
}

func ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func TestNoAuth(t *testing.T) {
	s := startServer(7677, false, "", "")
	defer s.Stop()
	c := buildClient(t, 7677, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Store(ctx, &pb.StoreRequest{})
	assert.NoError(t, err)
}

func TestAuthRequired(t *testing.T) {
	s := startServer(7678, true, "", "")
	defer s.Stop()
	c := buildClient(t, 7678, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Store(ctx, &pb.StoreRequest{})
	assert.Error(t, err, "Fails because the client doesn't use TLS")
}

func TestReadonlyAuth(t *testing.T) {
	s := startServer(7679, true, testCert, testCert2)
	defer s.Stop()
	c := buildClient(t, 7679, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Retrieve(ctx, &pb.RetrieveRequest{})
	assert.NoError(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.Store(ctx, &pb.StoreRequest{})
	assert.Error(t, err, "Fails because the client isn't authenticated")
}

func TestWritableAuth(t *testing.T) {
	s := startServer(7680, true, testCert, testCert)
	defer s.Stop()
	c := buildClient(t, 7680, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Retrieve(ctx, &pb.RetrieveRequest{})
	assert.NoError(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = c.Store(ctx, &pb.StoreRequest{})
	assert.NoError(t, err)
}

func TestDeleteNoAuth(t *testing.T) {
	s := startServer(7681, true, testCert, testCert2)
	defer s.Stop()
	c := buildClient(t, 7681, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Delete(ctx, &pb.DeleteRequest{})
	assert.Error(t, err, "Fails because the client isn't authenticated")
}

func TestDeleteAuth(t *testing.T) {
	s := startServer(7682, true, testCert, testCert)
	defer s.Stop()
	c := buildClient(t, 7682, true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Delete(ctx, &pb.DeleteRequest{})
	assert.NoError(t, err)
}

func TestMaxMessageSize(t *testing.T) {
	s := startServer(7677, false, "", "")
	defer s.Stop()
	c := buildClient(t, 7677, false)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := c.Store(ctx, &pb.StoreRequest{
		Os:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Hash: bytes.Repeat([]byte{'a'}, 28),
		Artifacts: []*pb.Artifact{
			{
				Package: "src/cache/server",
				Target:  "size_test",
				File:    "size_test.txt",
				Body:    bytes.Repeat([]byte{'a'}, 5*1024*1024),
			},
		},
	})
	assert.NoError(t, err)
}
