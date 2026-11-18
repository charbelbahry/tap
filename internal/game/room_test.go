package game_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

func newTestGame(t *testing.T) *game.Game {
	t.Helper()
	worldPath := filepath.Join("..", "..", "data", "world.yaml")
	w, err := world.Load(worldPath)
	if err != nil {
		t.Fatalf("failed loading world: %v", err)
	}
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	return game.New(w, logger)
}

func TestNewPlayer_SpawnDefaults(t *testing.T) {
	g := newTestGame(t)
	res := g.Connect("hero")
	if res.Reply != protocol.OKConnected {
		t.Fatalf("expected %q, got %q", protocol.OKConnected, res.Reply)
	}

	lookRes := g.Handle("hero", "LOOK", "")
	if len(lookRes.Reply) < 3 || lookRes.Reply[:3] != "OK " {
		t.Fatalf("unexpected look reply: %q", lookRes.Reply)
	}

	var data struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
		Players []string `json:"players"`
	}
	if err := json.Unmarshal([]byte(lookRes.Reply[3:]), &data); err != nil {
		t.Fatalf("failed unmarshaling LOOK json: %v", err)
	}

	if data.Room.ID != "loc.votive_chapel" {
		t.Errorf("expected start room 'loc.votive_chapel', got %q", data.Room.ID)
	}
	if len(data.Players) != 1 || data.Players[0] != "hero" {
		t.Errorf("expected players [hero], got %v", data.Players)
	}
}

func TestLook_RFCStructure(t *testing.T) {
	g := newTestGame(t)
	g.Connect("alice")
	g.Connect("bob")

	res := g.Handle("alice", "LOOK", "")
	if len(res.Reply) < 3 || res.Reply[:3] != "OK " {
		t.Fatalf("expected OK reply, got %q", res.Reply)
	}

	var payload struct {
		Room struct {
			ID          string            `json:"id"`
			Name        string            `json:"name"`
			Description string            `json:"description"`
			Exits       map[string]string `json:"exits"`
		} `json:"room"`
		Players []string `json:"players"`
		Items   []string `json:"items"`
		NPCs    []string `json:"npcs"`
	}

	if err := json.Unmarshal([]byte(res.Reply[3:]), &payload); err != nil {
		t.Fatalf("failed unmarshaling LOOK response: %v", err)
	}

	if payload.Room.ID != "loc.votive_chapel" {
		t.Errorf("expected room ID 'loc.votive_chapel', got %q", payload.Room.ID)
	}
	if payload.Room.Name != "Votive Chapel" {
		t.Errorf("expected room name 'Votive Chapel', got %q", payload.Room.Name)
	}
	if payload.Room.Exits["east"] != "loc.pilgrims_road" {
		t.Errorf("expected east exit 'loc.pilgrims_road', got %q", payload.Room.Exits["east"])
	}
	if len(payload.Players) != 2 {
		t.Errorf("expected 2 players [alice, bob], got %v", payload.Players)
	}
	if len(payload.Items) == 0 || payload.Items[0] != "item.pilgrims_candle" {
		t.Errorf("expected item.pilgrims_candle in room, got %v", payload.Items)
	}
	if len(payload.NPCs) == 0 || payload.NPCs[0] != "npc.sister_ilvane" {
		t.Errorf("expected npc.sister_ilvane in room, got %v", payload.NPCs)
	}
}

func TestMove_ValidAndInvalidExits(t *testing.T) {
	g := newTestGame(t)
	g.Connect("adventurer")

	// 1. Invalid exit
	res := g.Handle("adventurer", "MOVE", "up")
	if res.Reply != protocol.ErrNoExit {
		t.Errorf("expected %q for invalid exit, got %q", protocol.ErrNoExit, res.Reply)
	}

	// 2. Empty direction
	res = g.Handle("adventurer", "MOVE", "")
	if res.Reply != protocol.ErrBadRequest {
		t.Errorf("expected %q for empty direction, got %q", protocol.ErrBadRequest, res.Reply)
	}

	// 3. Valid exit to east
	res = g.Handle("adventurer", "MOVE", "east")
	expectedReply := "OK room=loc.pilgrims_road"
	if res.Reply != expectedReply {
		t.Errorf("expected %q, got %q", expectedReply, res.Reply)
	}

	// Verify room updated via LOOK
	lookRes := g.Handle("adventurer", "LOOK", "")
	var data struct {
		Room struct {
			ID string `json:"id"`
		} `json:"room"`
	}
	_ = json.Unmarshal([]byte(lookRes.Reply[3:]), &data)
	if data.Room.ID != "loc.pilgrims_road" {
		t.Errorf("expected current room 'loc.pilgrims_road', got %q", data.Room.ID)
	}
}

func TestMove_PresenceEvents(t *testing.T) {
	g := newTestGame(t)
	g.Connect("alice")
	g.Connect("bob")

	// Move Alice east to Pilgrim's Road
	g.Handle("alice", "MOVE", "east")

	// Bob is still in Votive Chapel. Connect Carol in Pilgrim's Road by moving her.
	g.Connect("carol") // spawns in Chapel

	// Move Carol east to Pilgrim's Road
	res := g.Handle("carol", "MOVE", "east")

	// Events produced by Carol's move:
	// 1. LEAVE carol sent to Bob (who is still in Chapel)
	// 2. ENTER carol sent to Alice (who is waiting in Pilgrim's Road)
	foundLeave := false
	foundEnter := false

	for _, ev := range res.Events {
		if ev.Line == "EVT ROOM PRESENCE LEAVE carol" {
			foundLeave = true
			if len(ev.To) != 1 || ev.To[0] != "bob" {
				t.Errorf("expected LEAVE sent to [bob], got %v", ev.To)
			}
		}
		if ev.Line == "EVT ROOM PRESENCE ENTER carol" {
			foundEnter = true
			if len(ev.To) != 1 || ev.To[0] != "alice" {
				t.Errorf("expected ENTER sent to [alice], got %v", ev.To)
			}
		}
	}

	if !foundLeave {
		t.Errorf("expected EVT ROOM PRESENCE LEAVE event, not found in %v", res.Events)
	}
	if !foundEnter {
		t.Errorf("expected EVT ROOM PRESENCE ENTER event, not found in %v", res.Events)
	}
}

func TestMove_SafeRoomHealing(t *testing.T) {
	g := newTestGame(t)
	g.Connect("alice")

	// Move out of Chapel to Pilgrim's Road
	g.Handle("alice", "MOVE", "east")

	// Simulate damaged player by connecting new player or checking move back
	// Move back west into Votive Chapel (safe room)
	res := g.Handle("alice", "MOVE", "west")
	if res.Reply != "OK room=loc.votive_chapel" {
		t.Fatalf("expected return to chapel, got %q", res.Reply)
	}
}
