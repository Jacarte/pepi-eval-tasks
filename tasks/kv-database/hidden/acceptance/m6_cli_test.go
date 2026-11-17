package acceptance

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

var (
	cliOnce sync.Once
	cliPath string
	cliErr  error
	cliDir  string
)

func TestMain(m *testing.M) {
	code := m.Run()
	if cliDir != "" {
		os.RemoveAll(cliDir)
	}
	os.Exit(code)
}

func goTool() string {
	if p, err := exec.LookPath("go"); err == nil {
		return p
	}
	return filepath.Join(runtime.GOROOT(), "bin", "go")
}

// buildCLI builds kvdb/cmd/kvdb once per test binary.
func buildCLI(t *testing.T) string {
	t.Helper()
	cliOnce.Do(func() {
		cliDir, cliErr = os.MkdirTemp("", "kvdb-cli-")
		if cliErr != nil {
			return
		}
		cliPath = filepath.Join(cliDir, "kvdb")
		if runtime.GOOS == "windows" {
			cliPath += ".exe"
		}
		out, err := exec.Command(goTool(), "build", "-o", cliPath, "kvdb/cmd/kvdb").CombinedOutput()
		if err != nil {
			cliErr = errors.New("go build kvdb/cmd/kvdb: " + err.Error() + "\n" + string(out))
		}
	})
	if cliErr != nil {
		t.Fatal(cliErr)
	}
	return cliPath
}

type cliResult struct {
	code           int
	stdout, stderr string
}

func runCLI(t *testing.T, args ...string) cliResult {
	t.Helper()
	cmd := exec.Command(buildCLI(t), args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(guard):
		cmd.Process.Kill()
		<-done
		t.Fatalf("kvdb %q did not exit", args)
	}
	return cliResult{code: cmd.ProcessState.ExitCode(), stdout: out.String(), stderr: errb.String()}
}

func wantCLI(t *testing.T, r cliResult, code int, stdout string) {
	t.Helper()
	if r.code != code || r.stdout != stdout {
		t.Fatalf("exit %d stdout %q (stderr %q); want exit %d stdout %q", r.code, r.stdout, r.stderr, code, stdout)
	}
}

type serveProc struct {
	cmd    *exec.Cmd
	addr   string
	done   chan struct{}
	stderr *bytes.Buffer
}

// startServe runs "kvdb serve" and waits for its "listening on" line.
func startServe(t *testing.T, args ...string) *serveProc {
	t.Helper()
	cmd := exec.Command(buildCLI(t), append([]string{"serve"}, args...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	p := &serveProc{cmd: cmd, done: make(chan struct{}), stderr: &bytes.Buffer{}}
	cmd.Stderr = p.stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	lines := make(chan string, 1)
	go func() {
		r := bufio.NewReader(stdout)
		l, _ := r.ReadString('\n')
		lines <- l
		io.Copy(io.Discard, r)
		cmd.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		select {
		case <-p.done:
		default:
			cmd.Process.Kill()
			<-p.done
		}
	})
	select {
	case l := <-lines:
		if !strings.HasPrefix(l, "listening on ") || !strings.HasSuffix(l, "\n") {
			t.Fatalf("serve printed %q (stderr %q), want \"listening on HOST:PORT\"", l, p.stderr.String())
		}
		p.addr = strings.TrimSuffix(strings.TrimPrefix(l, "listening on "), "\n")
	case <-time.After(guard):
		t.Fatal("serve did not print its address")
	}
	if _, port, err := net.SplitHostPort(p.addr); err != nil || port == "0" {
		t.Fatalf("serve printed address %q", p.addr)
	}
	return p
}

func (p *serveProc) wait(t *testing.T) int {
	t.Helper()
	select {
	case <-p.done:
	case <-time.After(guard):
		t.Fatal("serve did not exit")
	}
	return p.cmd.ProcessState.ExitCode()
}

func (p *serveProc) stop(t *testing.T, sig os.Signal) int {
	t.Helper()
	if err := p.cmd.Process.Signal(sig); err != nil {
		t.Fatal(err)
	}
	return p.wait(t)
}

func TestM6UsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"frobnicate"},
		{"get"},
		{"get", "a", "b"},
		{"set", "k"},
		{"set", "k", "v", "extra"},
		{"del"},
		{"get", "--nope", "k"},
		{"serve"},
		{"serve", "--dir", t.TempDir(), "--sync", "sometimes"},
	} {
		r := runCLI(t, args...)
		if r.code != 2 || r.stdout != "" {
			t.Errorf("kvdb %q: exit %d stdout %q; want exit 2 and empty stdout", args, r.code, r.stdout)
		}
	}
}

