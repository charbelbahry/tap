// Package game holds all game rules. It never touches the network.
//
// It is not safe for concurrent use: the server calls it under one lock,
// so nothing in this package needs a mutex.
//
// Owner: shared (written together on Day 1). Each other file in this
// package has one owner, named at its top.
package game

import (
	"log/slog"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// Event is one line to push to a set of players.
type Event struct {
	To   []string // player names
	Line string   // full line, such as "EVT ROOM CHAT alice hello"
}

// Result is what a command produces. The server sends Reply first, then Events.
type Result struct {
	Reply  string
	Events []Event
	Close  bool // true after QUIT
}

// Player is one connected player.
type Player struct {
	Name   string
	Room   string
	Group  *Group // Role A
	Invite *Group // Role A: last group this player was invited to

	// Role B adds HP, MaxHP, inventory, quest progress and so on.
}

// Group members are kept in join order. Members[0] is the leader.
type Group struct {
	ID      string
	Members []string
}

// Game is the whole world state.
type Game struct {
	world     *world.World
	log       *slog.Logger
	players   map[string]*Player
	nextGroup int

	// Role B adds item locations, NPC state and so on.
}

// New creates a game from loaded world data.
func New(w *world.World, log *slog.Logger) *Game {
	return &Game{world: w, log: log, players: map[string]*Player{}}
}

// Handle runs one command for a connected player.
// CONNECT is handled separately by Connect, before a player exists.
func (g *Game) Handle(name, cmd, args string) Result {
	p := g.players[name]
	switch cmd {
	// Role A
	case "QUIT":
		return Result{Reply: protocol.OKBye, Close: true}
	case "CHAT":
		return g.chat(p, args)
	case "WHO":
		return g.who()
	case "GROUP":
		return g.group(p, args)

	// Role B
	case "LOOK":
		return g.look(p)
	case "MOVE":
		return g.move(p, args)
	case "TAKE":
		return g.take(p, args)
	case "DROP":
		return g.drop(p, args)
	case "INVENTORY":
		return g.inventory(p)
	case "TALK":
		return g.talk(p, args)
	case "ATTACK":
		return g.attack(p, args)
	case "DEFEND":
		return g.defend(p)
	case "FLEE":
		return g.flee(p)
	case "STATUS":
		return g.status(p)
	case "QUEST":
		return g.quest(p, args)
	case "QUESTS":
		return g.quests(p)

	default:
		return Result{Reply: protocol.ErrBadRequest}
	}
}

// allNames returns every connected player's name.
func (g *Game) allNames() []string {
	names := make([]string, 0, len(g.players))
	for name := range g.players {
		names = append(names, name)
	}
	return names
}

// namesInRoom returns the names of everyone in a room.
func (g *Game) namesInRoom(room string) []string {
	var names []string
	for _, p := range g.players {
		if p.Room == room {
			names = append(names, p.Name)
		}
	}
	return names
}

// othersInRoom returns everyone in p's room except p.
func (g *Game) othersInRoom(p *Player) []string {
	var names []string
	for _, other := range g.players {
		if other.Room == p.Room && other != p {
			names = append(names, other.Name)
		}
	}
	return names
}
