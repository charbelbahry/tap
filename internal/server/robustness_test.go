package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor polls until cond is true, or fails the test after 5 seconds.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

// syncBuffer is a log destination that is safe to use from several goroutines.
type syncBuffer struct {
	mu sync.Mutex
	sb strings.Builder
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sb.Write(p)
}

// levels returns the log level of each msg seen so far.
func (b *syncBuffer) levels() map[string]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := map[string]string{}
	for line := range strings.SplitSeq(b.sb.String(), "\n") {
		var rec struct{ Level, Msg string }
		if json.Unmarshal([]byte(line), &rec) == nil {
			out[rec.Msg] = rec.Level
		}
	}
	return out
}

// A client that stops reading must be disconnected instead of blocking the sender.
func TestSendClosesSlowClient(t *testing.T) {
	a, b := net.Pipe()
	defer b.Close()
	c := &client{conn: a, out: make(chan string, 1)}
	c.send("fits in the buffer")
	c.send("buffer is full, so this closes the connection")
	if _, err := a.Write([]byte("x")); err == nil {
		t.Fatal("connection should be closed once the buffer is full")
	}
}

func TestLineLengthBoundary(t *testing.T) {
	addr := startServer(t)
	c, r := login(t, addr, "alice")
	expect(t, c, r, "EVT STATS players=1")

	msg := strings.Repeat("a", maxLine-len("CHAT GLOBAL ")) // a line of exactly maxLine bytes
	send(t, c, "CHAT GLOBAL "+msg+"\n")
	expect(t, c, r, "OK")
	expect(t, c, r, "EVT GLOBAL CHAT alice "+msg)

	send(t, c, "CHAT GLOBAL "+msg+"b\n") // one byte over
	expect(t, c, r, "ERR 400 BAD_REQUEST")
}

func TestNameLengthBoundary(t *testing.T) {
	addr := startServer(t)
	c, r := dial(t, addr)
	send(t, c, "CONNECT "+strings.Repeat("é", 32)+"\n") // 32 characters, 64 bytes
	expect(t, c, r, "OK connected")
}

// A busy window ends: after a second, commands are accepted again.
func TestRateLimitWindowResets(t *testing.T) {
	addr := startServer(t)
	c, r := login(t, addr, "alice")
	expect(t, c, r, "EVT STATS players=1")
	send(t, c, strings.Repeat("WHO\n", cmdLimit))
	for range cmdLimit - 1 { // CONNECT already used one of the 10
		expect(t, c, r, "OK players=1")
	}
	expect(t, c, r, "ERR 429 RATE_LIMITED")

	time.Sleep(1100 * time.Millisecond)
	send(t, c, "WHO\n")
	expect(t, c, r, "OK players=1")
}

// The 100-connection limit, and the WARN logs the subject asks for.
func TestConnectionLimitAndAbuseLogs(t *testing.T) {
	logs := &syncBuffer{}
	addr := startServerLogged(t, slog.NewJSONHandler(logs, nil))

	conns := make([]net.Conn, maxClients)
	for i := range conns {
		var r *bufio.Reader
		conns[i], r = dial(t, addr)
		_ = r
	}
	over, r := net.Dial("tcp", addr)
	if over == nil {
		t.Fatal(r)
	}
	defer over.Close()
	expect(t, over, bufio.NewReader(over), "ERR 900 CONNECTION_FAILED")

	// Flood on one of the open connections, and a too-long line on another.
	send(t, conns[0], "CONNECT alice\n"+strings.Repeat("WHO\n", cmdLimit+5))
	send(t, conns[1], strings.Repeat("a", 2000)+"\n")

	waitFor(t, "WARN logs", func() bool {
		l := logs.levels()
		return l["flood"] == "WARN" && l["rapid_connections"] == "WARN" &&
			l["server_full"] == "WARN" && l["line_too_long"] == "WARN"
	})

	// Closing one connection frees a slot.
	conns[2].Close()
	waitFor(t, "a free slot", func() bool {
		c, err := net.Dial("tcp", addr)
		if err != nil {
			return false
		}
		defer c.Close()
		line, _ := bufio.NewReader(c).ReadString('\n')
		return strings.HasPrefix(line, "OK hello")
	})
}

// reader counts what one client receives.
type reader struct {
	chats   atomic.Int32
	players atomic.Int32 // last EVT STATS value
}

func (rd *reader) run(r *bufio.Reader) {
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if strings.HasPrefix(line, "EVT GLOBAL CHAT") {
			rd.chats.Add(1)
		}
		var n int32
		if _, err := fmt.Sscanf(line, "EVT STATS players=%d", &n); err == nil {
			rd.players.Store(n)
		}
	}
}

// Many clients at once: unique names, every broadcast delivered to everyone,
// and abrupt disconnects settle to the right player count. Run with -race.
func TestConcurrentClients(t *testing.T) {
	const clients, chats = 20, 5
	addr := startServer(t)

	// Ten clients race for one name: exactly one wins.
	var wins atomic.Int32
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, r := dial(t, addr)
			send(t, c, "CONNECT dup\n")
			if line, _ := r.ReadString('\n'); line == "OK connected\n" {
				wins.Add(1)
			}
		}()
	}
	wg.Wait()
	if wins.Load() != 1 {
		t.Fatalf("%d clients got the same name, want 1", wins.Load())
	}

	conns := make([]net.Conn, clients)
	readers := make([]*reader, clients)
	for i := range clients {
		c, r := dial(t, addr)
		conns[i], readers[i] = c, &reader{}
		send(t, c, fmt.Sprintf("CONNECT p%d\n", i))
		go readers[i].run(r)
	}
	waitFor(t, "everyone to log in", func() bool {
		return readers[clients-1].players.Load() == clients+1 // the dup client counts too
	})

	// Everybody chats at the same time.
	for i := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range chats {
				send(t, conns[i], fmt.Sprintf("CHAT GLOBAL p%d says %d\n", i, j))
			}
		}()
	}
	wg.Wait()
	waitFor(t, "every client to get every chat", func() bool {
		for _, rd := range readers {
			if rd.chats.Load() != clients*chats {
				return false
			}
		}
		return true
	})

	// All but one drop without QUIT. The survivor ends up seeing 2 players (itself and dup).
	for _, c := range conns[1:] {
		c.Close()
	}
	waitFor(t, "the player count to settle", func() bool { return readers[0].players.Load() == 2 })
}
