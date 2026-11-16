// Command cli is a terminal client for TAP. Users type raw RFC commands,
// such as LOOK or MOVE north, and see server events as they arrive.
//
// Owner: Role A. See ROLE_A_PLAN.md step 11.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/chzyer/readline"
)

func main() {
	addr := flag.String("addr", "localhost:4242", "server address")
	flag.Parse()

	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot connect:", err)
		os.Exit(1)
	}
	defer conn.Close()

	rl, err := readline.New("> ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer rl.Close()

	// Print everything the server sends, while the user types.
	go func() {
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			fmt.Fprintln(rl.Stdout(), sc.Text())
		}
		fmt.Fprintln(rl.Stdout(), "connection closed")
		rl.Close() // unblocks Readline below so the program exits
	}()

	for {
		line, err := rl.Readline()
		if err != nil { // Ctrl+D, Ctrl+C, or the server closed the connection
			return
		}
		fmt.Fprintln(conn, line)
	}
}
