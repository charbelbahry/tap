// Owner: Role A. See ROLE_A_PLAN.md step 7.

package game

import (
	"fmt"

	"github.com/charbelbahry/tap/internal/protocol"
)

// chat handles CHAT <GLOBAL|ROOM|GROUP> <message>.
// Reply: OK. Event: EVT <scope> CHAT <name> <message>, sender included.
func (g *Game) chat(p *Player, args string) Result {
	// TODO(A): parse the scope, pick recipients, return the event.
	return Result{Reply: protocol.OK}
}

// who handles WHO. Reply: OK players=<count> (RFC format).
func (g *Game) who() Result {
	return Result{Reply: fmt.Sprintf("OK players=%d", len(g.players))}
}
