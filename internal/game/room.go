// Owner: Role B.

package game

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
)

type lookRoom struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

type lookResponse struct {
	Room    lookRoom `json:"room"`
	Players []string `json:"players"`
	Items   []string `json:"items"`
	NPCs    []string `json:"npcs"`
}

// newPlayer creates a player in the start room with full health (100 HP).
func (g *Game) newPlayer(name string) *Player {
	return &Player{
		Name:      name,
		Room:      g.world.Map.Start,
		HP:        100,
		MaxHP:     100,
		Inventory: []string{},
	}
}

// look handles LOOK. Reply: OK <room JSON> (RFC 42TAP section 5.1.2).
func (g *Game) look(p *Player) Result {
	r := g.world.Map.Rooms[p.Room]
	exits := r.Exits
	if exits == nil {
		exits = map[string]string{}
	}

	players := g.namesInRoom(p.Room)
	if players == nil {
		players = []string{}
	}
	sort.Strings(players)

	items := g.roomItems[p.Room]
	if items == nil {
		items = []string{}
	}

	var npcs []string
	for _, id := range r.NPCs {
		if g.isNPCAlive(id) {
			npcs = append(npcs, id)
		}
	}
	if npcs == nil {
		npcs = []string{}
	}

	resp := lookResponse{
		Room: lookRoom{
			ID:          p.Room,
			Name:        r.Name,
			Description: r.Description,
			Exits:       exits,
		},
		Players: players,
		Items:   items,
		NPCs:    npcs,
	}

	data, _ := json.Marshal(resp)
	return Result{Reply: "OK " + string(data)}
}

// move handles MOVE <direction>.
// Replies: OK room=<room-id>, ERR 400 BAD_REQUEST, ERR 301 NO_EXIT.
// Events: ROOM PRESENCE LEAVE to old room, ROOM PRESENCE ENTER to new room.
func (g *Game) move(p *Player, args string) Result {
	dir := strings.ToLower(strings.TrimSpace(args))
	if dir == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}

	target := g.world.Map.Rooms[p.Room].Exits[dir]
	if target == "" {
		return Result{Reply: protocol.ErrNoExit}
	}

	oldRoom := p.Room

	// In combat: guaranteed free counterattack hit before moving
	if p.InCombat {
		g.combatFreeHit(p)
		if p.HP <= 0 {
			g.handlePlayerDeath(p)
			return Result{
				Reply:  "OK room=" + p.Room,
				Events: g.presenceEvents(oldRoom, p),
			}
		}
		p.InCombat = false
		p.CombatTarget = ""
	}

	p.Room = target

	if target == g.world.Map.SafeRoom {
		p.HP = p.MaxHP
	}

	return Result{
		Reply:  "OK room=" + target,
		Events: g.presenceEvents(oldRoom, p),
	}
}

// presenceEvents builds leave events for the old room and enter events for the new room.
func (g *Game) presenceEvents(oldRoom string, p *Player) []Event {
	var events []Event
	if leave := g.namesInRoom(oldRoom); len(leave) > 0 {
		events = append(events, Event{To: leave, Line: "EVT ROOM PRESENCE LEAVE " + p.Name})
	}
	if enter := g.othersInRoom(p); len(enter) > 0 {
		events = append(events, Event{To: enter, Line: "EVT ROOM PRESENCE ENTER " + p.Name})
	}
	return events
}
