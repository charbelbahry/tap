package game

import (
	"io"
	"log/slog"
	"testing"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

func newTestNPCWorld() *world.World {
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
			},
		},
		NPCs: map[string]world.NPC{
			"npc.sister_ilvane": {
				Name:        "Sister Ilvane",
				Description: "Keeper of the last candle.",
				Role:        "quest_giver",
				Dialogue: []string{
					"The flame holds you here, Vigilant. I am sorry for that.",
					"Every time you fall, the candle pulls you back. It has never asked if you wanted that.",
				},
			},
			"npc.rotting_sentry": {
				Name:        "Rotting Sentry",
				Description: "Guarding a dead road.",
				Role:        "enemy",
				Dialogue:    []string{"Halt. Halt. Halt. Halt."},
			},
		},
	}
}

func newTestNPCGame() *Game {
	return New(newTestNPCWorld(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestTalk_FirstDialogue(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	res := g.Handle("alice", "TALK", "sister_ilvane")
	expected := "OK The flame holds you here, Vigilant. I am sorry for that."
	if res.Reply != expected {
		t.Fatalf("expected %q, got %q", expected, res.Reply)
	}
}

func TestTalk_DialogueRotation(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	// 1st talk
	res1 := g.Handle("alice", "TALK", "sister_ilvane")
	expected1 := "OK The flame holds you here, Vigilant. I am sorry for that."
	if res1.Reply != expected1 {
		t.Fatalf("talk 1: expected %q, got %q", expected1, res1.Reply)
	}

	// 2nd talk
	res2 := g.Handle("alice", "TALK", "sister_ilvane")
	expected2 := "OK Every time you fall, the candle pulls you back. It has never asked if you wanted that."
	if res2.Reply != expected2 {
		t.Fatalf("talk 2: expected %q, got %q", expected2, res2.Reply)
	}

	// 3rd talk wraps around
	res3 := g.Handle("alice", "TALK", "sister_ilvane")
	if res3.Reply != expected1 {
		t.Fatalf("talk 3 wrap: expected %q, got %q", expected1, res3.Reply)
	}
}

func TestTalk_FuzzyMatching(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	// Full ID
	res := g.Handle("alice", "TALK", "npc.sister_ilvane")
	if res.Reply != "OK The flame holds you here, Vigilant. I am sorry for that." {
		t.Fatalf("talk with full ID failed: %q", res.Reply)
	}

	// Display name (case-insensitive)
	res = g.Handle("alice", "TALK", "Sister Ilvane")
	if res.Reply != "OK Every time you fall, the candle pulls you back. It has never asked if you wanted that." {
		t.Fatalf("talk with display name failed: %q", res.Reply)
	}

	// Lowercase display name
	res = g.Handle("alice", "TALK", "sister ilvane")
	if res.Reply != "OK The flame holds you here, Vigilant. I am sorry for that." {
		t.Fatalf("talk with lowercase name failed: %q", res.Reply)
	}
}

func TestTalk_NPCInOtherRoom(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	// Alice is in votive chapel, sentry is in pilgrim's road
	res := g.Handle("alice", "TALK", "rotting_sentry")
	if res.Reply != protocol.ErrNPCNotFound {
		t.Fatalf("expected %q for NPC in different room, got %q", protocol.ErrNPCNotFound, res.Reply)
	}

	// Move Alice east to road
	g.Handle("alice", "MOVE", "east")
	res = g.Handle("alice", "TALK", "rotting_sentry")
	if res.Reply != "OK Halt. Halt. Halt. Halt." {
		t.Fatalf("talk after move failed: %q", res.Reply)
	}
}

func TestTalk_NonexistentNPC(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	res := g.Handle("alice", "TALK", "ghost")
	if res.Reply != protocol.ErrNPCNotFound {
		t.Fatalf("expected %q, got %q", protocol.ErrNPCNotFound, res.Reply)
	}
}

func TestTalk_EmptyArgs(t *testing.T) {
	g := newTestNPCGame()
	g.Connect("alice")

	res := g.Handle("alice", "TALK", "")
	if res.Reply != protocol.ErrBadRequest {
		t.Fatalf("expected %q, got %q", protocol.ErrBadRequest, res.Reply)
	}
}
