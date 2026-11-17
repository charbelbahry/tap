package gui

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/server"
	"github.com/charbelbahry/tap/internal/world"
)

// startBridge serves the bridge for tcpAddr and returns its WebSocket URL and base URL.
func startBridge(t *testing.T, tcpAddr string) (wsURL, baseURL string) {
	t.Helper()
	h, err := handler(tcpAddr)
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws", ts.URL
}

func dialWS(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ws.Close() })
	return ws
}

func wsExpect(t *testing.T, ws *websocket.Conn, want string) {
	t.Helper()
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := ws.ReadMessage()
	if err != nil || string(msg) != want {
		t.Fatalf("got %q (err %v), want %q", msg, err, want)
	}
}

// lineServer accepts TCP connections and hands each one to onConn.
func lineServer(t *testing.T, onConn func(net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go onConn(c)
		}
	}()
	return ln.Addr().String()
}

func TestServesPage(t *testing.T) {
	_, base := startBridge(t, "127.0.0.1:1")
	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("GET / = %d", resp.StatusCode)
	}
}

func TestRelaysLinesBothWays(t *testing.T) {
	// The fake server greets, then replies "got <line>" to each line, then pushes an event.
	addr := lineServer(t, func(c net.Conn) {
		defer c.Close()
		c.Write([]byte("OK hello proto=1\n"))
		sc := bufio.NewScanner(c)
		for sc.Scan() {
			c.Write([]byte("got " + sc.Text() + "\nEVT STATS players=1\n"))
		}
	})
	wsURL, _ := startBridge(t, addr)
	ws := dialWS(t, wsURL)

	wsExpect(t, ws, "OK hello proto=1")
	ws.WriteMessage(websocket.TextMessage, []byte("CONNECT alice"))
	wsExpect(t, ws, "got CONNECT alice") // one WebSocket message per line
	wsExpect(t, ws, "EVT STATS players=1")
}

func TestEachTabGetsItsOwnTCPConnection(t *testing.T) {
	conns := make(chan struct{}, 2)
	addr := lineServer(t, func(c net.Conn) {
		defer c.Close()
		conns <- struct{}{}
		c.Write([]byte("OK hello proto=1\n"))
		bufio.NewReader(c).ReadByte() // hold the connection open
	})
	wsURL, _ := startBridge(t, addr)
	wsExpect(t, dialWS(t, wsURL), "OK hello proto=1")
	wsExpect(t, dialWS(t, wsURL), "OK hello proto=1")
	<-conns
	<-conns
}

func TestServerHangupClosesWebSocket(t *testing.T) {
	addr := lineServer(t, func(c net.Conn) {
		c.Write([]byte("OK bye\n"))
		c.Close()
	})
	wsURL, _ := startBridge(t, addr)
	ws := dialWS(t, wsURL)
	wsExpect(t, ws, "OK bye")
	ws.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := ws.ReadMessage(); err == nil {
		t.Fatal("WebSocket should close after the server hangs up")
	}
}

func TestBrowserCloseClosesTCP(t *testing.T) {
	closed := make(chan struct{})
	addr := lineServer(t, func(c net.Conn) {
		bufio.NewReader(c).ReadByte() // returns when the bridge closes the connection
		close(closed)
	})
	wsURL, _ := startBridge(t, addr)
	dialWS(t, wsURL).Close()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("TCP connection stayed open after the browser left")
	}
}

func TestUnreachableServer(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	dead := ln.Addr().String()
	ln.Close() // nothing listens here any more
	wsURL, _ := startBridge(t, dead)
	wsExpect(t, dialWS(t, wsURL), "ERR 900 CONNECTION_FAILED")
}

// TestWithRealServer sends a browser's commands through the bridge to our game server.
func TestWithRealServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	log := slog.New(slog.DiscardHandler)
	go server.New(game.New(&world.World{}, log), log).Serve(ln)

	wsURL, _ := startBridge(t, ln.Addr().String())
	alice, bob := dialWS(t, wsURL), dialWS(t, wsURL)

	wsExpect(t, alice, "OK hello proto=1")
	alice.WriteMessage(websocket.TextMessage, []byte("CONNECT alice"))
	wsExpect(t, alice, "OK connected")
	wsExpect(t, alice, "EVT STATS players=1")

	wsExpect(t, bob, "OK hello proto=1")
	bob.WriteMessage(websocket.TextMessage, []byte("CONNECT bob"))
	wsExpect(t, bob, "OK connected")
	wsExpect(t, alice, "EVT ROOM PRESENCE ENTER bob")
	wsExpect(t, alice, "EVT STATS players=2")
	wsExpect(t, bob, "EVT STATS players=2")

	bob.WriteMessage(websocket.TextMessage, []byte("CHAT GLOBAL hi from the browser"))
	wsExpect(t, bob, "OK")
	wsExpect(t, bob, "EVT GLOBAL CHAT bob hi from the browser")
	wsExpect(t, alice, "EVT GLOBAL CHAT bob hi from the browser")

	bob.Close() // closing the tab acts like a disconnect
	wsExpect(t, alice, "EVT ROOM PRESENCE LEAVE bob")
	wsExpect(t, alice, "EVT STATS players=1")
}
