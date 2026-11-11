// Owner: Role A. See ROLE_A_PLAN.md step 6.

package game

import "github.com/charbelbahry/tap/internal/protocol"

// Connect validates a name and adds a new player.
// Replies: OK connected, ERR 201 NAME_IN_USE, ERR 400 BAD_REQUEST.
// Events: ROOM PRESENCE ENTER to others in the room, STATS to everyone.
func (g *Game) Connect(name string) Result {
	// TODO(A): validate the name, reject duplicates, build the events.
	g.players[name] = g.newPlayer(name)
	return Result{Reply: protocol.OKConnected}
}

// Disconnect removes the player first, then builds the events others should see:
// GROUP LEAVE, ROOM PRESENCE LEAVE and STATS.
func (g *Game) Disconnect(name string) []Event {
	// TODO(A): drop items (dropAllItems), leave the group, build the events.
	delete(g.players, name)
	return nil
}
