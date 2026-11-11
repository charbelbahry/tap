// Owner: Role A. See ROLE_A_PLAN.md step 3.

package server

import "net"

// client is one TCP connection.
type client struct {
	conn net.Conn
	ip   string
	name string      // empty until CONNECT succeeds
	out  chan string // lines waiting to be written by writeLoop
}

// TODO(A): send (non-blocking queue) and writeLoop (the only socket writer).
