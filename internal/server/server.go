// Package server handles TCP connections and passes lines to the game.
//
// Owner: Role A. See ROLE_A_PLAN.md steps 2 to 5.
package server

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/protocol"
)

const (
	maxLine        = 1024 // RFC 42TAP recommended line limit
	maxClients     = 100
	cmdLimit       = 10 // commands per second before ERR 429
	rapidConnLimit = 10 // connections per minute per IP before a warning
)

// Server owns all connections. One lock guards its maps and every game call.
type Server struct {
	game *game.Game
	log  *slog.Logger

	mu      sync.Mutex
	clients map[string]*client // connected players by name
	conns   int                // open connections, logged in or not
	recent  map[string]*window // connection attempts per IP
}

// New creates a server for a game.
func New(g *game.Game, log *slog.Logger) *Server {
	return &Server{
		game:    g,
		log:     log,
		clients: map[string]*client{},
		recent:  map[string]*window{},
	}
}

// Run listens on addr and serves until the listener fails.
func (s *Server) Run(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.log.Info("listening", "addr", addr)
	return s.Serve(ln)
}

// Serve accepts connections on ln. Tests pass a listener on a random port.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}

// handleConn owns one connection. When it returns, cleanup has run.
func (s *Server) handleConn(conn net.Conn) {
	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if !s.admit(ip) {
		fmt.Fprintln(conn, protocol.ErrConnectionFailed)
		conn.Close()
		return
	}

	c := &client{conn: conn, ip: ip, out: make(chan string, 256)}
	go c.writeLoop()
	defer s.cleanup(c)

	s.mu.Lock()
	s.reply(c, protocol.Greeting)
	s.mu.Unlock()

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, maxLine+1), maxLine+1) // +1 for the newline
	for sc.Scan() {
		if !s.handleLine(c, sc.Text()) {
			return
		}
	}
	if errors.Is(sc.Err(), bufio.ErrTooLong) {
		s.mu.Lock()
		s.reply(c, protocol.ErrBadRequest)
		s.mu.Unlock()
		s.log.Warn("line_too_long", "ip", ip, "player", c.name)
	}
}

// handleLine runs one command. It returns false when the connection should close.
func (s *Server) handleLine(c *client, line string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, args := protocol.Split(line)
	if cmd == "" {
		return true // ignore blank lines
	}
	s.log.Info("command", "ip", c.ip, "player", c.name, "cmd", cmd, "args", args)

	if n := c.cmds.hit(time.Second); n > cmdLimit {
		if n == cmdLimit+1 {
			s.log.Warn("flood", "ip", c.ip, "player", c.name)
		}
		s.reply(c, protocol.ErrRateLimited)
		return true
	}
	if !protocol.Valid(line) {
		s.reply(c, protocol.ErrBadRequest)
		return true
	}

	var res game.Result
	switch {
	case c.name == "" && cmd == "CONNECT":
		res = s.game.Connect(args)
		if res.Reply == protocol.OKConnected {
			c.name = args
			s.clients[args] = c
		}
	case c.name == "" && cmd == "QUIT":
		res = game.Result{Reply: protocol.OKBye, Close: true}
	case c.name == "":
		res = game.Result{Reply: protocol.ErrNotAuthenticated}
	case cmd == "CONNECT":
		res = game.Result{Reply: protocol.ErrAlreadyConnected}
	default:
		res = s.game.Handle(c.name, cmd, args)
	}

	s.reply(c, res.Reply)
	s.deliver(res.Events)
	return !res.Close
}

// reply logs a reply and queues it. Call it while holding s.mu.
func (s *Server) reply(c *client, line string) {
	if strings.HasPrefix(line, "ERR") {
		s.log.Warn("reply", "ip", c.ip, "player", c.name, "line", line)
	} else {
		s.log.Info("reply", "ip", c.ip, "player", c.name, "line", line)
	}
	c.send(line)
}

// deliver sends each event to the players it names. Call it while holding s.mu.
func (s *Server) deliver(events []game.Event) {
	for _, e := range events {
		s.log.Info("event", "line", e.Line, "recipients", len(e.To))
		for _, name := range e.To {
			if c, ok := s.clients[name]; ok {
				c.send(e.Line)
			}
		}
	}
}

// cleanup runs exactly once per connection, when its read loop ends.
func (s *Server) cleanup(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c.name != "" {
		delete(s.clients, c.name)
		s.deliver(s.game.Disconnect(c.name))
	}
	s.conns--
	close(c.out) // writeLoop sends what is left, then closes the socket
	s.log.Info("disconnect", "ip", c.ip, "player", c.name)
}
