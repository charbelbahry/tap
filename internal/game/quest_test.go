package game

import (
	"encoding/json"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/charbelbahry/tap/internal/world"
)

func newTestQuestWorld() *world.World {
	return &world.World{
		Map: world.Map{
			Start:    "loc.votive_chapel",
			SafeRoom: "loc.votive_chapel",
			Rooms: map[string]world.Room{
				"loc.votive_chapel": {
					Name:        "Votive Chapel",
					Description: "Safe sanctuary.",
					Exits:       map[string]string{"east": "loc.pilgrims_road", "north": "loc.crypt_gate"},
					NPCs:        []string{"npc.sister_ilvane"},
				},
				"loc.pilgrims_road": {
					Name:        "Pilgrim's Road",
					Description: "A long road.",
					Exits:       map[string]string{"west": "loc.votive_chapel"},
					NPCs:        []string{"npc.rotting_sentry"},
				},
				"loc.crypt_gate": {
					Name:        "Crypt Gate",
					Description: "A heavy gate.",
					Exits:       map[string]string{"south": "loc.votive_chapel"},
					NPCs:        []string{"npc.old_corvin"},
				},
				"loc.burned_library": {
					Name:        "Burned Library",
					Description: "Ash and stone.",
					Items:       []string{"item.burial_shroud"},
				},
			},
		},
		Items: map[string]world.Item{
			"item.burial_shroud": {
				Name:       "Burial Shroud",
				Obtainable: true,
			},
			"item.vergers_hammer": {
				Name:        "Verger's Hammer",
				Obtainable:  true,
				DamageBonus: 6,
				MaxHPBonus:  20,
			},
		},
		NPCs: map[string]world.NPC{
			"npc.sister_ilvane": {
				Name: "Sister Ilvane",
				Role: "quest_giver",
			},
			"npc.old_corvin": {
				Name: "Old Corvin",
				Role: "quest_giver",
			},
			"npc.rotting_sentry": {
				Name:  "Rotting Sentry",
				Role:  "enemy",
				Stats: world.Stats{HP: 30, DamageMin: 4, DamageMax: 8},
			},
		},
		Quests: map[string]world.Quest{
			"road_without_watchers": {
				Title:       "A Road Without Watchers",
				Giver:       "npc.sister_ilvane",
				Description: "Destroy the Rotting Sentry on Pilgrim's Road.",
				Objective:   world.Objective{Kill: "npc.rotting_sentry"},
				Reward:      world.Reward{DamageBonus: 2},
			},
			"the_last_toll": {
				Title:       "The Last Toll",
				Giver:       "npc.sister_ilvane",
				Description: "Silence the bell.",
				Needs:       "road_without_watchers",
				Objective:   world.Objective{Kill: "npc.rotting_sentry"},
				Reward:      world.Reward{DamageBonus: 5, Title: "Unbound"},
			},
			"a_proper_burial": {
				Title:       "A Proper Burial",
				Giver:       "npc.old_corvin",
				Description: "Deliver the Burial Shroud.",
				Objective:   world.Objective{Deliver: "item.burial_shroud"},
				Reward:      world.Reward{Item: "item.vergers_hammer", MaxHPBonus: 20},
			},
		},
	}
}

