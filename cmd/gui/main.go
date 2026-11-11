// Command gui serves the TAP web client and connects it to a TAP server.
//
// Owner: Role A.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charbelbahry/tap/internal/gui"
)

func main() {
	port := flag.Int("port", 8080, "port for the web page")
	server := flag.String("server", "localhost:4242", "TAP server address")
	flag.Parse()

	listen := fmt.Sprintf("127.0.0.1:%d", *port)
	fmt.Printf("Open http://%s in your browser\n", listen)
	if err := gui.Run(listen, *server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
