// Owner: Role A. See ROLE_A_PLAN.md step 3.

package server

import (
	"io"
	"net"
	"time"
)

// client is one TCP connection.
type client struct {
	conn net.Conn
	ip   string
	name string      // empty until CONNECT succeeds
	out  chan string // lines waiting to be written by writeLoop
	cmds window      // commands in the current second, for flood detection
}

// send queues a line without ever blocking. Call it only while holding s.mu.
func (c *client) send(line string) {
	select {
	case c.out <- line:
	default:
		// The buffer is full: the client stopped reading. Closing the socket
		// makes its read loop fail, which runs the normal cleanup.
		c.conn.Close()
	}
}

// writeLoop is the only place that writes to the socket.
// It ends when c.out is closed and drained, or on a write error.
func (c *client) writeLoop() {
	defer c.conn.Close()
	for line := range c.out {
		c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if _, err := io.WriteString(c.conn, line+"\n"); err != nil {
			return
		}
	}
}
