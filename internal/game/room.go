// Owner: Role B.

package game

// newPlayer creates a player in the start room with full health.
func (g *Game) newPlayer(name string) *Player {
	// TODO(B): use the world's start room and set HP and MaxHP.
	return &Player{Name: name, Room: "loc.votive_chapel"}
}

// look handles LOOK. Reply: OK <room JSON> (RFC 42TAP section 5.1.2).
func (g *Game) look(p *Player) Result {
	// TODO(B): build the JSON from the real room state.
	return Result{Reply: `OK {"room":{"id":"loc.votive_chapel","name":"Votive Chapel","description":"Stub room.","exits":{}},"players":[],"items":[],"npcs":[]}`}
}

// move handles MOVE <direction>.
// Replies: OK room=<room-id>, ERR 301 NO_EXIT.
// Events: ROOM PRESENCE LEAVE to the old room, ROOM PRESENCE ENTER to the new one.
// Moving during combat gives the enemy a free hit.
func (g *Game) move(p *Player, args string) Result {
	// TODO(B): check the exit, move the player, build the events.
	return Result{Reply: "OK room=" + p.Room}
}
