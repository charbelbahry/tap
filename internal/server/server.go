// Package server handles TCP connections and passes lines to the game.
//
// Owner: Role A. See ROLE_A_PLAN.md steps 2 to 5.
package server

import (
	"errors"
	"log/slog"
	"net"
	"sync"

	"github.com/charbelbahry/tap/internal/game"
)

// Server owns all connections. One lock guards its maps and every game call.
type Server struct {
	game *game.Game
	log  *slog.Logger

	mu      sync.Mutex
	clients map[string]*client // connected players by name
	conns   int                // open connections, logged in or not
}

// New creates a server for a game.
func New(g *game.Game, log *slog.Logger) *Server {
	return &Server{game: g, log: log, clients: map[string]*client{}}
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
	// TODO(A): accept loop, one goroutine per connection (handleConn).
	ln.Close()
	return errors.New("server.Serve is not implemented yet (Role A)")
}
