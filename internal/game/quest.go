// Owner: Role B. Rules: PLAN.md section 6, "Quests".

package game

// quest handles QUEST <npc>: accepts, checks or completes the NPC's next quest.
// Replies: OK {"quest_id":..,"description":..,"reward":..,"status":".."},
// ERR 404 NPC_NOT_FOUND, ERR 406 NO_QUEST_AVAILABLE.
func (g *Game) quest(p *Player, args string) Result {
	// TODO(B)
	return Result{Reply: `OK {"quest_id":"road_without_watchers","description":"Destroy the Rotting Sentry.","reward":"+2 damage","status":"available"}`}
}

// quests handles QUESTS. Reply: OK [{"quest_id":..,"status":..,"progress":..}].
func (g *Game) quests(p *Player) Result {
	// TODO(B)
	return Result{Reply: "OK []"}
}
