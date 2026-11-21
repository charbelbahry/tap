// Owner: Role B. Rules: PLAN.md section 6, "Quests".

package game

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// rewardBonusParts formats stat bonus strings for quest rewards.
func (g *Game) rewardBonusParts(r world.Reward) []string {
	var parts []string
	if r.DamageBonus > 0 {
		parts = append(parts, fmt.Sprintf("+%d damage", r.DamageBonus))
	}
	if r.MaxHPBonus > 0 {
		parts = append(parts, fmt.Sprintf("+%d max HP", r.MaxHPBonus))
	}
	return parts
}

// rewardItemName resolves the display name for a reward item.
func (g *Game) rewardItemName(itemID string) string {
	if it, ok := g.world.Items[itemID]; ok && it.Name != "" {
		return it.Name
	}
	return itemID
}

// rewardStr formats a human-readable summary of quest rewards.
func (g *Game) rewardStr(r world.Reward) string {
	parts := g.rewardBonusParts(r)
	if r.Item != "" {
		parts = append(parts, g.rewardItemName(r.Item))
	}
	if r.Title != "" {
		parts = append(parts, fmt.Sprintf("title %q", r.Title))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// findStartingRoomForItem finds which room initially spawns itemID.
func (g *Game) findStartingRoomForItem(itemID string) string {
	for id, r := range g.world.Map.Rooms {
		if slices.Contains(r.Items, itemID) {
			return id
		}
	}
	return ""
}

// itemCarriedByAnyone reports whether any player is currently holding itemID.
func (g *Game) itemCarriedByAnyone(itemID string) bool {
	for _, p := range g.players {
		if slices.Contains(p.Inventory, itemID) {
			return true
		}
	}
	return false
}

// giverQuestIDs returns sorted list of quest IDs offered by giverID.
func (g *Game) giverQuestIDs(giverID string) []string {
	var ids []string
	for id, q := range g.world.Quests {
		if q.Giver == giverID {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// isQuestAvailable reports whether player can accept or work on quest qID.
func (g *Game) isQuestAvailable(p *Player, qID string, q world.Quest) bool {
	prog := p.Quests[qID]
	if prog != nil && prog.Status == "completed" {
		return false
	}
	if q.Needs != "" {
		req := p.Quests[q.Needs]
		return req != nil && req.Status == "completed"
	}
	return true
}

// findGiverQuest selects the active or next available quest for p from giverID.
func (g *Game) findGiverQuest(p *Player, giverID string) (string, world.Quest) {
	for _, id := range g.giverQuestIDs(giverID) {
		q := g.world.Quests[id]
		if g.isQuestAvailable(p, id, q) {
			return id, q
		}
	}
	return "", world.Quest{}
}

// advanceKillQuest updates active quests with kill objectives matching npcID.
func (g *Game) advanceKillQuest(p *Player, npcID string) {
	if p == nil || p.Quests == nil {
		return
	}
	for id, prog := range p.Quests {
		if prog.Status == "active" && g.world.Quests[id].Objective.Kill == npcID {
			prog.Killed = true
		}
	}
}

// isObjectiveMet checks if the player fulfilled the quest requirements.
func (g *Game) isObjectiveMet(p *Player, q world.Quest, prog *QuestProgress) bool {
	if q.Objective.Kill != "" && prog.Killed {
		return true
	}
	if q.Objective.Deliver != "" && slices.Contains(p.Inventory, q.Objective.Deliver) {
		return true
	}
	return false
}

// applyQuestRewards grants items, stats, and handles quest item cleanup.
func (g *Game) applyQuestRewards(p *Player, q world.Quest) {
	if q.Objective.Deliver != "" {
		p.Inventory = slices.DeleteFunc(p.Inventory, func(id string) bool { return id == q.Objective.Deliver })
		spawn := g.findStartingRoomForItem(q.Objective.Deliver)
		if spawn != "" {
			g.roomItems[spawn] = append(g.roomItems[spawn], q.Objective.Deliver)
		}
	}

	p.DamageBonus += q.Reward.DamageBonus
	p.MaxHP += q.Reward.MaxHPBonus
	p.HP += q.Reward.MaxHPBonus
	if q.Reward.Title != "" {
		p.Title = q.Reward.Title
	}
	if q.Reward.Item != "" && !g.itemCarriedByAnyone(q.Reward.Item) {
		p.Inventory = append(p.Inventory, q.Reward.Item)
		if it, ok := g.world.Items[q.Reward.Item]; ok {
			p.DamageBonus += it.DamageBonus
			p.MaxHP += it.MaxHPBonus
		}
	}
}

// questResult formats standard QUEST JSON reply.
func questResult(id, desc, reward, status string) Result {
	resp, _ := json.Marshal(map[string]any{
		"quest_id":    id,
		"description": desc,
		"reward":      reward,
		"status":      status,
	})
	return Result{Reply: "OK " + string(resp)}
}

// resolveGiverQuest validates player and NPC, ensuring player quests map is initialized.
func (g *Game) resolveGiverQuest(p *Player, args string) (string, world.Quest, *Result) {
	if p == nil || strings.TrimSpace(args) == "" {
		res := Result{Reply: protocol.ErrBadRequest}
		return "", world.Quest{}, &res
	}
	npcID, _ := g.findNPCInRoom(p.Room, args)
	if npcID == "" {
		res := Result{Reply: protocol.ErrNPCNotFound}
		return "", world.Quest{}, &res
	}
	if p.Quests == nil {
		p.Quests = make(map[string]*QuestProgress)
	}
	qID, q := g.findGiverQuest(p, npcID)
	if qID == "" {
		res := Result{Reply: protocol.ErrNoQuest}
		return "", world.Quest{}, &res
	}
	return qID, q, nil
}

// quest handles QUEST <npc>: accepts, checks or completes the NPC's next quest.
// Replies: OK {"quest_id":..,"description":..,"reward":..,"status":".."},
// ERR 404 NPC_NOT_FOUND, ERR 406 NO_QUEST_AVAILABLE.
func (g *Game) quest(p *Player, args string) Result {
	qID, q, errRes := g.resolveGiverQuest(p, args)
	if errRes != nil {
		return *errRes
	}

	prog := p.Quests[qID]
	reward := g.rewardStr(q.Reward)

	if prog == nil {
		p.Quests[qID] = &QuestProgress{Status: "active"}
		return questResult(qID, q.Description, reward, "available")
	}

	if !g.isObjectiveMet(p, q, prog) {
		return questResult(qID, q.Description, reward, "active")
	}

	g.applyQuestRewards(p, q)
	prog.Status = "completed"
	return questResult(qID, q.Description, reward, "completed")
}

// questProgressStr computes 0/1 or 1/1 progress string for active quests.
func (g *Game) questProgressStr(p *Player, q world.Quest, prog *QuestProgress) string {
	if prog.Status != "active" {
		return ""
	}
	if q.Objective.Kill != "" {
		if prog.Killed {
			return "1/1"
		}
		return "0/1"
	}
	if q.Objective.Deliver != "" {
		if slices.Contains(p.Inventory, q.Objective.Deliver) {
			return "1/1"
		}
		return "0/1"
	}
	return ""
}

// quests handles QUESTS. Reply: OK [{"quest_id":..,"status":..,"progress":..}].
func (g *Game) quests(p *Player) Result {
	if p == nil {
		return Result{Reply: protocol.ErrBadRequest}
	}
	if len(p.Quests) == 0 {
		return Result{Reply: "OK []"}
	}

	var qIDs []string
	for id := range p.Quests {
		qIDs = append(qIDs, id)
	}
	sort.Strings(qIDs)

	var list []map[string]any
	for _, id := range qIDs {
		entry := map[string]any{
			"quest_id": id,
			"status":   p.Quests[id].Status,
		}
		if pr := g.questProgressStr(p, g.world.Quests[id], p.Quests[id]); pr != "" {
			entry["progress"] = pr
		}
		list = append(list, entry)
	}

	data, _ := json.Marshal(list)
	return Result{Reply: "OK " + string(data)}
}
