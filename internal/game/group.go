// Owner: Role A. See ROLE_A_PLAN.md step 8.

package game

import "github.com/charbelbahry/tap/internal/protocol"

// group handles GROUP CREATE, GROUP INVITE <name>, GROUP JOIN <leader>
// and GROUP LEAVE.
func (g *Game) group(p *Player, args string) Result {
	// TODO(A): implement the four subcommands.
	return Result{Reply: protocol.ErrBadRequest}
}

// leaveGroup removes p from its group and returns the GROUP LEAVE event.
// Used by GROUP LEAVE and Disconnect.
func (g *Game) leaveGroup(p *Player) []Event {
	// TODO(A): remove p from p.Group.Members.
	return nil
}

// groupMembersInRoom returns p's group members in p's room, p included.
// Role B uses it for shared kill credit.
func (g *Game) groupMembersInRoom(p *Player) []*Player {
	// TODO(A): return real group members once groups work.
	return []*Player{p}
}
