// Package gui serves the web client and bridges its WebSocket to the TAP server.
//
// Owner: Role A (bridge). The files in web/ belong to Role C.
// See ROLE_A_PLAN.md step 12.
package gui

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed web
var webFiles embed.FS

// Run serves the page on listen. Each browser tab will open a WebSocket at /ws,
// and the bridge will open one TCP connection to server for it.
func Run(listen, server string) error {
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		return err
	}
	http.Handle("/", http.FileServerFS(static))
	// TODO(A): http.HandleFunc("/ws", ...) to bridge WebSocket <-> TCP at server.
	// Add the dependency with: go get github.com/gorilla/websocket
	return http.ListenAndServe(listen, nil)
}
