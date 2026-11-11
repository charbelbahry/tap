// Owner: Role B.

package game

// take handles TAKE <item>. Items match by full ID, short ID or display name, ignoring case.
// Replies: OK taken=<item-id>, ERR 404 ITEM_NOT_FOUND. Event: ROOM ITEM TAKE.
func (g *Game) take(p *Player, args string) Result {
	// TODO(B)
	return Result{Reply: "OK taken=item.stub"}
}

// drop handles DROP <item>.
// Replies: OK dropped=<item-id>, ERR 404 ITEM_NOT_IN_INVENTORY. Event: ROOM ITEM DROP.
func (g *Game) drop(p *Player, args string) Result {
	// TODO(B)
	return Result{Reply: "OK dropped=item.stub"}
}

// inventory handles INVENTORY. Reply: OK ["item.a","item.b"].
func (g *Game) inventory(p *Player) Result {
	// TODO(B)
	return Result{Reply: "OK []"}
}

// dropAllItems moves everything p carries into p's room.
// Called by Disconnect so unique items are never lost.
func (g *Game) dropAllItems(p *Player) {
	// TODO(B)
}
