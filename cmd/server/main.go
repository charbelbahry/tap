// Command server runs the TAP game server.
//
// Owner: Role A.
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/server"
	"github.com/charbelbahry/tap/internal/world"
)

func main() {
	port := flag.Int("port", 4242, "TCP port to listen on")
	worldPath := flag.String("world", "data/world.yaml", "world data file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w, err := world.Load(*worldPath)
	if err != nil {
		log.Error("world_invalid", "err", err)
		os.Exit(1)
	}

	s := server.New(game.New(w, log), log)
	if err := s.Run(fmt.Sprintf(":%d", *port)); err != nil {
		log.Error("server_stopped", "err", err)
		os.Exit(1)
	}
}
