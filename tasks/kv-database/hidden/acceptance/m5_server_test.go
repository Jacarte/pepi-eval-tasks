package acceptance

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"kvdb/query"
	"kvdb/server"
	"kvdb/store"
)

// guard is a failure guard only; correct implementations finish far sooner.
const guard = 10 * time.Second

func startServer(t *testing.T) (*store.Store, *server.Server) {
	t.Helper()
	st := store.New()
	srv := server.New(query.NewEngine(st))
	if err := srv.Start("127.0.0.1:0"); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), guard)
		defer cancel()
		srv.Shutdown(ctx)
	})
	return st, srv
}

type client struct {
	t  *testing.T
	nc net.Conn
	r  *bufio.Reader
}

func dial(t *testing.T, addr string) *client {
	t.Helper()
	nc, err := net.DialTimeout("tcp", addr, guard)
	if err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	nc.SetDeadline(time.Now().Add(3 * guard))
	t.Cleanup(func() { nc.Close() })
	return &client{t: t, nc: nc, r: bufio.NewReaderSize(nc, 1<<16)}
}

func (c *client) send(s string) {
	c.t.Helper()
	if _, err := io.WriteString(c.nc, s); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

func (c *client) readLine() (string, error) {
	line, err := c.r.ReadString('\n')
	if err != nil {
		return line, err
	}
	return strings.TrimSuffix(line, "\n"), nil
}

// readResp reads one full response, including the lines of an array.
func (c *client) readResp() (string, error) {
	first, err := c.readLine()
	if err != nil {
		return first, err
	}
	if !strings.HasPrefix(first, "(array) ") {
		return first, nil
	}
	n, err := strconv.Atoi(strings.TrimPrefix(first, "(array) "))
	if err != nil {
		return first, fmt.Errorf("bad array header %q", first)
	}
	lines := []string{first}
	for i := 0; i < n; i++ {
		l, err := c.readLine()
		if err != nil {
			return strings.Join(lines, "\n"), err
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n"), nil
}

func (c *client) do(line string) string {
	c.t.Helper()
	c.send(line + "\n")
	resp, err := c.readResp()
	if err != nil {
		c.t.Fatalf("response to %q: %v (partial %q)", line, err, resp)
	}
	return resp
}

func (c *client) expect(line, want string) {
	c.t.Helper()
	if got := c.do(line); got != want {
		c.t.Fatalf("%q\n got: %q\nwant: %q", line, got, want)
	}
}

// expectClosed asserts that the server closed the connection.
func (c *client) expectClosed() {
	c.t.Helper()
	c.nc.SetReadDeadline(time.Now().Add(guard))
	b, err := c.r.ReadByte()
	if err == nil {
		c.t.Fatalf("expected connection closed, read byte %q", b)
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		c.t.Fatal("connection not closed by server")
	}
}

func shutdown(t *testing.T, srv *server.Server) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), guard)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestM5AddrAndBasicCommands(t *testing.T) {
	st, srv := startServer(t)
	host, port, err := net.SplitHostPort(srv.Addr())
	if err != nil || host != "127.0.0.1" || port == "0" || port == "" {
		t.Fatalf("Addr() = %q, want 127.0.0.1:<real port>", srv.Addr())
	}
	if server.New(query.NewEngine(store.New())).Addr() != "" {
		t.Fatal("Addr() before Start must be empty")
	}
	c := dial(t, srv.Addr())
	c.expect("SET greeting \"hello world\"", "OK")
	c.expect("GET greeting", `"hello world"`)
	c.expect("SET user:2 b", "OK")
	c.expect("SET user:1 a", "OK")
	c.expect("SCAN user:", arrayResp([2]string{"user:1", "a"}, [2]string{"user:2", "b"}))
	c.expect("SCAN none", "(array) 0")
	c.expect("COUNT user:", "(integer) 2")
	c.expect("DEL user:1", "(integer) 1")
	c.expect("bogus", "(error) ERR unknown command 'bogus'")
	c.expect("", "(error) ERR empty command")
	mustGet(t, st, "greeting", "hello world")
	// CRLF line endings.
	c.send("GET user:2\r\n")
	if r, err := c.readResp(); err != nil || r != `"b"` {
		t.Fatalf("CRLF request: %q, %v", r, err)
	}
	c.expect("quit", "OK")
	c.expectClosed()
}

func TestM5StartErrors(t *testing.T) {
	_, srv := startServer(t)
	if err := srv.Start("127.0.0.1:0"); err == nil {
		t.Fatal("second Start succeeded")
	}
	other := server.New(query.NewEngine(store.New()))
	if err := other.Start(srv.Addr()); err == nil {
		other.Shutdown(context.Background())
		t.Fatal("Start on an address in use succeeded")
	}
}

func TestM5PipelinedRequests(t *testing.T) {
	st, srv := startServer(t)
	c := dial(t, srv.Addr())
	var req bytes.Buffer
	const n = 500
	for i := 0; i < n; i++ {
		fmt.Fprintf(&req, "SET p%03d v%d\nGET p%03d\nSCAN p%03d\n", i, i, i, i)
	}
	go io.Copy(c.nc, bytes.NewReader(req.Bytes()))
	for i := 0; i < n; i++ {
		for _, want := range []string{"OK", fmt.Sprintf(`"v%d"`, i), arrayResp([2]string{fmt.Sprintf("p%03d", i), fmt.Sprintf("v%d", i)})} {
			got, err := c.readResp()
			if err != nil || got != want {
				t.Fatalf("pipelined response %d: %q, %v; want %q", i, got, err, want)
			}
		}
	}
	if st.Len() != n {
		t.Fatalf("store has %d keys, want %d", st.Len(), n)
	}
}

func TestM5LongLines(t *testing.T) {
	st, srv := startServer(t)
	c := dial(t, srv.Addr())
	big := bytes.Repeat([]byte("0123456789abcdef"), 40000) // 640 KB
	big[100] = '"'
	big[200] = 0
	line := "SET big " + specQuote(big)
	go io.WriteString(c.nc, line+"\n"+"GET big\n")
	if r, err := c.readResp(); err != nil || r != "OK" {
		t.Fatalf("SET big: %q, %v", r, err)
	}
	r, err := c.readResp()
	if err != nil || r != specQuote(big) {
		t.Fatalf("GET big: len %d, %v", len(r), err)
	}
	got, _ := st.Get("big")
	if !bytes.Equal(got, big) {
		t.Fatal("stored big value differs")
	}
}

func TestM5PerConnectionTransactions(t *testing.T) {
	st, srv := startServer(t)
	a := dial(t, srv.Addr())
	b := dial(t, srv.Addr())
	a.expect("BEGIN", "OK")
	a.expect("SET tx 1", "OK")
	b.expect("GET tx", "(nil)")
	b.expect("COMMIT", "(error) ERR no transaction in progress")
	b.expect("BEGIN", "OK")
	b.expect("SET tx 2", "OK")
	a.expect("COMMIT", "OK")
	b.expect("COMMIT", "(error) ERR transaction conflict")
	b.expect("GET tx", `"1"`)
	mustGet(t, st, "tx", "1")

	// A client that disconnects mid-transaction leaves nothing behind.
	d := dial(t, srv.Addr())
	d.expect("BEGIN", "OK")
	d.expect("SET abandoned yes", "OK")
	d.nc.Close()
	a.expect("GET abandoned", "(nil)")
	mustMissing(t, st, "abandoned")
}

func TestM5ConcurrentClients(t *testing.T) {
	st, srv := startServer(t)
	setup := dial(t, srv.Addr())
	setup.expect("SET counter 0", "OK")
	const clients, ops = 16, 40
	var wg sync.WaitGroup
	errs := make(chan error, clients)
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nc, err := net.DialTimeout("tcp", srv.Addr(), guard)
			if err != nil {
				errs <- err
				return
			}
			defer nc.Close()
			nc.SetDeadline(time.Now().Add(3 * guard))
			c := &client{t: t, nc: nc, r: bufio.NewReader(nc)}
			call := func(line string) string {
				if _, err := io.WriteString(nc, line+"\n"); err != nil {
					return "write error: " + err.Error()
				}
				r, err := c.readResp()
				if err != nil {
					return "read error: " + err.Error()
				}
				return r
			}
			for j := 0; j < ops; j++ {
				k := fmt.Sprintf("c%02d:%02d", id, j)
				if r := call("SET " + k + " " + k); r != "OK" {
					errs <- fmt.Errorf("SET %s: %q", k, r)
					return
				}
				if r := call("GET " + k); r != q(k) {
					errs <- fmt.Errorf("GET %s: %q", k, r)
					return
				}
				for {
					call("BEGIN")
					v, err := strconv.Unquote(call("GET counter"))
					if err != nil {
						errs <- err
						return
					}
					n, _ := strconv.Atoi(v)
					call(fmt.Sprintf("SET counter %d", n+1))
					r := call("COMMIT")
					if r == "OK" {
						break
					}
					if r != "(error) ERR transaction conflict" {
						errs <- fmt.Errorf("COMMIT: %q", r)
						return
					}
				}
			}
			if r := call(fmt.Sprintf("COUNT c%02d:", id)); r != fmt.Sprintf("(integer) %d", ops) {
				errs <- fmt.Errorf("COUNT: %q", r)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	mustGet(t, st, "counter", strconv.Itoa(clients*ops))
	if st.Len() != clients*ops+1 {
		t.Fatalf("store has %d keys, want %d", st.Len(), clients*ops+1)
	}
}

func TestM5GracefulShutdown(t *testing.T) {
	st, srv := startServer(t)
	addr := srv.Addr()
	idle := dial(t, addr)
	idle.expect("SET committed yes", "OK")
	tx := dial(t, addr)
	tx.expect("BEGIN", "OK")
	tx.expect("SET uncommitted yes", "OK")
	partial := dial(t, addr)
	partial.expect("GET committed", `"yes"`)
	partial.send("SET partial line-without-newline")
	silent := dial(t, addr) // connected, never sends anything
	_ = silent

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), guard)
		defer cancel()
		done <- srv.Shutdown(ctx)
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Shutdown: %v", err)
		}
	case <-time.After(guard + time.Second):
		t.Fatal("Shutdown did not return with idle clients connected")
	}
	for _, c := range []*client{idle, tx, partial, silent} {
		c.expectClosed()
	}
	mustGet(t, st, "committed", "yes")
	mustMissing(t, st, "uncommitted")
	mustMissing(t, st, "partial")
	if nc, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		nc.SetDeadline(time.Now().Add(guard))
		io.WriteString(nc, "GET committed\n")
		b, rerr := bufio.NewReader(nc).ReadString('\n')
		nc.Close()
		if rerr == nil {
			t.Fatalf("server still answering after Shutdown: %q", b)
		}
	}
	// The store stays open: Shutdown must not close it.
	mustPut(t, st, "after", "shutdown")
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}
}

