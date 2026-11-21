// Owner: Role B. Rules: PLAN.md section 6, "Combat".

package game

import (
	"encoding/json"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// isNPCAlive reports whether the NPC is currently alive and revives it if 45s elapsed.
func (g *Game) isNPCAlive(id string) bool {
	npc, ok := g.world.NPCs[id]
	if !ok {
		return false
	}
	if npc.Role != "enemy" {
		return true
	}
	respawn, dead := g.enemyRespawnAt[id]
	if !dead {
		return true
	}
	if time.Now().After(respawn) {
		delete(g.enemyRespawnAt, id)
		g.enemyHP[id] = npc.Stats.HP
		return true
	}
	return false
}

// calcDamage calculates random damage in [min, max] plus bonus.
func calcDamage(min, max, bonus int) int {
	dmg := min
	if max > min {
		dmg += rand.Intn(max - min + 1)
	}
	return dmg + bonus
}

// roomHasItem reports whether any player in roomID carries itemID.
func (g *Game) roomHasItem(roomID, itemID string) bool {
	for _, p := range g.players {
		if p.Room == roomID {
			for _, carried := range p.Inventory {
				if carried == itemID {
					return true
				}
			}
		}
	}
	return false
}

// handlePlayerDeath moves player to safe room with 50 HP and clears combat.
func (g *Game) handlePlayerDeath(p *Player) {
	p.HP = 50
	p.Room = g.world.Map.SafeRoom
	p.InCombat = false
	p.CombatTarget = ""
}

// combatFreeHit applies one enemy counterattack to p.
func (g *Game) combatFreeHit(p *Player) {
	if p.CombatTarget == "" {
		return
	}
	npc := g.world.NPCs[p.CombatTarget]
	p.HP -= calcDamage(npc.Stats.DamageMin, npc.Stats.DamageMax, 0)
}

// creditKill awards kill credit to p and all group members in the same room.
func (g *Game) creditKill(p *Player, npcID string) {
	for _, m := range g.groupMembersInRoom(p) {
		g.recordQuestKill(m, npcID)
	}
}

// recordQuestKill is hooked into quest progression for kill objectives.
func (g *Game) recordQuestKill(p *Player, npcID string) {
	g.advanceKillQuest(p, npcID)
}

// resolveAttackTarget validates and returns the target NPC for attack.
func (g *Game) resolveAttackTarget(p *Player, args string) (string, world.NPC, *Result) {
	if p == nil || strings.TrimSpace(args) == "" {
		res := Result{Reply: protocol.ErrBadRequest}
		return "", world.NPC{}, &res
	}
	npcID, npc := g.findNPCInRoom(p.Room, args)
	if npcID == "" || !g.isNPCAlive(npcID) {
		res := Result{Reply: protocol.ErrNPCNotFound}
		return "", world.NPC{}, &res
	}
	if npc.Role != "enemy" {
		res := Result{Reply: protocol.ErrNPCNotHostile}
		return "", world.NPC{}, &res
	}
	return npcID, npc, nil
}

// computePlayerDamage calculates player strike damage against target.
func (g *Game) computePlayerDamage(p *Player, npcID string) int {
	if npcID == "npc.baal" && !g.roomHasItem(p.Room, "item.unspoken_question") {
		return 0
	}
	return 8 + rand.Intn(5) + p.DamageBonus
}

// handleVictory processes enemy defeat, rewards, and notification.
func (g *Game) handleVictory(p *Player, npcID string, playerDmg int) Result {
	g.enemyHP[npcID] = 0
	g.enemyRespawnAt[npcID] = time.Now().Add(45 * time.Second)
	p.InCombat = false
	p.CombatTarget = ""
	g.creditKill(p, npcID)

	resp, _ := json.Marshal(map[string]any{
		"attacker_hp": p.HP,
		"target_hp":   0,
		"damage":      playerDmg,
		"status":      "victory",
	})
	evt := Event{
		To:   g.namesInRoom(p.Room),
		Line: "EVT ROOM COMBAT " + p.Name + " " + npcID + " " + string(resp),
	}
	return Result{Reply: "OK " + string(resp), Events: []Event{evt}}
}

// handlePlayerDefeat processes player death in combat.
func (g *Game) handlePlayerDefeat(p *Player, npcID string, playerDmg int) Result {
	oldRoom := p.Room
	g.handlePlayerDeath(p)
	resp, _ := json.Marshal(map[string]any{
		"attacker_hp": 50,
		"target_hp":   g.enemyHP[npcID],
		"damage":      playerDmg,
		"status":      "defeat",
	})
	events := append([]Event{{
		To:   g.namesInRoom(oldRoom),
		Line: "EVT ROOM COMBAT " + p.Name + " " + npcID + " " + string(resp),
	}}, g.presenceEvents(oldRoom, p)...)
	return Result{Reply: "OK " + string(resp), Events: events}
}

// attack handles ATTACK <npc>.
// Replies: OK {"attacker_hp":..,"target_hp":..,"damage":..,"status":".."},
// ERR 404 NPC_NOT_FOUND, ERR 405 NPC_NOT_HOSTILE. Event: ROOM COMBAT.
func (g *Game) attack(p *Player, args string) Result {
	npcID, npc, errRes := g.resolveAttackTarget(p, args)
	if errRes != nil {
		return *errRes
	}

	if g.enemyHP[npcID] == 0 {
		g.enemyHP[npcID] = npc.Stats.HP
	}

	playerDmg := g.computePlayerDamage(p, npcID)
	g.enemyHP[npcID] -= playerDmg
	if g.enemyHP[npcID] <= 0 {
		return g.handleVictory(p, npcID, playerDmg)
	}

	p.InCombat = true
	p.CombatTarget = npcID
	p.HP -= calcDamage(npc.Stats.DamageMin, npc.Stats.DamageMax, 0)
	if p.HP <= 0 {
		return g.handlePlayerDefeat(p, npcID, playerDmg)
	}

	resp, _ := json.Marshal(map[string]any{
		"attacker_hp": p.HP,
		"target_hp":   g.enemyHP[npcID],
		"damage":      playerDmg,
		"status":      "combat",
	})
	evt := Event{
		To:   g.othersInRoom(p),
		Line: "EVT ROOM COMBAT " + p.Name + " " + npcID + " " + string(resp),
	}
	return Result{Reply: "OK " + string(resp), Events: []Event{evt}}
}

// defend handles DEFEND. Replies: OK {"hp":..,"damage_taken":..,"status":".."},
// ERR 408 NOT_IN_COMBAT.
func (g *Game) defend(p *Player) Result {
	if p == nil || !p.InCombat || p.CombatTarget == "" {
		return Result{Reply: protocol.ErrNotInCombat}
	}

	npc := g.world.NPCs[p.CombatTarget]
	taken := calcDamage(npc.Stats.DamageMin, npc.Stats.DamageMax, 0) / 2
	p.HP -= taken

	if p.HP <= 0 {
		oldRoom := p.Room
		g.handlePlayerDeath(p)
		resp, _ := json.Marshal(map[string]any{
			"hp":           50,
			"damage_taken": taken,
			"status":       "defeat",
		})
		events := append([]Event{{
			To:   g.namesInRoom(oldRoom),
			Line: "EVT ROOM COMBAT " + p.Name + " " + p.CombatTarget + " " + string(resp),
		}}, g.presenceEvents(oldRoom, p)...)
		return Result{Reply: "OK " + string(resp), Events: events}
	}

	resp, _ := json.Marshal(map[string]any{
		"hp":           p.HP,
		"damage_taken": taken,
		"status":       "combat",
	})
	evt := Event{
		To:   g.othersInRoom(p),
		Line: "EVT ROOM COMBAT " + p.Name + " " + p.CombatTarget + " " + string(resp),
	}
	return Result{Reply: "OK " + string(resp), Events: []Event{evt}}
}

// randomExit selects an exit room for fleeing.
func (g *Game) randomExit(roomID string) string {
	r := g.world.Map.Rooms[roomID]
	var exits []string
	for _, target := range r.Exits {
		exits = append(exits, target)
	}
	if len(exits) == 0 {
		return roomID
	}
	sort.Strings(exits)
	return exits[rand.Intn(len(exits))]
}

// flee handles FLEE. Replies: OK room=<room-id>, ERR 408 NOT_IN_COMBAT.
func (g *Game) flee(p *Player) Result {
	if p == nil || !p.InCombat {
		return Result{Reply: protocol.ErrNotInCombat}
	}

	targetRoom := g.randomExit(p.Room)
	if rand.Intn(2) == 1 {
		g.combatFreeHit(p)
		if p.HP <= 0 {
			oldRoom := p.Room
			g.handlePlayerDeath(p)
			return Result{Reply: "OK room=" + p.Room, Events: g.presenceEvents(oldRoom, p)}
		}
	}

	oldRoom := p.Room
	p.Room = targetRoom
	p.InCombat = false
	p.CombatTarget = ""

	if targetRoom == g.world.Map.SafeRoom {
		p.HP = p.MaxHP
	}

	return Result{Reply: "OK room=" + targetRoom, Events: g.presenceEvents(oldRoom, p)}
}

// status handles STATUS. Reply: OK {"hp":..,"max_hp":..,"status":"healthy|wounded|combat"}.
func (g *Game) status(p *Player) Result {
	if p == nil {
		return Result{Reply: protocol.ErrBadRequest}
	}
	st := "healthy"
	if p.InCombat {
		st = "combat"
	} else if p.HP*100 < p.MaxHP*60 {
		st = "wounded"
	}

	resp, _ := json.Marshal(map[string]any{
		"hp":     p.HP,
		"max_hp": p.MaxHP,
		"status": st,
	})
	return Result{Reply: "OK " + string(resp)}
}
