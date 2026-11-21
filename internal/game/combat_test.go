package game

import (
	"encoding/json"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

func newTestCombatWorld() *world.World {
	return &world.World{
		Map: world.Map{
			Start:    "loc.votive_chapel",
			SafeRoom: "loc.votive_chapel",
			Rooms: map[string]world.Room{
				"loc.votive_chapel": {
					Name:        "Votive Chapel",
					Description: "Safe sanctuary.",
					Exits:       map[string]string{"east": "loc.pilgrims_road"},
					NPCs:        []string{"npc.sister_ilvane"},
				},
				"loc.pilgrims_road": {
					Name:        "Pilgrim's Road",
					Description: "A long road.",
					Exits:       map[string]string{"west": "loc.votive_chapel"},
					NPCs:        []string{"npc.rotting_sentry"},
				},
				"loc.hall_answer": {
					Name:        "Hall of the Answer",
					Description: "The deep vault.",
					NPCs:        []string{"npc.baal"},
				},
			},
		},
		Items: map[string]world.Item{
			"item.unspoken_question": {
				Name:       "The Unspoken Question",
				Obtainable: true,
			},
		},
		NPCs: map[string]world.NPC{
			"npc.sister_ilvane": {
				Name: "Sister Ilvane",
				Role: "quest_giver",
			},
			"npc.rotting_sentry": {
				Name:  "Rotting Sentry",
				Role:  "enemy",
				Stats: world.Stats{HP: 30, DamageMin: 4, DamageMax: 8},
			},
			"npc.baal": {
				Name:  "Baal",
				Role:  "enemy",
				Stats: world.Stats{HP: 300, DamageMin: 15, DamageMax: 25},
			},
		},
	}
}

func newTestCombatGame() *Game {
	return New(newTestCombatWorld(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestAttack_NonHostileNPC(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")

	res := g.Handle("alice", "ATTACK", "sister_ilvane")
	if res.Reply != protocol.ErrNPCNotHostile {
		t.Fatalf("expected %q, got %q", protocol.ErrNPCNotHostile, res.Reply)
	}
}

func TestAttack_NonexistentNPC(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")

	res := g.Handle("alice", "ATTACK", "ghost")
	if res.Reply != protocol.ErrNPCNotFound {
		t.Fatalf("expected %q, got %q", protocol.ErrNPCNotFound, res.Reply)
	}
}

func TestAttack_EnemyExchange(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")
	g.Connect("bob")

	g.Handle("alice", "MOVE", "east")
	g.Handle("bob", "MOVE", "east")

	alice := g.players["alice"]
	initialHP := alice.HP

	res := g.Handle("alice", "ATTACK", "rotting_sentry")
	if res.Reply[:3] != "OK " {
		t.Fatalf("attack failed: %q", res.Reply)
	}

	var d struct {
		AttackerHP int    `json:"attacker_hp"`
		TargetHP   int    `json:"target_hp"`
		Damage     int    `json:"damage"`
		Status     string `json:"status"`
	}
	if err := json.Unmarshal([]byte(res.Reply[3:]), &d); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if d.Damage < 8 || d.Damage > 12 {
		t.Fatalf("player damage should be in [8, 12], got %d", d.Damage)
	}
	if d.TargetHP != 30-d.Damage {
		t.Fatalf("target HP should be %d, got %d", 30-d.Damage, d.TargetHP)
	}
	if alice.HP >= initialHP {
		t.Fatalf("alice should take counterattack damage, hp=%d", alice.HP)
	}
	if !alice.InCombat || alice.CombatTarget != "npc.rotting_sentry" {
		t.Fatalf("alice should be in combat with sentry")
	}

	// Verify event broadcast to Bob
	if len(res.Events) != 1 || !slices.Contains(res.Events[0].To, "bob") {
		t.Fatalf("expected combat event to bob, got %v", res.Events)
	}
}

func TestAttack_EnemyDefeatAndRespawn(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")
	g.Handle("alice", "MOVE", "east")

	alice := g.players["alice"]
	alice.DamageBonus = 50 // One-shot sentry

	res := g.Handle("alice", "ATTACK", "rotting_sentry")
	var d struct {
		TargetHP int    `json:"target_hp"`
		Status   string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)

	if d.Status != "victory" || d.TargetHP != 0 {
		t.Fatalf("expected victory and 0 HP, got %+v", d)
	}
	if alice.InCombat {
		t.Fatal("alice should not be in combat after enemy dies")
	}

	// LOOK should not list the dead enemy
	lookRes := g.Handle("alice", "LOOK", "")
	var lookData struct {
		NPCs []string `json:"npcs"`
	}
	_ = json.Unmarshal([]byte(lookRes.Reply[3:]), &lookData)
	if slices.Contains(lookData.NPCs, "npc.rotting_sentry") {
		t.Fatalf("dead sentry should not appear in room: %v", lookData.NPCs)
	}

	// Revive after timer expires
	g.enemyRespawnAt["npc.rotting_sentry"] = time.Now().Add(-1 * time.Second)
	if !g.isNPCAlive("npc.rotting_sentry") {
		t.Fatal("sentry should respawn after timer expires")
	}
}

func TestDefend_HalvesDamage(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")

	// Not in combat -> error
	res := g.Handle("alice", "DEFEND", "")
	if res.Reply != protocol.ErrNotInCombat {
		t.Fatalf("expected ErrNotInCombat, got %q", res.Reply)
	}

	g.Handle("alice", "MOVE", "east")
	g.Handle("alice", "ATTACK", "rotting_sentry")

	hpBefore := g.players["alice"].HP
	res = g.Handle("alice", "DEFEND", "")

	var d struct {
		HP          int    `json:"hp"`
		DamageTaken int    `json:"damage_taken"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal([]byte(res.Reply[3:]), &d); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Sentry damage is 4-8, halved is 2-4
	if d.DamageTaken < 2 || d.DamageTaken > 4 {
		t.Fatalf("halved damage should be [2, 4], got %d", d.DamageTaken)
	}
	if d.HP != hpBefore-d.DamageTaken {
		t.Fatalf("expected hp %d, got %d", hpBefore-d.DamageTaken, d.HP)
	}
}

func TestFlee_EscapesCombat(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")

	res := g.Handle("alice", "FLEE", "")
	if res.Reply != protocol.ErrNotInCombat {
		t.Fatalf("expected ErrNotInCombat, got %q", res.Reply)
	}

	g.Handle("alice", "MOVE", "east")
	g.Handle("alice", "ATTACK", "rotting_sentry")

	res = g.Handle("alice", "FLEE", "")
	expected := "OK room=loc.votive_chapel"
	if res.Reply != expected {
		t.Fatalf("expected %q, got %q", expected, res.Reply)
	}

	alice := g.players["alice"]
	if alice.InCombat {
		t.Fatal("alice should not be in combat after fleeing")
	}
	if alice.Room != "loc.votive_chapel" {
		t.Fatalf("expected room loc.votive_chapel, got %s", alice.Room)
	}
}

func TestPlayerDeath_RespawnSafeRoom(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")
	alice := g.players["alice"]

	alice.Room = "loc.pilgrims_road"
	alice.HP = 2 // Extremely low HP
	alice.Inventory = []string{"item.pilgrims_candle"}

	// Attack causes counterattack killing Alice
	res := g.Handle("alice", "ATTACK", "rotting_sentry")
	var d struct {
		AttackerHP int    `json:"attacker_hp"`
		Status     string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)

	if d.Status != "defeat" || d.AttackerHP != 50 {
		t.Fatalf("expected defeat at 50 HP, got %+v", d)
	}
	if alice.Room != "loc.votive_chapel" {
		t.Fatalf("alice should respawn in safe room, got %s", alice.Room)
	}
	if alice.HP != 50 {
		t.Fatalf("alice should have 50 HP, got %d", alice.HP)
	}
	if !slices.Contains(alice.Inventory, "item.pilgrims_candle") {
		t.Fatal("alice should keep items on death")
	}
}

func TestStatus_HealthyWoundedCombat(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")
	alice := g.players["alice"]

	// 100 HP -> healthy
	res := g.Handle("alice", "STATUS", "")
	var d struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "healthy" {
		t.Fatalf("expected healthy, got %s", d.Status)
	}

	// 50 HP -> wounded
	alice.HP = 50
	res = g.Handle("alice", "STATUS", "")
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "wounded" {
		t.Fatalf("expected wounded, got %s", d.Status)
	}

	// InCombat -> combat
	alice.InCombat = true
	res = g.Handle("alice", "STATUS", "")
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "combat" {
		t.Fatalf("expected combat, got %s", d.Status)
	}
}

func TestBaal_ImmunityWithoutQuestion(t *testing.T) {
	g := newTestCombatGame()
	g.Connect("alice")
	alice := g.players["alice"]
	alice.Room = "loc.hall_answer"

	// Alice attacks Baal without item -> 0 damage
	res := g.Handle("alice", "ATTACK", "baal")
	var d struct {
		Damage int `json:"damage"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Damage != 0 {
		t.Fatalf("expected 0 damage to Baal without question, got %d", d.Damage)
	}

	// Alice gets the unspoken question -> can now wound Baal!
	alice.Inventory = append(alice.Inventory, "item.unspoken_question")
	res = g.Handle("alice", "ATTACK", "baal")
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Damage < 8 || d.Damage > 12 {
		t.Fatalf("expected normal damage to Baal with question, got %d", d.Damage)
	}
}
