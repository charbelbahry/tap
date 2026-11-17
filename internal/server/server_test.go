package server

import (
	"bufio"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/world"
)

// startServer runs a server on a random port and returns its address.
func startServer(t *testing.T) string {
	return startServerLogged(t, slog.DiscardHandler)
}

// startServerLogged is startServer with a chosen log handler.
func startServerLogged(t *testing.T, h slog.Handler) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(h)
	go New(game.New(&world.World{}, log), log).Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// dial connects and checks the greeting.
func dial(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	r := bufio.NewReader(conn)
	expect(t, conn, r, "OK hello proto=1")
	return conn, r
}

// login dials and connects as name, draining the connect replies and events.
func login(t *testing.T, addr, name string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, r := dial(t, addr)
	send(t, conn, "CONNECT "+name+"\n")
	expect(t, conn, r, "OK connected")
	return conn, r
}

func send(t *testing.T, conn net.Conn, s string) {
	t.Helper()
	if _, err := conn.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
}

func expect(t *testing.T, conn net.Conn, r *bufio.Reader, want string) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	got, err := r.ReadString('\n')
	if err != nil || strings.TrimSuffix(got, "\n") != want {
		t.Fatalf("got %q (err %v), want %q", got, err, want)
	}
}

func TestConnectAndDuplicateName(t *testing.T) {
	addr := startServer(t)
	a, ra := dial(t, addr)
	send(t, a, "CONNECT alice\n")
	expect(t, a, ra, "OK connected")
	expect(t, a, ra, "EVT STATS players=1")

	b, rb := dial(t, addr)
	send(t, b, "CONNECT alice\n")
	expect(t, b, rb, "ERR 201 NAME_IN_USE")
}

