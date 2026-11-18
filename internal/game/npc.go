// Owner: Role B.

package game

import (
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// matchNPC reports whether query matches an NPC's full ID, short ID, or display name.
func matchNPC(query, npcID string, npc world.NPC) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return false
	}
	if strings.EqualFold(npcID, q) {
		return true
	}
	if strings.EqualFold(strings.TrimPrefix(npcID, "npc."), q) {
		return true
	}
	if npc.Name != "" && strings.EqualFold(npc.Name, q) {
		return true
	}
	return false
}

// findNPCInRoom searches NPCs in roomID matching the query.
func (g *Game) findNPCInRoom(roomID, query string) (string, world.NPC) {
	for _, id := range g.world.Map.Rooms[roomID].NPCs {
		npc := g.world.NPCs[id]
		if matchNPC(query, id, npc) {
			return id, npc
		}
	}
	return "", world.NPC{}
}

// talk handles TALK <npc>.
// Replies: OK <dialogue> (RFC format), ERR 404 NPC_NOT_FOUND, ERR 400 BAD_REQUEST.
func (g *Game) talk(p *Player, args string) Result {
	if p == nil || strings.TrimSpace(args) == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}

	id, npc := g.findNPCInRoom(p.Room, args)
	if id == "" {
		return Result{Reply: protocol.ErrNPCNotFound}
	}
	if len(npc.Dialogue) == 0 {
		return Result{Reply: "OK ..."}
	}

	if g.dialogueIdx == nil {
		g.dialogueIdx = make(map[string]int)
	}
	idx := g.dialogueIdx[id] % len(npc.Dialogue)
	g.dialogueIdx[id] = (idx + 1) % len(npc.Dialogue)

	return Result{Reply: "OK " + npc.Dialogue[idx]}
}
