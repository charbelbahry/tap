// Owner: Role B.
// Package game implements room inspection and player navigation.

package game

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
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

// nonNil returns s if not nil, otherwise an empty slice.
// Cyclomatic Complexity: 2 (base 1 + 1 if)
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// newPlayer creates a player in the start room with full health (100 HP).
// Cyclomatic Complexity: 3 (base 1 + 2 ifs)
func (g *Game) newPlayer(name string) *Player {
	startRoom := "loc.votive_chapel"
	if g.world != nil && g.world.Map.Start != "" {
		startRoom = g.world.Map.Start
	}
	return &Player{
		Name:      name,
		Room:      startRoom,
		HP:        100,
		MaxHP:     100,
		Inventory: []string{},
	}
}

// look handles LOOK. Reply: OK <room JSON> (RFC 42TAP section 5.1.2).
// Cyclomatic Complexity: 5 (base 1 + 4 ifs)
func (g *Game) look(p *Player) Result {
	roomID := p.Room
	var r world.Room
	if g.world != nil {
		if found, ok := g.world.Map.Rooms[roomID]; ok {
			r = found
		}
	}
	if r.Name == "" {
		r.Name = roomID
	}

	exits := r.Exits
	if exits == nil {
		exits = map[string]string{}
	}

	players := nonNil(g.namesInRoom(roomID))
	sort.Strings(players)

	resp := lookResponse{
		Room: lookRoom{
			ID:          roomID,
			Name:        r.Name,
			Description: r.Description,
			Exits:       exits,
		},
		Players: players,
		Items:   nonNil(g.roomItems[roomID]),
		NPCs:    nonNil(r.NPCs),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return Result{Reply: protocol.ErrBadRequest}
	}
	return Result{Reply: "OK " + string(data)}
}

// move handles MOVE <direction>.
// Replies: OK room=<room-id>, ERR 301 NO_EXIT.
// Events: ROOM PRESENCE LEAVE to old room, ROOM PRESENCE ENTER to new room.
// Cyclomatic Complexity: 5 (base 1 + 4 ifs)
func (g *Game) move(p *Player, args string) Result {
	dir := strings.ToLower(strings.TrimSpace(args))
	if dir == "" || g.world == nil {
		return Result{Reply: protocol.ErrNoExit}
	}

	target := g.world.Map.Rooms[p.Room].Exits[dir]
	if target == "" {
		return Result{Reply: protocol.ErrNoExit}
	}

	oldRoom := p.Room
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
// Cyclomatic Complexity: 3 (base 1 + 2 ifs)
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
