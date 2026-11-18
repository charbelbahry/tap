// Owner: Role B.

package game

import (
	"encoding/json"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// matchItem reports whether query matches an item's full ID, short ID, or display name.
func matchItem(query, itemID string, item world.Item) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return false
	}
	if strings.EqualFold(itemID, q) {
		return true
	}
	if strings.EqualFold(strings.TrimPrefix(itemID, "item."), q) {
		return true
	}
	if item.Name != "" && strings.EqualFold(item.Name, q) {
		return true
	}
	return false
}

// findItemInRoom searches items in roomID matching the query.
func (g *Game) findItemInRoom(roomID, query string) (int, string) {
	for i, id := range g.roomItems[roomID] {
		it := g.world.Items[id]
		if matchItem(query, id, it) {
			return i, id
		}
	}
	return -1, ""
}

// findItemInInventory searches items in p's inventory matching the query.
func (g *Game) findItemInInventory(p *Player, query string) (int, string) {
	for i, id := range p.Inventory {
		it := g.world.Items[id]
		if matchItem(query, id, it) {
			return i, id
		}
	}
	return -1, ""
}

// itemEvent builds an EVT ROOM ITEM <action> line for other players in the room.
func itemEvent(to []string, action, player, itemID string) []Event {
	if len(to) == 0 {
		return nil
	}
	return []Event{{
		To:   to,
		Line: "EVT ROOM ITEM " + action + " " + player + " " + itemID,
	}}
}

// take handles TAKE <item>. Items match by full ID, short ID or display name, ignoring case.
// Replies: OK taken=<item-id>, ERR 404 ITEM_NOT_FOUND. Event: ROOM ITEM TAKE.
func (g *Game) take(p *Player, args string) Result {
	if p == nil || strings.TrimSpace(args) == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}
	idx, itemID := g.findItemInRoom(p.Room, args)
	if idx < 0 {
		return Result{Reply: protocol.ErrItemNotFound}
	}

	it := g.world.Items[itemID]
	if !it.Obtainable {
		return Result{Reply: protocol.ErrItemNotFound}
	}

	roomItems := g.roomItems[p.Room]
	g.roomItems[p.Room] = append(roomItems[:idx], roomItems[idx+1:]...)
	p.Inventory = append(p.Inventory, itemID)
	p.DamageBonus += it.DamageBonus
	p.MaxHP += it.MaxHPBonus

	return Result{
		Reply:  "OK taken=" + itemID,
		Events: itemEvent(g.othersInRoom(p), "TAKE", p.Name, itemID),
	}
}

// drop handles DROP <item>.
// Replies: OK dropped=<item-id>, ERR 404 ITEM_NOT_IN_INVENTORY. Event: ROOM ITEM DROP.
func (g *Game) drop(p *Player, args string) Result {
	if p == nil || strings.TrimSpace(args) == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}
	idx, itemID := g.findItemInInventory(p, args)
	if idx < 0 {
		return Result{Reply: protocol.ErrItemNotInInv}
	}

	p.Inventory = append(p.Inventory[:idx], p.Inventory[idx+1:]...)
	g.roomItems[p.Room] = append(g.roomItems[p.Room], itemID)

	it := g.world.Items[itemID]
	p.DamageBonus -= it.DamageBonus
	p.MaxHP -= it.MaxHPBonus
	if p.HP > p.MaxHP {
		p.HP = p.MaxHP
	}

	return Result{
		Reply:  "OK dropped=" + itemID,
		Events: itemEvent(g.othersInRoom(p), "DROP", p.Name, itemID),
	}
}

// inventory handles INVENTORY. Reply: OK ["item.a","item.b"].
func (g *Game) inventory(p *Player) Result {
	if p == nil {
		return Result{Reply: protocol.ErrBadRequest}
	}
	inv := p.Inventory
	if inv == nil {
		inv = []string{}
	}
	data, _ := json.Marshal(inv)
	return Result{Reply: "OK " + string(data)}
}

// dropAllItems moves everything p carries into p's room.
// Called by Disconnect so unique items are never lost.
func (g *Game) dropAllItems(p *Player) {
	if p == nil || len(p.Inventory) == 0 {
		return
	}
	if g.roomItems == nil {
		g.roomItems = make(map[string][]string)
	}
	for _, itemID := range p.Inventory {
		g.roomItems[p.Room] = append(g.roomItems[p.Room], itemID)
		it := g.world.Items[itemID]
		p.DamageBonus -= it.DamageBonus
		p.MaxHP -= it.MaxHPBonus
	}
	p.Inventory = nil
}
