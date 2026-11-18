package game

import (
	"encoding/json"
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

func newTestItemWorld() *world.World {
	return &world.World{
		Map: world.Map{
			Start:    "loc.votive_chapel",
			SafeRoom: "loc.votive_chapel",
			Rooms: map[string]world.Room{
				"loc.votive_chapel": {
					Name:        "Votive Chapel",
					Description: "Safe sanctuary.",
					Exits:       map[string]string{"east": "loc.pilgrims_road"},
					Items:       []string{"item.pilgrims_candle", "item.sealed_altar"},
				},
				"loc.pilgrims_road": {
					Name:        "Pilgrim's Road",
					Description: "A long road.",
					Exits:       map[string]string{"west": "loc.votive_chapel"},
					Items:       []string{"item.vergers_hammer"},
				},
			},
		},
		Items: map[string]world.Item{
			"item.pilgrims_candle": {
				Name:       "Pilgrim's Candle",
				Obtainable: true,
			},
			"item.sealed_altar": {
				Name:       "Sealed Altar",
				Obtainable: false,
			},
			"item.vergers_hammer": {
				Name:        "Verger's Hammer",
				Obtainable:  true,
				DamageBonus: 6,
				MaxHPBonus:  20,
			},
		},
	}
}

func newTestItemGame() *Game {
	return New(newTestItemWorld(), slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestTakeItemByDisplayName(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")
	g.Connect("bob")

	// TAKE with exact display name
	res := g.Handle("alice", "TAKE", "Pilgrim's Candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("expected OK taken=item.pilgrims_candle, got %q", res.Reply)
	}

	alice := g.players["alice"]
	if !slices.Contains(alice.Inventory, "item.pilgrims_candle") {
		t.Fatalf("alice inventory should contain candle, got %v", alice.Inventory)
	}

	// Verify room no longer has it
	if slices.Contains(g.roomItems["loc.votive_chapel"], "item.pilgrims_candle") {
		t.Fatal("room should no longer contain taken item")
	}

	// Verify event was broadcast to Bob (in same room)
	if len(res.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(res.Events))
	}
	evt := res.Events[0]
	if !slices.Contains(evt.To, "bob") || evt.Line != "EVT ROOM ITEM TAKE alice item.pilgrims_candle" {
		t.Fatalf("unexpected event: %+v", evt)
	}
}

func TestTakeItemByShortIDAndFullID(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")

	// TAKE using short ID
	res := g.Handle("alice", "TAKE", "pilgrims_candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("expected OK taken=item.pilgrims_candle with short ID, got %q", res.Reply)
	}

	// DROP it back
	res = g.Handle("alice", "DROP", "item.pilgrims_candle")
	if res.Reply != "OK dropped=item.pilgrims_candle" {
		t.Fatalf("expected OK dropped, got %q", res.Reply)
	}

	// TAKE using full ID
	res = g.Handle("alice", "TAKE", "item.pilgrims_candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("expected OK taken=item.pilgrims_candle with full ID, got %q", res.Reply)
	}
}

func TestTakeUnobtainableItem(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")

	res := g.Handle("alice", "TAKE", "Sealed Altar")
	if res.Reply != protocol.ErrItemNotFound {
		t.Fatalf("expected %q for unobtainable item, got %q", protocol.ErrItemNotFound, res.Reply)
	}

	alice := g.players["alice"]
	if len(alice.Inventory) != 0 {
		t.Fatalf("alice inventory should be empty, got %v", alice.Inventory)
	}
}

func TestItemUniquenessAndDuplicateTake(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")
	g.Connect("bob")

	// Alice takes the candle
	res := g.Handle("alice", "TAKE", "pilgrims_candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("alice take failed: %q", res.Reply)
	}

	// Bob attempts to take the same candle
	res = g.Handle("bob", "TAKE", "pilgrims_candle")
	if res.Reply != protocol.ErrItemNotFound {
		t.Fatalf("bob should get ERR 404 for already taken item, got %q", res.Reply)
	}
}

func TestDropItem(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")
	g.Connect("bob")

	g.Handle("alice", "TAKE", "pilgrims_candle")

	// Alice drops the candle
	res := g.Handle("alice", "DROP", "Pilgrim's Candle")
	if res.Reply != "OK dropped=item.pilgrims_candle" {
		t.Fatalf("expected OK dropped, got %q", res.Reply)
	}

	alice := g.players["alice"]
	if len(alice.Inventory) != 0 {
		t.Fatalf("alice inventory should be empty after drop, got %v", alice.Inventory)
	}

	// Verify room now has the candle
	if !slices.Contains(g.roomItems["loc.votive_chapel"], "item.pilgrims_candle") {
		t.Fatal("room should contain dropped candle")
	}

	// Verify Bob received the DROP event
	if len(res.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(res.Events))
	}
	if res.Events[0].Line != "EVT ROOM ITEM DROP alice item.pilgrims_candle" {
		t.Fatalf("unexpected drop event line: %q", res.Events[0].Line)
	}

	// Bob can now take the dropped item
	res = g.Handle("bob", "TAKE", "pilgrims_candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("bob should be able to take dropped item, got %q", res.Reply)
	}
}

func TestDropItemNotInInventory(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")

	res := g.Handle("alice", "DROP", "vergers_hammer")
	if res.Reply != protocol.ErrItemNotInInv {
		t.Fatalf("expected %q, got %q", protocol.ErrItemNotInInv, res.Reply)
	}
}

func TestItemStatBonuses(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")

	g.Handle("alice", "MOVE", "east") // Move to pilgrim's road where hammer is
	alice := g.players["alice"]

	origMaxHP := alice.MaxHP
	origDmg := alice.DamageBonus

	res := g.Handle("alice", "TAKE", "Verger's Hammer")
	if res.Reply != "OK taken=item.vergers_hammer" {
		t.Fatalf("take hammer failed: %q", res.Reply)
	}

	if alice.DamageBonus != origDmg+6 {
		t.Fatalf("expected DamageBonus %d, got %d", origDmg+6, alice.DamageBonus)
	}
	if alice.MaxHP != origMaxHP+20 {
		t.Fatalf("expected MaxHP %d, got %d", origMaxHP+20, alice.MaxHP)
	}

	// Drop hammer reverts bonuses
	res = g.Handle("alice", "DROP", "Verger's Hammer")
	if res.Reply != "OK dropped=item.vergers_hammer" {
		t.Fatalf("drop hammer failed: %q", res.Reply)
	}

	if alice.DamageBonus != origDmg {
		t.Fatalf("expected DamageBonus %d, got %d", origDmg, alice.DamageBonus)
	}
	if alice.MaxHP != origMaxHP {
		t.Fatalf("expected MaxHP %d, got %d", origMaxHP, alice.MaxHP)
	}
}

func TestInventoryFormat(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")

	// Empty inventory
	res := g.Handle("alice", "INVENTORY", "")
	if res.Reply != "OK []" {
		t.Fatalf("expected OK [], got %q", res.Reply)
	}

	// Take item
	g.Handle("alice", "TAKE", "pilgrims_candle")
	res = g.Handle("alice", "INVENTORY", "")

	var items []string
	if err := json.Unmarshal([]byte(res.Reply[3:]), &items); err != nil {
		t.Fatalf("invalid inventory JSON: %v", err)
	}
	if len(items) != 1 || items[0] != "item.pilgrims_candle" {
		t.Fatalf("unexpected inventory contents: %v", items)
	}
}

func TestDropAllItemsOnDisconnect(t *testing.T) {
	g := newTestItemGame()
	g.Connect("alice")
	g.Connect("bob")

	// Alice takes candle and moves east
	g.Handle("alice", "TAKE", "pilgrims_candle")
	g.Handle("alice", "MOVE", "east")

	// Alice disconnects abruptly
	g.Disconnect("alice")

	// Candle should now be on floor in loc.pilgrims_road
	if !slices.Contains(g.roomItems["loc.pilgrims_road"], "item.pilgrims_candle") {
		t.Fatalf("pilgrims_road should contain dropped candle, got %v", g.roomItems["loc.pilgrims_road"])
	}

	// Bob moves east and picks up the dropped candle
	g.Handle("bob", "MOVE", "east")
	res := g.Handle("bob", "TAKE", "pilgrims_candle")
	if res.Reply != "OK taken=item.pilgrims_candle" {
		t.Fatalf("bob failed to take dropped item: %q", res.Reply)
	}
}
