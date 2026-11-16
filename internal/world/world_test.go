package world_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charbelbahry/tap/internal/world"
	"gopkg.in/yaml.v3"
)

func TestLoad_ValidWorld(t *testing.T) {
	// Find data/world.yaml from repo root
	path := filepath.Join("..", "..", "data", "world.yaml")
	w, err := world.Load(path)
	if err != nil {
		t.Fatalf("world.Load(%q) failed: %v", path, err)
	}

	if w.Map.Start != "loc.votive_chapel" {
		t.Errorf("expected start room 'loc.votive_chapel', got %q", w.Map.Start)
	}
	if w.Map.SafeRoom != "loc.votive_chapel" {
		t.Errorf("expected safe room 'loc.votive_chapel', got %q", w.Map.SafeRoom)
	}
	if len(w.Map.Rooms) == 0 {
		t.Errorf("expected rooms to be loaded, got 0")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := world.Load("non_existent_file.yaml")
	if err == nil {
		t.Fatal("expected error for non-existent file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "bad_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("world:\n  start: [invalid yaml"); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = world.Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected YAML parse error, got nil")
	}
}

// validBaseWorld returns a minimal valid world to mutate for validation tests.
func validBaseWorld() world.World {
	return world.World{
		Map: world.Map{
			Start:    "room1",
			SafeRoom: "room1",
			Rooms: map[string]world.Room{
				"room1": {
					Name:        "Room One",
					Description: "First room",
					Exits:       map[string]string{"east": "room2"},
					Items:       []string{"item1"},
					NPCs:        []string{"npc1"},
				},
				"room2": {
					Name:        "Room Two",
					Description: "Second room",
					Exits:       map[string]string{"west": "room1"},
					NPCs:        []string{"enemy1"},
				},
			},
		},
		Items: map[string]world.Item{
			"item1": {Name: "Test Item", Obtainable: true},
		},
		NPCs: map[string]world.NPC{
			"npc1":   {Name: "Giver", Role: "quest_giver"},
			"enemy1": {Name: "Enemy", Role: "enemy"},
		},
		Quests: map[string]world.Quest{
			"q1": {
				Title:     "Quest One",
				Giver:     "npc1",
				Objective: world.Objective{Kill: "enemy1"},
			},
		},
	}
}

func TestValidate_ValidationCases(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(w *world.World)
		wantErrSub string
	}{
		{
			name: "no rooms",
			mutate: func(w *world.World) {
				w.Map.Rooms = nil
			},
			wantErrSub: "has no rooms",
		},
		{
			name: "missing start room",
			mutate: func(w *world.World) {
				w.Map.Start = "non_existent"
			},
			wantErrSub: "start room",
		},
		{
			name: "missing safe room",
			mutate: func(w *world.World) {
				w.Map.SafeRoom = "non_existent"
			},
			wantErrSub: "safe room",
		},
		{
			name: "dangling exit target",
			mutate: func(w *world.World) {
				r := w.Map.Rooms["room1"]
				r.Exits = map[string]string{"north": "room_void"}
				w.Map.Rooms["room1"] = r
			},
			wantErrSub: "room_void",
		},
		{
			name: "missing room item",
			mutate: func(w *world.World) {
				r := w.Map.Rooms["room1"]
				r.Items = []string{"item_ghost"}
				w.Map.Rooms["room1"] = r
			},
			wantErrSub: "item_ghost",
		},
		{
			name: "missing room NPC",
			mutate: func(w *world.World) {
				r := w.Map.Rooms["room1"]
				r.NPCs = []string{"npc_ghost"}
				w.Map.Rooms["room1"] = r
			},
			wantErrSub: "npc_ghost",
		},
		{
			name: "quest missing giver",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Giver = "nobody"
				w.Quests["q1"] = q
			},
			wantErrSub: "nobody",
		},
		{
			name: "quest kill target not found",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Objective.Kill = "nobody"
				w.Quests["q1"] = q
			},
			wantErrSub: "nobody",
		},
		{
			name: "quest kill target is not enemy",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Objective.Kill = "npc1" // npc1 is quest_giver
				w.Quests["q1"] = q
			},
			wantErrSub: "must have role 'enemy'",
		},
		{
			name: "quest deliver target not found",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Objective.Deliver = "missing_item"
				w.Quests["q1"] = q
			},
			wantErrSub: "missing_item",
		},
		{
			name: "quest missing needs prerequisite",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Needs = "missing_quest"
				w.Quests["q1"] = q
			},
			wantErrSub: "missing_quest",
		},
		{
			name: "quest missing reward item",
			mutate: func(w *world.World) {
				q := w.Quests["q1"]
				q.Reward.Item = "missing_reward_item"
				w.Quests["q1"] = q
			},
			wantErrSub: "missing_reward_item",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := validBaseWorld()
			tt.mutate(&w)
			err := w.Validate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
			}
			if !strings.Contains(err.Error(), tt.wantErrSub) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErrSub, err)
			}
		})
	}
}

// Verify that validBaseWorld passes Validate()
func TestValidate_ValidWorldPasses(t *testing.T) {
	w := validBaseWorld()
	if err := w.Validate(); err != nil {
		t.Fatalf("expected base world to be valid, got: %v", err)
	}
}

// Test unmarshaling via YAML directly to verify field tags
func TestLoad_YAMLRoundTrip(t *testing.T) {
	raw := `
world:
  start: r1
  safe_room: r1
  rooms:
    r1:
      name: "R1"
      description: "D1"
      exits: {}
items: {}
npcs: {}
quests: {}
`
	var w world.World
	if err := yaml.Unmarshal([]byte(raw), &w); err != nil {
		t.Fatalf("failed unmarshaling raw yaml: %v", err)
	}
	if err := w.Validate(); err != nil {
		t.Fatalf("failed validating unmarshaled yaml: %v", err)
	}
}