func TestM6SetGetDel(t *testing.T) {
	p := startServe(t, "--dir", t.TempDir(), "--addr", "127.0.0.1:0")
	a := p.addr
	wantCLI(t, runCLI(t, "get", "--addr", a, "missing"), 1, "")
	wantCLI(t, runCLI(t, "set", "--addr", a, "name", "alice"), 0, "OK\n")
	wantCLI(t, runCLI(t, "get", "-addr", a, "name"), 0, "alice\n")
	wantCLI(t, runCLI(t, "set", "--addr", a, "name", "bob"), 0, "OK\n")
	wantCLI(t, runCLI(t, "get", "--addr", a, "name"), 0, "bob\n")
	wantCLI(t, runCLI(t, "del", "--addr", a, "name"), 0, "deleted\n")
	wantCLI(t, runCLI(t, "del", "--addr", a, "name"), 1, "")
	wantCLI(t, runCLI(t, "get", "--addr", a, "name"), 1, "")
	if code := p.stop(t, syscall.SIGTERM); code != 0 {
		t.Fatalf("serve exit code after SIGTERM = %d, want 0 (stderr %q)", code, p.stderr.String())
	}
}

func TestM6ArbitraryBytesRoundTrip(t *testing.T) {
	p := startServe(t, "--dir", t.TempDir(), "--addr", "127.0.0.1:0", "--sync", "never")
	values := map[string]string{
		"with space":       "hello  world",
		`quo"te`:           `say "hi" \n not a newline \\`,
		"multi\nline\tkey": "line1\nline2\r\n\ttabbed",
		"utf8-ключ":        "héllo ✓ \x01\x7f",
		"empty":            "",
	}
	for k, v := range values {
		wantCLI(t, runCLI(t, "set", "--addr", p.addr, k, v), 0, "OK\n")
	}
	for k, v := range values {
		wantCLI(t, runCLI(t, "get", "--addr", p.addr, k), 0, v+"\n")
	}
	// The server sees the exact bytes too.
	c := dial(t, p.addr)
	c.expect("GET "+q(`quo"te`), q(`say "hi" \n not a newline \\`))
	c.expect("COUNT \"\"", "(integer) 5")
}

func TestM6ServerErrorAndConnectionFailure(t *testing.T) {
	p := startServe(t, "--dir", t.TempDir(), "--addr", "127.0.0.1:0")
	r := runCLI(t, "set", "--addr", p.addr, "", "v")
	if r.code != 4 || r.stdout != "" || !strings.Contains(r.stderr, "empty key") {
		t.Fatalf("set empty key: exit %d stdout %q stderr %q; want exit 4 and the server error on stderr", r.code, r.stdout, r.stderr)
	}
	// An address nothing listens on.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	dead := ln.Addr().String()
	ln.Close()
	for _, args := range [][]string{{"get", "--addr", dead, "k"}, {"set", "--addr", dead, "k", "v"}, {"del", "--addr", dead, "k"}} {
		r := runCLI(t, args...)
		if r.code != 3 || r.stdout != "" {
			t.Errorf("kvdb %q against dead address: exit %d stdout %q; want 3", args, r.code, r.stdout)
		}
	}
}

func TestM6ServeFailures(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	r := runCLI(t, "serve", "--dir", t.TempDir(), "--addr", ln.Addr().String())
	if r.code != 3 || r.stdout != "" {
		t.Fatalf("serve on busy address: exit %d stdout %q; want 3", r.code, r.stdout)
	}
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	r = runCLI(t, "serve", "--dir", file, "--addr", "127.0.0.1:0")
	if r.code != 3 || r.stdout != "" {
		t.Fatalf("serve with a file as --dir: exit %d stdout %q; want 3", r.code, r.stdout)
	}
}

func TestM6DataSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	p := startServe(t, "--dir", dir, "--addr", "127.0.0.1:0")
	wantCLI(t, runCLI(t, "set", "--addr", p.addr, "graceful", "kept"), 0, "OK\n")
	wantCLI(t, runCLI(t, "set", "--addr", p.addr, "gone", "x"), 0, "OK\n")
	wantCLI(t, runCLI(t, "del", "--addr", p.addr, "gone"), 0, "deleted\n")
	if code := p.stop(t, os.Interrupt); code != 0 {
		t.Fatalf("serve exit code after SIGINT = %d, want 0", code)
	}

	p2 := startServe(t, "--dir", dir, "--addr", "127.0.0.1:0")
	wantCLI(t, runCLI(t, "get", "--addr", p2.addr, "graceful"), 0, "kept\n")
	wantCLI(t, runCLI(t, "get", "--addr", p2.addr, "gone"), 1, "")
	wantCLI(t, runCLI(t, "set", "--addr", p2.addr, "crash", "proof"), 0, "OK\n")
	// SIGKILL: no chance to flush or close anything.
	p2.stop(t, syscall.SIGKILL)

	p3 := startServe(t, "--dir", dir, "--addr", "127.0.0.1:0")
	wantCLI(t, runCLI(t, "get", "--addr", p3.addr, "crash"), 0, "proof\n")
	wantCLI(t, runCLI(t, "get", "--addr", p3.addr, "graceful"), 0, "kept\n")
	if code := p3.stop(t, syscall.SIGTERM); code != 0 {
		t.Fatalf("serve exit code after SIGTERM = %d, want 0", code)
	}
	// Graceful shutdown with a client still connected must not hang.
	p4 := startServe(t, "--dir", dir, "--addr", "127.0.0.1:0")
	c := dial(t, p4.addr)
	c.expect("GET crash", `"proof"`)
	if code := p4.stop(t, syscall.SIGTERM); code != 0 {
		t.Fatalf("serve exit code with a connected client = %d, want 0", code)
	}
	c.expectClosed()
}
