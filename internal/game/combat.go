// Owner: Role B. Rules: PLAN.md section 6, "Combat".

package game

// attack handles ATTACK <npc>.
// Replies: OK {"attacker_hp":..,"target_hp":..,"damage":..,"status":".."},
// ERR 404 NPC_NOT_FOUND, ERR 405 NPC_NOT_HOSTILE. Event: ROOM COMBAT.
func (g *Game) attack(p *Player, args string) Result {
	// TODO(B)
	return Result{Reply: `OK {"attacker_hp":100,"target_hp":20,"damage":10,"status":"combat"}`}
}

// defend handles DEFEND. Replies: OK {"hp":..,"damage_taken":..,"status":".."},
// ERR 408 NOT_IN_COMBAT.
func (g *Game) defend(p *Player) Result {
	// TODO(B)
	return Result{Reply: `OK {"hp":100,"damage_taken":0,"status":"combat"}`}
}

// flee handles FLEE. Replies: OK room=<room-id>, ERR 408 NOT_IN_COMBAT.
func (g *Game) flee(p *Player) Result {
	// TODO(B)
	return Result{Reply: "OK room=" + p.Room}
}

// status handles STATUS. Reply: OK {"hp":..,"max_hp":..,"status":"healthy|wounded|combat"}.
func (g *Game) status(p *Player) Result {
	// TODO(B)
	return Result{Reply: `OK {"hp":100,"max_hp":100,"status":"healthy"}`}
}
