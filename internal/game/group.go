// Owner: Role A. See ROLE_A_PLAN.md step 8.

package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
)

// group handles GROUP CREATE, GROUP INVITE <name>, GROUP JOIN <leader>
// and GROUP LEAVE.
func (g *Game) group(p *Player, args string) Result {
	sub, arg, _ := strings.Cut(args, " ")
	arg = strings.TrimSpace(arg)

	switch strings.ToUpper(sub) {
	case "CREATE":
		if p.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		g.nextGroup++
		p.Group = &Group{ID: fmt.Sprintf("grp.%d", g.nextGroup), Members: []string{p.Name}}
		return Result{Reply: "OK group=" + p.Group.ID}

	case "INVITE":
		if p.Group == nil {
			return Result{Reply: protocol.ErrNotInGroup}
		}
		target := g.players[arg]
		if target == nil {
			return Result{Reply: protocol.ErrPlayerNotFound}
		}
		if target.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		target.Invite = p.Group
		leader := p.Group.Members[0]
		return Result{Reply: protocol.OK, Events: []Event{
			{To: []string{target.Name}, Line: "EVT GROUP INVITE " + leader},
		}}

	case "JOIN":
		if p.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		grp := p.Invite
		if grp == nil || len(grp.Members) == 0 || grp.Members[0] != arg {
			return Result{Reply: protocol.ErrNotInvited}
		}
		grp.Members = append(grp.Members, p.Name)
		p.Group, p.Invite = grp, nil
		return Result{Reply: "OK group=" + grp.ID, Events: []Event{
			{To: grp.Members, Line: "EVT GROUP JOIN " + p.Name},
		}}

	case "LEAVE":
		if p.Group == nil {
			return Result{Reply: protocol.ErrNotInGroup}
		}
		return Result{Reply: protocol.OK, Events: g.leaveGroup(p)}
	}
	return Result{Reply: protocol.ErrBadRequest}
}

// leaveGroup removes p from its group and returns the GROUP LEAVE event.
// Used by GROUP LEAVE and Disconnect.
func (g *Game) leaveGroup(p *Player) []Event {
	grp := p.Group
	if grp == nil {
		return nil
	}
	grp.Members = slices.DeleteFunc(grp.Members, func(n string) bool { return n == p.Name })
	p.Group = nil
	return []Event{{To: grp.Members, Line: "EVT GROUP LEAVE " + p.Name}}
}

// groupMembersInRoom returns p's group members in p's room, p included.
// Role B uses it for shared kill credit.
func (g *Game) groupMembersInRoom(p *Player) []*Player {
	if p.Group == nil {
		return []*Player{p}
	}
	var out []*Player
	for _, name := range p.Group.Members {
		if m := g.players[name]; m != nil && m.Room == p.Room {
			out = append(out, m)
		}
	}
	return out
}