func TestBadNames(t *testing.T) {
	addr := startServer(t)
	c, r := dial(t, addr)
	send(t, c, "CONNECT\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
	send(t, c, "CONNECT bob smith\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
	send(t, c, "CONNECT "+strings.Repeat("x", 33)+"\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
	send(t, c, "connect алиса\n") // lowercase command, Unicode name
	expect(t, c, r, "OK connected")
}

func TestStateErrors(t *testing.T) {
	addr := startServer(t)
	c, r := dial(t, addr)
	send(t, c, "LOOK\n")
	expect(t, c, r, "ERR 403 NOT_AUTHENTICATED")
	send(t, c, "CONNECT alice\n")
	expect(t, c, r, "OK connected")
	expect(t, c, r, "EVT STATS players=1")
	send(t, c, "CONNECT alice2\n")
	expect(t, c, r, "ERR 402 ALREADY_CONNECTED")
	send(t, c, "FLY\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
}

func TestFramingAndBadInput(t *testing.T) {
	addr := startServer(t)
	c, r := dial(t, addr)

	// One command split across packets.
	send(t, c, "CONN")
	time.Sleep(50 * time.Millisecond)
	send(t, c, "ECT alice\n")
	expect(t, c, r, "OK connected")
	expect(t, c, r, "EVT STATS players=1")

	// Several commands in one packet, with CRLF on one of them.
	send(t, c, "WHO\r\nWHO\n")
	expect(t, c, r, "OK players=1")
	expect(t, c, r, "OK players=1")

	// Control characters and invalid UTF-8.
	send(t, c, "WHO \x07\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
	send(t, c, "WHO \xff\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")

	// A line that is too long gets an error, then the connection closes.
	send(t, c, strings.Repeat("a", 2000)+"\n")
	expect(t, c, r, "ERR 400 BAD_REQUEST")
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := r.ReadString('\n'); err == nil {
		t.Fatal("connection should be closed after a too-long line")
	}
}

func TestQuitSendsByeThenCloses(t *testing.T) {
	addr := startServer(t)
	c, r := login(t, addr, "alice")
	expect(t, c, r, "EVT STATS players=1")
	send(t, c, "QUIT\n")
	expect(t, c, r, "OK bye")
	c.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, err := r.ReadString('\n'); err == nil {
		t.Fatal("connection should be closed after QUIT")
	}
}

func TestDisconnectBroadcastsAndFreesName(t *testing.T) {
	addr := startServer(t)
	a, ra := login(t, addr, "alice")
	expect(t, a, ra, "EVT STATS players=1")

	b, rb := login(t, addr, "bob")
	expect(t, a, ra, "EVT ROOM PRESENCE ENTER bob")
	expect(t, a, ra, "EVT STATS players=2")
	expect(t, b, rb, "EVT STATS players=2")

	b.Close() // abrupt disconnect, no QUIT
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")
	expect(t, a, ra, "EVT STATS players=1")

	_, _ = login(t, addr, "bob") // the name is free again
}

func TestFloodGetsRateLimited(t *testing.T) {
	addr := startServer(t)
	c, r := login(t, addr, "alice")
	expect(t, c, r, "EVT STATS players=1")
	send(t, c, strings.Repeat("WHO\n", 20))
	limited := 0
	for range 20 {
		c.SetReadDeadline(time.Now().Add(2 * time.Second))
		line, _ := r.ReadString('\n')
		if strings.HasPrefix(line, "ERR 429") {
			limited++
		}
	}
	if limited == 0 {
		t.Fatal("expected some ERR 429 RATE_LIMITED replies")
	}
}

// twoPlayers logs in alice and bob and drains the connect-time events.
func twoPlayers(t *testing.T) (a net.Conn, ra *bufio.Reader, b net.Conn, rb *bufio.Reader) {
	t.Helper()
	addr := startServer(t)
	a, ra = login(t, addr, "alice")
	expect(t, a, ra, "EVT STATS players=1")
	b, rb = login(t, addr, "bob")
	expect(t, a, ra, "EVT ROOM PRESENCE ENTER bob")
	expect(t, a, ra, "EVT STATS players=2")
	expect(t, b, rb, "EVT STATS players=2")
	return
}

func TestChatAndWho(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, a, "CHAT GLOBAL hi all\n")
	expect(t, a, ra, "OK") // reply comes before the event
	expect(t, a, ra, "EVT GLOBAL CHAT alice hi all")
	expect(t, b, rb, "EVT GLOBAL CHAT alice hi all")

	send(t, b, "chat room héllo 🙂\n")
	expect(t, b, rb, "OK")
	expect(t, b, rb, "EVT ROOM CHAT bob héllo 🙂")
	expect(t, a, ra, "EVT ROOM CHAT bob héllo 🙂")

	send(t, a, "CHAT GROUP hello\n")
	expect(t, a, ra, "ERR 401 NOT_IN_GROUP")
	send(t, a, "CHAT NOWHERE hello\n")
	expect(t, a, ra, "ERR 400 BAD_REQUEST")
	send(t, a, "CHAT GLOBAL\n")
	expect(t, a, ra, "ERR 400 BAD_REQUEST")

	send(t, a, "WHO\n")
	expect(t, a, ra, "OK players=2")
}

func TestGroupFlow(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, a, "GROUP CREATE\n")
	expect(t, a, ra, "OK group=grp.1")
	send(t, a, "GROUP CREATE\n")
	expect(t, a, ra, "ERR 402 ALREADY_IN_GROUP")

	send(t, b, "GROUP JOIN alice\n") // no invite yet
	expect(t, b, rb, "ERR 407 NOT_INVITED")
	send(t, a, "GROUP INVITE nobody\n")
	expect(t, a, ra, "ERR 404 PLAYER_NOT_FOUND")

	send(t, a, "GROUP INVITE bob\n")
	expect(t, a, ra, "OK")
	expect(t, b, rb, "EVT GROUP INVITE alice")

	send(t, b, "GROUP JOIN carol\n") // wrong leader name
	expect(t, b, rb, "ERR 407 NOT_INVITED")
	send(t, b, "GROUP JOIN alice\n")
	expect(t, b, rb, "OK group=grp.1")
	expect(t, a, ra, "EVT GROUP JOIN bob")
	expect(t, b, rb, "EVT GROUP JOIN bob")

	send(t, b, "CHAT GROUP ready\n")
	expect(t, b, rb, "OK")
	expect(t, b, rb, "EVT GROUP CHAT bob ready")
	expect(t, a, ra, "EVT GROUP CHAT bob ready")

	send(t, b, "GROUP LEAVE\n")
	expect(t, b, rb, "OK")
	expect(t, a, ra, "EVT GROUP LEAVE bob")
	send(t, b, "GROUP LEAVE\n")
	expect(t, b, rb, "ERR 401 NOT_IN_GROUP")
	send(t, b, "GROUP DANCE\n")
	expect(t, b, rb, "ERR 400 BAD_REQUEST")
}

func TestGroupLeaderDisconnectPassesLeadership(t *testing.T) {
	addr := startServer(t)
	a, ra := login(t, addr, "alice")
	expect(t, a, ra, "EVT STATS players=1")
	send(t, a, "GROUP CREATE\n")
	expect(t, a, ra, "OK group=grp.1")

	b, rb := login(t, addr, "bob")
	expect(t, a, ra, "EVT ROOM PRESENCE ENTER bob")
	expect(t, a, ra, "EVT STATS players=2")
	expect(t, b, rb, "EVT STATS players=2")
	c, rc := login(t, addr, "carol")
	expect(t, a, ra, "EVT ROOM PRESENCE ENTER carol")
	expect(t, a, ra, "EVT STATS players=3")
	expect(t, b, rb, "EVT ROOM PRESENCE ENTER carol")
	expect(t, b, rb, "EVT STATS players=3")
	expect(t, c, rc, "EVT STATS players=3")

	send(t, a, "GROUP INVITE bob\n")
	expect(t, a, ra, "OK")
	expect(t, b, rb, "EVT GROUP INVITE alice")
	send(t, b, "GROUP JOIN alice\n")
	expect(t, b, rb, "OK group=grp.1")
	expect(t, a, ra, "EVT GROUP JOIN bob")
	expect(t, b, rb, "EVT GROUP JOIN bob")

	a.Close() // the leader drops without QUIT
	expect(t, b, rb, "EVT GROUP LEAVE alice")
	expect(t, b, rb, "EVT ROOM PRESENCE LEAVE alice")
	expect(t, b, rb, "EVT STATS players=2")

	// Bob is now the leader, so his invite carries his own name.
	send(t, b, "GROUP INVITE carol\n")
	expect(t, b, rb, "OK")
	expect(t, c, rc, "EVT ROOM PRESENCE LEAVE alice")
	expect(t, c, rc, "EVT STATS players=2")
	expect(t, c, rc, "EVT GROUP INVITE bob")
}
