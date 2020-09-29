package benchmark

import (
	"github.com/stretchr/testify/require"

	"os"
	"os/exec"
	"testing"
	"time"
)

func TestBenchDuration(t *testing.T) {
	r := require.New(t)

	dataCmd, _ := os.LookupEnv("DATA")
	cmd := exec.Command(dataCmd)

	start := time.Now()
	out, err := cmd.Output()
	r.NoError(err)

	r.Greater(int64(time.Since(start)), int64(time.Second), "Benchmark run was too quick to have actually run the tests")

	r.Contains(string(out), "BenchmarkOneSecWait")
	r.Contains(string(out), "Benchmark100msWait")
}