func newTestQuestGame() *Game {
	return New(newTestQuestWorld(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestQuest_AcceptanceAndActive(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")

	// 1. Accept quest (first call -> status available)
	res := g.Handle("alice", "QUEST", "sister_ilvane")
	var d struct {
		QuestID string `json:"quest_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal([]byte(res.Reply[3:]), &d); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if d.QuestID != "road_without_watchers" || d.Status != "available" {
		t.Fatalf("expected available for road_without_watchers, got %+v", d)
	}

	// 2. Calling again before completing objective -> status active
	res = g.Handle("alice", "QUEST", "sister_ilvane")
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "active" {
		t.Fatalf("expected active status, got %s", d.Status)
	}
}

func TestQuest_KillCompletionAndReward(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")
	alice := g.players["alice"]

	// Accept quest
	g.Handle("alice", "QUEST", "sister_ilvane")

	// Kill sentry
	g.Handle("alice", "MOVE", "east")
	alice.DamageBonus = 50
	g.Handle("alice", "ATTACK", "rotting_sentry")

	// Return to chapel and complete quest
	g.Handle("alice", "MOVE", "west")
	res := g.Handle("alice", "QUEST", "sister_ilvane")

	var d struct {
		QuestID string `json:"quest_id"`
		Status  string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "completed" {
		t.Fatalf("expected completed status, got %s", d.Status)
	}
	// Initial bonus was 50, reward adds +2 -> 52
	if alice.DamageBonus != 52 {
		t.Fatalf("expected DamageBonus 52, got %d", alice.DamageBonus)
	}
}

func TestQuest_SharedPartyKillCredit(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")
	g.Connect("bob")

	// Form group
	g.Handle("alice", "GROUP", "CREATE")
	g.Handle("alice", "GROUP", "INVITE bob")
	g.Handle("bob", "GROUP", "JOIN alice")

	// Both accept quest
	g.Handle("alice", "QUEST", "sister_ilvane")
	g.Handle("bob", "QUEST", "sister_ilvane")

	// Both move to road
	g.Handle("alice", "MOVE", "east")
	g.Handle("bob", "MOVE", "east")

	// Alice strikes killing blow
	alice := g.players["alice"]
	alice.DamageBonus = 50
	g.Handle("alice", "ATTACK", "rotting_sentry")

	// Both return to chapel
	g.Handle("alice", "MOVE", "west")
	g.Handle("bob", "MOVE", "west")

	// Both can complete the quest!
	resA := g.Handle("alice", "QUEST", "sister_ilvane")
	var dA struct{ Status string }
	_ = json.Unmarshal([]byte(resA.Reply[3:]), &dA)
	if dA.Status != "completed" {
		t.Fatalf("alice should complete quest, got %s", dA.Status)
	}

	resB := g.Handle("bob", "QUEST", "sister_ilvane")
	var dB struct{ Status string }
	_ = json.Unmarshal([]byte(resB.Reply[3:]), &dB)
	if dB.Status != "completed" {
		t.Fatalf("bob should complete quest from shared party credit, got %s", dB.Status)
	}
}

func TestQuest_DeliverCompletionAndRespawn(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")
	alice := g.players["alice"]

	// Alice moves to crypt gate where Old Corvin is
	g.Handle("alice", "MOVE", "north")

	// Accept delivery quest
	res := g.Handle("alice", "QUEST", "old_corvin")
	if res.Reply[:3] != "OK " {
		t.Fatalf("failed to accept quest: %q", res.Reply)
	}

	// Alice gets the burial shroud in inventory
	alice.Inventory = append(alice.Inventory, "item.burial_shroud")

	// Complete quest
	res = g.Handle("alice", "QUEST", "old_corvin")
	var d struct {
		Status string `json:"status"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.Status != "completed" {
		t.Fatalf("expected completed status, got %s", d.Status)
	}

	// Shroud must be removed from inventory
	if slices.Contains(alice.Inventory, "item.burial_shroud") {
		t.Fatal("burial shroud should be removed from inventory upon delivery")
	}

	// Reward item (Verger's Hammer) must be added to inventory
	if !slices.Contains(alice.Inventory, "item.vergers_hammer") {
		t.Fatal("verger's hammer should be rewarded to alice")
	}

	// Shroud must be returned to starting room (loc.burned_library)
	if !slices.Contains(g.roomItems["loc.burned_library"], "item.burial_shroud") {
		t.Fatal("shroud should return to burned library spawn room")
	}
}

func TestQuest_Prerequisites(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")

	// Sister Ilvane gives Quest 1 and Quest 2 (which needs Quest 1)
	// First accept Quest 1
	res := g.Handle("alice", "QUEST", "sister_ilvane")
	var d struct {
		QuestID string `json:"quest_id"`
	}
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.QuestID != "road_without_watchers" {
		t.Fatalf("expected quest 1, got %s", d.QuestID)
	}

	// Kill sentry and complete Quest 1
	g.Handle("alice", "MOVE", "east")
	g.players["alice"].DamageBonus = 50
	g.Handle("alice", "ATTACK", "rotting_sentry")
	g.Handle("alice", "MOVE", "west")
	g.Handle("alice", "QUEST", "sister_ilvane") // completes Quest 1

	// Now Sister Ilvane offers Quest 2 ("the_last_toll")
	res = g.Handle("alice", "QUEST", "sister_ilvane")
	_ = json.Unmarshal([]byte(res.Reply[3:]), &d)
	if d.QuestID != "the_last_toll" {
		t.Fatalf("expected quest 2 to unlock after quest 1, got %s", d.QuestID)
	}
}

func TestQuests_ListingFormat(t *testing.T) {
	g := newTestQuestGame()
	g.Connect("alice")

	// Empty quest list
	res := g.Handle("alice", "QUESTS", "")
	if res.Reply != "OK []" {
		t.Fatalf("expected OK [], got %q", res.Reply)
	}

	// Accept quest
	g.Handle("alice", "QUEST", "sister_ilvane")
	res = g.Handle("alice", "QUESTS", "")

	var list []struct {
		QuestID  string `json:"quest_id"`
		Status   string `json:"status"`
		Progress string `json:"progress"`
	}
	if err := json.Unmarshal([]byte(res.Reply[3:]), &list); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(list) != 1 || list[0].QuestID != "road_without_watchers" || list[0].Progress != "0/1" {
		t.Fatalf("unexpected quest listing: %+v", list)
	}
}
