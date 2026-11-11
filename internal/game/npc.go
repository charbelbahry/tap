// Owner: Role B.

package game

// talk handles TALK <npc>.
// Replies: OK <dialogue> (RFC format), ERR 404 NPC_NOT_FOUND.
func (g *Game) talk(p *Player, args string) Result {
	// TODO(B): find the NPC in the room, return its next dialogue line.
	return Result{Reply: "OK The flame holds you here, Vigilant."}
}
