// Owner: Role A. See ROLE_A_PLAN.md step 6.

package game

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charbelbahry/tap/internal/protocol"
)

// Connect validates a name and adds a new player.
// Replies: OK connected, ERR 201 NAME_IN_USE, ERR 400 BAD_REQUEST.
// Events: ROOM PRESENCE ENTER to others in the room, STATS to everyone.
func (g *Game) Connect(name string) Result {
	if !validName(name) {
		return Result{Reply: protocol.ErrBadRequest}
	}
	if _, taken := g.players[name]; taken {
		return Result{Reply: protocol.ErrNameInUse}
	}
	p := g.newPlayer(name)
	g.players[name] = p
	return Result{
		Reply: protocol.OKConnected,
		Events: []Event{
			{To: g.othersInRoom(p), Line: "EVT ROOM PRESENCE ENTER " + name},
			g.statsEvent(),
		},
	}
}

// Disconnect removes the player first, then builds the events others should see:
// GROUP LEAVE, ROOM PRESENCE LEAVE and STATS.
func (g *Game) Disconnect(name string) []Event {
	p := g.players[name]
	delete(g.players, name)
	g.dropAllItems(p)

	events := g.leaveGroup(p)
	return append(events,
		Event{To: g.namesInRoom(p.Room), Line: "EVT ROOM PRESENCE LEAVE " + name},
		g.statsEvent(),
	)
}

// statsEvent tells everyone the current player count.
func (g *Game) statsEvent() Event {
	return Event{To: g.allNames(), Line: fmt.Sprintf("EVT STATS players=%d", len(g.players))}
}

// validName allows 1 to 32 characters, Unicode included, with no whitespace.
// Control characters were already rejected by protocol.Valid.
func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n >= 1 && n <= 32 && !strings.ContainsFunc(name, unicode.IsSpace)
}
