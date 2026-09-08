package testutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewRedis starts an isolated redis-server for an integration test.
func NewRedis(t *testing.T) *redis.Client {
	t.Helper()
	serverPath, err := exec.LookPath("redis-server")
	if err != nil {
		t.Skip("redis-server is required for this integration test")
	}
	socket := fmt.Sprintf("/tmp/koyomi-test-redis-%d-%d.sock", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() { _ = os.Remove(socket) })
	var output bytes.Buffer
	command := exec.Command(
		serverPath,
		"--port", "0",
		"--unixsocket", socket,
		"--unixsocketperm", "700",
		"--save", "",
		"--appendonly", "no",
	)
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Start(); err != nil {
		t.Skipf("start redis-server: %v", err)
	}
	t.Cleanup(func() {
		_ = command.Process.Signal(os.Interrupt)
		_ = command.Wait()
	})

	client := redis.NewClient(&redis.Options{Network: "unix", Addr: socket})
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(3 * time.Second)
	for {
		if err := client.Ping(context.Background()).Err(); err == nil {
			return client
		}
		if time.Now().After(deadline) {
			t.Fatalf("redis-server did not become ready: %s", output.String())
		}
		time.Sleep(10 * time.Millisecond)
	}
}
