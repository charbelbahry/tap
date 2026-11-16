// Owner: Role A. See ROLE_A_PLAN.md step 7.

package game

import (
	"fmt"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
)

// chat handles CHAT <GLOBAL|ROOM|GROUP> <message>.
// Reply: OK. Event: EVT <scope> CHAT <name> <message>, sender included.
func (g *Game) chat(p *Player, args string) Result {
	scope, msg, _ := strings.Cut(args, " ")
	scope, msg = strings.ToUpper(scope), strings.TrimSpace(msg)
	if msg == "" {
		return Result{Reply: protocol.ErrBadRequest}
	}

	var to []string
	switch scope {
	case "GLOBAL":
		to = g.allNames()
	case "ROOM":
		to = g.namesInRoom(p.Room)
	case "GROUP":
		if p.Group == nil {
			return Result{Reply: protocol.ErrNotInGroup}
		}
		to = p.Group.Members
	default:
		return Result{Reply: protocol.ErrBadRequest}
	}

	line := fmt.Sprintf("EVT %s CHAT %s %s", scope, p.Name, msg)
	return Result{Reply: protocol.OK, Events: []Event{{To: to, Line: line}}}
}

// who handles WHO. Reply: OK players=<count> (RFC format).
func (g *Game) who() Result {
	return Result{Reply: fmt.Sprintf("OK players=%d", len(g.players))}
}
