// Package gui serves the web client and bridges its WebSocket to the TAP server.
//
// Owner: Role A (bridge). The files in web/ belong to Role C.
// See ROLE_A_PLAN.md step 12.
package gui

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

//go:embed web
var webFiles embed.FS

// The default upgrader only accepts pages served by this bridge (same origin).
var upgrader = websocket.Upgrader{}

// Run serves the page on listen. Each browser tab opens a WebSocket at /ws,
// and the bridge opens one TCP connection to server for it.
func Run(listen, server string) error {
	h, err := handler(server)
	if err != nil {
		return err
	}
	return http.ListenAndServe(listen, h)
}

// handler serves the embedded page at / and the bridge at /ws.
func handler(server string) (http.Handler, error) {
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServerFS(static))
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		bridge(w, r, server)
	})
	return mux, nil
}

// bridge copies protocol lines between one WebSocket and one TCP connection.
// It does not understand the protocol, so any group's server works.
func bridge(w http.ResponseWriter, r *http.Request, server string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade has already replied with an HTTP error
	}
	defer ws.Close()

	conn, err := net.Dial("tcp", server)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte("ERR 900 CONNECTION_FAILED"))
		return
	}
	defer conn.Close()

	// Server to browser: one WebSocket message per protocol line.
	go func() {
		sc := bufio.NewScanner(conn)
		for sc.Scan() {
			if ws.WriteMessage(websocket.TextMessage, sc.Bytes()) != nil {
				break
			}
		}
		ws.Close() // the server hung up, so end the browser side too
	}()

	// Browser to server.
	for {
		_, msg, err := ws.ReadMessage()
		if err != nil {
			return
		}
		if _, err := fmt.Fprintf(conn, "%s\n", msg); err != nil {
			return
		}
	}
}
