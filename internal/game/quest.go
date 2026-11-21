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

// rewardStr formats a human-readable summary of quest rewards.
func (g *Game) rewardStr(r world.Reward) string {
	var parts []string
	if r.DamageBonus > 0 {
		parts = append(parts, fmt.Sprintf("+%d damage", r.DamageBonus))
	}
	if r.MaxHPBonus > 0 {
		parts = append(parts, fmt.Sprintf("+%d max HP", r.MaxHPBonus))
	}
	if r.Item != "" {
		name := r.Item
		if it, ok := g.world.Items[r.Item]; ok && it.Name != "" {
			name = it.Name
		}
		parts = append(parts, name)
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

// findGiverQuest selects the active or next available quest for p from giverID.
func (g *Game) findGiverQuest(p *Player, giverID string) (string, world.Quest) {
	var candidateIDs []string
	for id, q := range g.world.Quests {
		if q.Giver == giverID {
			candidateIDs = append(candidateIDs, id)
		}
	}
	sort.Strings(candidateIDs)

	for _, id := range candidateIDs {
		q := g.world.Quests[id]
		prog := p.Quests[id]
		if prog != nil && prog.Status == "completed" {
			continue
		}
		if q.Needs != "" {
			req := p.Quests[q.Needs]
			if req == nil || req.Status != "completed" {
				continue
			}
		}
		return id, q
	}
	return "", world.Quest{}
}

// advanceKillQuest updates active quests with kill objectives matching npcID.
func (g *Game) advanceKillQuest(p *Player, npcID string) {
	if p == nil || p.Quests == nil {
		return
	}
	for id, prog := range p.Quests {
		if prog.Status == "active" {
			q := g.world.Quests[id]
			if q.Objective.Kill == npcID {
				prog.Killed = true
			}
		}
	}
}

// quest handles QUEST <npc>: accepts, checks or completes the NPC's next quest.
// Replies: OK {"quest_id":..,"description":..,"reward":..,"status":".."},
// ERR 404 NPC_NOT_FOUND, ERR 406 NO_QUEST_AVAILABLE.
func (g *Game) quest(p *Player, args string) Result {
	if p == nil || strings.TrimSpace(args) == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}

	npcID, _ := g.findNPCInRoom(p.Room, args)
	if npcID == "" {
		return Result{Reply: protocol.ErrNPCNotFound}
	}

	if p.Quests == nil {
		p.Quests = make(map[string]*QuestProgress)
	}

	qID, q := g.findGiverQuest(p, npcID)
	if qID == "" {
		return Result{Reply: protocol.ErrNoQuest}
	}

	prog := p.Quests[qID]
	reward := g.rewardStr(q.Reward)

	// First call: accept quest and return status available
	if prog == nil {
		p.Quests[qID] = &QuestProgress{Status: "active"}
		resp, _ := json.Marshal(map[string]any{
			"quest_id":    qID,
			"description": q.Description,
			"reward":      reward,
			"status":      "available",
		})
		return Result{Reply: "OK " + string(resp)}
	}

	// Check if objective is met
	met := false
	if q.Objective.Kill != "" && prog.Killed {
		met = true
	} else if q.Objective.Deliver != "" && slices.Contains(p.Inventory, q.Objective.Deliver) {
		met = true
	}

	if !met {
		resp, _ := json.Marshal(map[string]any{
			"quest_id":    qID,
			"description": q.Description,
			"reward":      reward,
			"status":      "active",
		})
		return Result{Reply: "OK " + string(resp)}
	}

	// Complete quest and grant rewards
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

	prog.Status = "completed"
	resp, _ := json.Marshal(map[string]any{
		"quest_id":    qID,
		"description": q.Description,
		"reward":      reward,
		"status":      "completed",
	})
	return Result{Reply: "OK " + string(resp)}
}

// quests handles QUESTS. Reply: OK [{"quest_id":..,"status":..,"progress":..}].
func (g *Game) quests(p *Player) Result {
	if p == nil {
		return Result{Reply: protocol.ErrBadRequest}
	}
	if p.Quests == nil || len(p.Quests) == 0 {
		return Result{Reply: "OK []"}
	}

	var qIDs []string
	for id := range p.Quests {
		qIDs = append(qIDs, id)
	}
	sort.Strings(qIDs)

	var list []map[string]any
	for _, id := range qIDs {
		prog := p.Quests[id]
		q := g.world.Quests[id]
		progress := ""
		if prog.Status == "active" {
			if q.Objective.Kill != "" {
				if prog.Killed {
					progress = "1/1"
				} else {
					progress = "0/1"
				}
			} else if q.Objective.Deliver != "" {
				if slices.Contains(p.Inventory, q.Objective.Deliver) {
					progress = "1/1"
				} else {
					progress = "0/1"
				}
			}
		}
		entry := map[string]any{
			"quest_id": id,
			"status":   prog.Status,
		}
		if progress != "" {
			entry["progress"] = progress
		}
		list = append(list, entry)
	}

	data, _ := json.Marshal(list)
	return Result{Reply: "OK " + string(data)}
}
