// Package slowlock serialises the slow integration tests across packages.
package slowlock

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// Acquire blocks until no other test binary holds the shared lock, then holds it until t ends.
// The lock lives in the temp dir so that packages run in parallel by `go test ./...` cannot overlap.
func Acquire(t *testing.T) {
	t.Helper()
	f, err := os.OpenFile(filepath.Join(os.TempDir(), "shopbill-integration.lock"), os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	})
}
