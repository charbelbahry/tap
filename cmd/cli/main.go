// Command cli is a terminal client for TAP. Users type raw RFC commands,
// such as LOOK or MOVE north, and see server events as they arrive.
//
// Owner: Role A. See ROLE_A_PLAN.md step 11.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	addr := flag.String("addr", "localhost:4242", "server address")
	flag.Parse()

	// TODO(A): dial addr, print server lines in a goroutine, send typed lines.
	// Add the dependency with: go get github.com/chzyer/readline
	fmt.Fprintf(os.Stderr, "cli: not implemented yet (would connect to %s)\n", *addr)
	os.Exit(1)
}