// Clients hammering the server while it shuts down: every acknowledged
// write is applied, and Shutdown returns.
func TestM5ShutdownUnderLoad(t *testing.T) {
	st, srv := startServer(t)
	const clients = 8
	acked := make([][]string, clients)
	var wg sync.WaitGroup
	started := make(chan struct{}, clients)
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nc, err := net.DialTimeout("tcp", srv.Addr(), guard)
			if err != nil {
				t.Error(err)
				started <- struct{}{}
				return
			}
			defer nc.Close()
			nc.SetDeadline(time.Now().Add(3 * guard))
			r := bufio.NewReader(nc)
			for j := 0; ; j++ {
				k := fmt.Sprintf("load%d:%d", id, j)
				if _, err := io.WriteString(nc, "SET "+k+" x\n"); err != nil {
					return
				}
				line, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if line != "OK\n" {
					t.Errorf("SET %s: %q", k, line)
					return
				}
				acked[id] = append(acked[id], k)
				if j == 20 {
					started <- struct{}{}
				}
			}
		}(i)
	}
	for i := 0; i < clients; i++ {
		select {
		case <-started:
		case <-time.After(guard):
			t.Fatal("clients did not get going")
		}
	}
	shutdown(t, srv)
	finished := make(chan struct{})
	go func() { wg.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(guard):
		t.Fatal("clients still being served after Shutdown returned")
	}
	total := 0
	for _, keys := range acked {
		for _, k := range keys {
			mustGet(t, st, k, "x")
			total++
		}
	}
	if total < clients*20 {
		t.Fatalf("only %d acknowledged writes", total)
	}
}
