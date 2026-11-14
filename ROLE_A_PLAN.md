# Role A: Network and Social - Implementation Plan

This is the step-by-step guide for Role A in [PLAN.md](PLAN.md). It covers the TCP server, connection handling, CONNECT, QUIT, CHAT, WHO, GROUP, logging, abuse detection, the CLI client and the GUI bridge.

**Rules for this plan**

- Use the simplest approach that is correct and complies with the subject and RFC.
- No interfaces, frameworks, config files, context trees or graceful shutdown. Nothing the subject doesn't ask for.
- Idiomatic Go 1.27, formatted with `gofmt`.
- Every snippet is a starting point. Type it yourself and make sure you understand each line, because evaluators can ask you to explain or change any of it.

**Contents**

- [Step 0: Set up Go](#step-0-set-up-go)
- [Step 1: Shared code for Day 1](#step-1-shared-code-for-day-1)
- [Step 2: Accept connections](#step-2-accept-connections)
- [Step 3: Read and write per client](#step-3-read-and-write-per-client)
- [Step 4: Handle each line](#step-4-handle-each-line)
- [Step 5: Deliver events and clean up disconnects](#step-5-deliver-events-and-clean-up-disconnects)
- [Step 6: CONNECT and disconnect in the game](#step-6-connect-and-disconnect-in-the-game)
- [Step 7: CHAT and WHO](#step-7-chat-and-who)
- [Step 8: GROUP](#step-8-group)
- [Step 9: Abuse detection and limits](#step-9-abuse-detection-and-limits)
- [Step 10: Logging](#step-10-logging)
- [Step 11: CLI client](#step-11-cli-client)
- [Step 12: GUI bridge](#step-12-gui-bridge)
- [Step 13: Tests](#step-13-tests)
- [Step 14: README sections](#step-14-readme-sections)
- [What we deliberately skip](#what-we-deliberately-skip)
- [Evaluation prep](#evaluation-prep)

---

## The whole design in one picture

```
              one goroutine per client                     one lock for everything
           +---------------------------+              +------------------------------+
client --> | read loop: bufio.Scanner  | --- line --> | s.mu.Lock()                  |
           +---------------------------+              |   game.Handle(...) -> Result |
                                                      |   send Reply to this client  |
           +---------------------------+              |   send Events to recipients  |
client <-- | write loop: drains c.out  | <-- chan --- | s.mu.Unlock()                |
           +---------------------------+              +------------------------------+
```

**Three ideas make this work:**

1. **One lock (`s.mu`) around every game call and every send.** The game code needs no locking, which makes it much simpler for Role B. Events also always arrive in the same order the game produced them. With at most a few dozen players, one lock is plenty fast.
2. **Each client has a buffered channel, and only its write goroutine touches the socket for writing.** Sending never blocks. If a client's channel is full, it's too slow or dead, so we close its connection and move on. A broadcast can therefore never get stuck, which is what the subject means by "broadcasts without interruption if a client disconnects mid-send".
3. **The read goroutine owns the client's lifetime.** When reading stops for any reason (QUIT, a closed socket, a line that's too long), it runs one cleanup function. Every exit path goes through that one place.

---

## Step 0: Set up Go

**Do:**

1. Install **Go 1.27** (1.27.4 is the latest release at the time of writing):
   - **Ubuntu:** download the Linux tarball from [go.dev/dl](https://go.dev/dl/) and follow the install steps there. The `apt` version is usually too old.
   - **Windows:** run the `.msi` installer from the same page.
2. Check with `go version`.
3. With the team, create the module (replace the path with your repository's):

```sh
go mod init github.com/charbelbahry/tap
```

`go.mod` must say `go 1.27`.

**Why:** Go modules are the build tool the subject names for Go. Version 1.27 gives us everything used below: `strings.Cut`, `slices`, `log/slog`, `http.FileServerFS` and `slog.DiscardHandler`.

---

## Step 1: Shared code for Day 1

You write these with the team on Day 1. Everyone builds on top of them.

### `internal/protocol/protocol.go`

All fixed reply strings live in one place, so nobody misspells an error code.

```go
// Package protocol holds the fixed strings and parsing rules of RFC 42TAP.
package protocol

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	Greeting    = "OK hello proto=1"
	OK          = "OK"
	OKConnected = "OK connected"
	OKBye       = "OK bye"

	// RFC 42TAP error codes.
	ErrNameInUse         = "ERR 201 NAME_IN_USE"
	ErrNoExit            = "ERR 301 NO_EXIT"
	ErrNotInGroup        = "ERR 401 NOT_IN_GROUP"
	ErrAlreadyInGroup    = "ERR 402 ALREADY_IN_GROUP"
	ErrItemNotFound      = "ERR 404 ITEM_NOT_FOUND"
	ErrItemNotInInv      = "ERR 404 ITEM_NOT_IN_INVENTORY"
	ErrNPCNotFound       = "ERR 404 NPC_NOT_FOUND"
	ErrNPCNotHostile     = "ERR 405 NPC_NOT_HOSTILE"
	ErrNoQuest           = "ERR 406 NO_QUEST_AVAILABLE"
	ErrConnectionFailed  = "ERR 900 CONNECTION_FAILED"

	// Our extensions, documented in the README.
	ErrBadRequest        = "ERR 400 BAD_REQUEST"
	ErrAlreadyConnected  = "ERR 402 ALREADY_CONNECTED"
	ErrNotAuthenticated  = "ERR 403 NOT_AUTHENTICATED"
	ErrPlayerNotFound    = "ERR 404 PLAYER_NOT_FOUND"
	ErrNotInvited        = "ERR 407 NOT_INVITED"
	ErrNotInCombat       = "ERR 408 NOT_IN_COMBAT"
	ErrRateLimited       = "ERR 429 RATE_LIMITED"
)

// Split returns the upper-cased command and the rest of the line.
// "chat ROOM  hello there" gives ("CHAT", "ROOM  hello there").
func Split(line string) (cmd, args string) {
	cmd, args, _ = strings.Cut(strings.TrimSpace(line), " ")
	return strings.ToUpper(cmd), strings.TrimSpace(args)
}

// Valid reports whether a line is UTF-8 with no control characters.
func Valid(line string) bool {
	return utf8.ValidString(line) && !strings.ContainsFunc(line, unicode.IsControl)
}
```

**Why:**

- **`Split`:** the RFC says command names are case-insensitive, so we upper-case only the command. Everything after the first space is the argument, which is how multi-word item names and chat messages work.
- **`Valid`:** the RFC requires UTF-8 and says control characters should be rejected. Rejecting them also guarantees no line we send out ever contains a hidden newline, since chat text is copied into events.

### `internal/game/game.go`

This is the contract from PLAN.md, plus the player and group types and helpers both A and B use.

```go
// Package game holds all game rules. It never touches the network.
// It is not safe for concurrent use: the server calls it under one lock.
package game

import (
	"log/slog"

	"github.com/charbelbahry/tap/internal/protocol"
	"github.com/charbelbahry/tap/internal/world"
)

// Event is one line to push to a set of players.
type Event struct {
	To   []string
	Line string
}

// Result is what a command produces. The server sends Reply first, then Events.
type Result struct {
	Reply  string
	Events []Event
	Close  bool
}

type Player struct {
	Name   string
	Room   string
	Group  *Group
	Invite *Group // last group this player was invited to
	// Role B adds HP, inventory, quests and so on.
}

// Group members are kept in join order. Members[0] is the leader.
type Group struct {
	ID      string
	Members []string
}

type Game struct {
	world     *world.World
	log       *slog.Logger
	players   map[string]*Player
	nextGroup int
}

func New(w *world.World, log *slog.Logger) *Game {
	return &Game{world: w, log: log, players: map[string]*Player{}}
}

// Handle runs one command for a connected player.
func (g *Game) Handle(name, cmd, args string) Result {
	p := g.players[name]
	switch cmd {
	case "QUIT":
		return Result{Reply: protocol.OKBye, Close: true}
	case "CHAT":
		return g.chat(p, args)
	case "WHO":
		return g.who()
	case "GROUP":
		return g.group(p, args)
	case "LOOK":
		return g.look(p)
	case "MOVE":
		return g.move(p, args)
	// ... one case per Role B command: TAKE, DROP, INVENTORY, TALK,
	// ATTACK, DEFEND, FLEE, STATUS, QUEST, QUESTS
	default:
		return Result{Reply: protocol.ErrBadRequest}
	}
}

func (g *Game) allNames() []string {
	names := make([]string, 0, len(g.players))
	for name := range g.players {
		names = append(names, name)
	}
	return names
}

func (g *Game) namesInRoom(room string) []string {
	var names []string
	for _, p := range g.players {
		if p.Room == room {
			names = append(names, p.Name)
		}
	}
	return names
}

// othersInRoom is everyone in p's room except p.
func (g *Game) othersInRoom(p *Player) []string {
	var names []string
	for _, other := range g.players {
		if other.Room == p.Room && other != p {
			names = append(names, other.Name)
		}
	}
	return names
}
```

**Day 1 stubs:** every Role B handler (`look`, `move` and so on) starts as a one-line function returning a valid fake reply, and Role B also stubs:

- `newPlayer(name string) *Player`, which sets the start room
- `dropAllItems(p *Player)`, which starts as a no-op
- `world.Load(path string) (*world.World, error)`

That way your server runs from Day 1.

**Why no mutex in the game:** the server already holds one lock for every call (see Step 4). A second lock inside the game would protect nothing new and could cause deadlocks.

---

## Step 2: Accept connections

### `cmd/server/main.go`

```go
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/charbelbahry/tap/internal/game"
	"github.com/charbelbahry/tap/internal/server"
	"github.com/charbelbahry/tap/internal/world"
)

func main() {
	port := flag.Int("port", 4242, "TCP port to listen on")
	worldPath := flag.String("world", "data/world.yaml", "world data file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	w, err := world.Load(*worldPath)
	if err != nil {
		log.Error("world_invalid", "err", err)
		os.Exit(1)
	}

	s := server.New(game.New(w, log), log)
	if err := s.Run(fmt.Sprintf(":%d", *port)); err != nil {
		log.Error("server_stopped", "err", err)
		os.Exit(1)
	}
}
```

### `internal/server/server.go` (first part)

```go
// Package server handles TCP connections and passes lines to the game.
package server

import (
	"log/slog"
	"net"
	"sync"

	"github.com/charbelbahry/tap/internal/game"
)

type Server struct {
	game *game.Game
	log  *slog.Logger

	mu      sync.Mutex         // guards everything below and every game call
	clients map[string]*client // connected players by name
	conns   int                // open connections, logged in or not
	recent  map[string]*window // connection attempts per IP
}

func New(g *game.Game, log *slog.Logger) *Server {
	return &Server{
		game:    g,
		log:     log,
		clients: map[string]*client{},
		recent:  map[string]*window{},
	}
}

func (s *Server) Run(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.log.Info("listening", "addr", addr)
	return s.Serve(ln)
}

// Serve is split from Run so tests can pass a listener on a random port.
func (s *Server) Serve(ln net.Listener) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go s.handleConn(conn)
	}
}
```

**Why:**

- **One goroutine per connection** is the standard Go way to write a TCP server. Goroutines are cheap, so there's no need for an event loop or a worker pool.
- **Any `Accept` error stops the server.** In practice that only happens when the listener is closed, as it is at the end of tests.
- **The world is loaded before listening.** A broken world file stops the server at startup, as the subject requires.

---

## Step 3: Read and write per client

### `internal/server/client.go`

```go
package server

import (
	"io"
	"net"
	"time"
)

type client struct {
	conn net.Conn
	ip   string
	name string      // empty until CONNECT succeeds
	out  chan string // lines waiting to be written
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
```

**Why:**

- **One writer per socket** means lines are never mixed together. The GUI's pending queue relies on every reply arriving as one complete line, in order.
- **A non-blocking `send`** means a broadcast can't stall on one dead client.
- **`writeLoop` closes the socket only after draining the channel**, so on QUIT the `OK bye` line is fully sent before the connection closes. No extra code is needed for that.
- **The write deadline** handles a client that stops reading without disconnecting. Otherwise the write goroutine would wait forever.
- **"Call only while holding `s.mu`"** is the one rule that makes closing `c.out` safe. Cleanup removes the client and closes the channel under the same lock, so nobody can send to a closed channel.

### `internal/server/server.go` (connection handler)

Snippets from here on leave out the `import` block. Your editor (with the Go extension) or `goimports` adds the imports for you.

```go
const (
	maxLine        = 1024 // RFC 42TAP recommended line limit
	maxClients     = 100
	cmdLimit       = 10 // commands per second before ERR 429
	rapidConnLimit = 10 // connections per minute per IP before a warning
)

func (s *Server) handleConn(conn net.Conn) {
	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if !s.admit(ip) {
		fmt.Fprintln(conn, protocol.ErrConnectionFailed)
		conn.Close()
		return
	}

	c := &client{conn: conn, ip: ip, out: make(chan string, 256)}
	go c.writeLoop()
	defer s.cleanup(c)

	s.mu.Lock()
	s.reply(c, protocol.Greeting)
	s.mu.Unlock()

	sc := bufio.NewScanner(conn)
	sc.Buffer(make([]byte, maxLine+1), maxLine+1) // +1 for the newline
	for sc.Scan() {
		if !s.handleLine(c, sc.Text()) {
			return
		}
	}
	if errors.Is(sc.Err(), bufio.ErrTooLong) {
		s.mu.Lock()
		s.reply(c, protocol.ErrBadRequest)
		s.mu.Unlock()
		s.log.Warn("line_too_long", "ip", ip, "player", c.name)
	}
}
```

**Why:**

- **`bufio.Scanner` handles TCP framing for us.** If a command arrives in pieces, `Scan` waits for the newline. If several commands arrive in one packet, each `Scan` returns the next one. That covers the RFC's fragmentation and coalescing requirements.
- **The scanner buffer is capped at 1024 bytes**, the RFC's recommended line limit. A longer line makes `Scan` stop with `ErrTooLong`. We reply with an error and disconnect, because once a scanner fails it can't resume. That's simple and acceptable, since no normal client sends a 1 KB command.
- **The greeting goes through the same `send` path as everything else**, so it's guaranteed to be the first line the client receives.

---

## Step 4: Handle each line

### `internal/server/server.go`

```go
// handleLine runs one command. It returns false when the connection should close.
func (s *Server) handleLine(c *client, line string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	line = strings.TrimSuffix(line, "\r") // accept CRLF clients
	cmd, args := protocol.Split(line)
	if cmd == "" {
		return true // ignore blank lines
	}
	s.log.Info("command", "ip", c.ip, "player", c.name, "cmd", cmd, "args", args)

	if n := c.cmds.hit(time.Second); n > cmdLimit {
		if n == cmdLimit+1 {
			s.log.Warn("flood", "ip", c.ip, "player", c.name)
		}
		s.reply(c, protocol.ErrRateLimited)
		return true
	}
	if !protocol.Valid(line) {
		s.reply(c, protocol.ErrBadRequest)
		return true
	}

	var res game.Result
	switch {
	case c.name == "" && cmd == "CONNECT":
		res = s.game.Connect(args)
		if res.Reply == protocol.OKConnected {
			c.name = args
			s.clients[args] = c
		}
	case c.name == "" && cmd == "QUIT":
		res = game.Result{Reply: protocol.OKBye, Close: true}
	case c.name == "":
		res = game.Result{Reply: protocol.ErrNotAuthenticated}
	case cmd == "CONNECT":
		res = game.Result{Reply: protocol.ErrAlreadyConnected}
	default:
		res = s.game.Handle(c.name, cmd, args)
	}

	s.reply(c, res.Reply)
	s.deliver(res.Events)
	return !res.Close
}

func (s *Server) reply(c *client, line string) {
	if strings.HasPrefix(line, "ERR") {
		s.log.Warn("reply", "ip", c.ip, "player", c.name, "line", line)
	} else {
		s.log.Info("reply", "ip", c.ip, "player", c.name, "line", line)
	}
	c.send(line)
}
```

**Why:**

- **The switch is the RFC state machine.** The four states are CONNECTED (no name yet), AUTHENTICATED (has a name), TERMINATED (QUIT) and the error cases in between. A plain `switch` shows them all in one screen.
- **The reply is sent before the events** for the same command, matching the subject's examples (`OK`, then `EVT GLOBAL CHAT ...`).
- **Blank lines are ignored** because people often press Enter by accident in `nc` or the CLI. Document this in the README.
- **The whole function holds `s.mu`,** so only one command in the whole server runs at a time. The game code stays simple, and events always reflect game state in the right order.

---

## Step 5: Deliver events and clean up disconnects

### `internal/server/server.go`

```go
// deliver sends each event to the players it names. Call it while holding s.mu.
func (s *Server) deliver(events []game.Event) {
	for _, e := range events {
		s.log.Info("event", "line", e.Line, "recipients", len(e.To))
		for _, name := range e.To {
			if c, ok := s.clients[name]; ok {
				c.send(e.Line)
			}
		}
	}
}

// cleanup runs exactly once per connection, when its read loop ends.
func (s *Server) cleanup(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c.name != "" {
		delete(s.clients, c.name)
		s.deliver(s.game.Disconnect(c.name))
	}
	s.conns--
	close(c.out) // writeLoop sends what is left, then closes the socket
	s.log.Info("disconnect", "ip", c.ip, "player", c.name)
}
```

**Why:**

- **Removing the player before delivering the leave events** is exactly the subject's rule: "remove player state before broadcasting leave events". The leaving player is gone from both the server's map and the game, so they can't receive their own leave event.
- **The same `cleanup` runs for QUIT, a dropped connection, a crash or a slow client.** One path means one place to get right.
- **Closing `c.out` instead of the socket** lets the write loop finish sending first, so `OK bye` always arrives.

---

## Step 6: CONNECT and disconnect in the game

### `internal/game/session.go`

```go
package game

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charbelbahry/tap/internal/protocol"
)

func (g *Game) Connect(name string) Result {
	if !validName(name) {
		return Result{Reply: protocol.ErrBadRequest}
	}
	if _, taken := g.players[name]; taken {
		return Result{Reply: protocol.ErrNameInUse}
	}
	p := g.newPlayer(name) // Role B: start room, HP and so on
	g.players[name] = p
	return Result{
		Reply: protocol.OKConnected,
		Events: []Event{
			{To: g.othersInRoom(p), Line: "EVT ROOM PRESENCE ENTER " + name},
			g.statsEvent(),
		},
	}
}

// Disconnect removes the player first, then builds the events others should see.
func (g *Game) Disconnect(name string) []Event {
	p := g.players[name]
	delete(g.players, name)
	g.dropAllItems(p) // Role B: items go back into the room

	events := g.leaveGroup(p)
	return append(events,
		Event{To: g.namesInRoom(p.Room), Line: "EVT ROOM PRESENCE LEAVE " + name},
		g.statsEvent(),
	)
}

func (g *Game) statsEvent() Event {
	return Event{To: g.allNames(), Line: fmt.Sprintf("EVT STATS players=%d", len(g.players))}
}

// validName allows 1 to 32 characters, Unicode included, with no whitespace.
// Control characters were already rejected by protocol.Valid.
func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	return n >= 1 && n <= 32 && !strings.ContainsFunc(name, unicode.IsSpace)
}
```

**Why:**

- **Names can't contain spaces** because events like `EVT ROOM CHAT <username> <message>` are split on spaces. A name like `bob smith` would break every client's parser.
- **`EVT STATS` is sent on every connect and disconnect,** which keeps the GUI's online counter correct without polling.
- **The name becomes free again automatically** once `Disconnect` deletes it from the map.

---

## Step 7: CHAT and WHO

### `internal/game/chat.go`

```go
package game

import (
	"fmt"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
)

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

func (g *Game) who() Result {
	return Result{Reply: fmt.Sprintf("OK players=%d", len(g.players))}
}
```

**Why:**

- **The sender is included in the recipients,** as in the subject's example. The sender's client then shows its own message from the event, just like everyone else's, so clients need no special case.
- **`WHO` uses the RFC format.** PLAN.md section 5 explains why, and how the GUI still gets room counts from `LOOK`.
- **No separate chat limit:** the command limit (10 per second, Step 9) already caps chat frequency, which covers the RFC's recommendation.

---

## Step 8: GROUP

### `internal/game/group.go`

```go
package game

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charbelbahry/tap/internal/protocol"
)

func (g *Game) group(p *Player, args string) Result {
	sub, arg, _ := strings.Cut(args, " ")
	arg = strings.TrimSpace(arg)

	switch strings.ToUpper(sub) {
	case "CREATE":
		if p.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		g.nextGroup++
		p.Group = &Group{ID: fmt.Sprintf("grp.%d", g.nextGroup), Members: []string{p.Name}}
		return Result{Reply: "OK group=" + p.Group.ID}

	case "INVITE":
		if p.Group == nil {
			return Result{Reply: protocol.ErrNotInGroup}
		}
		target := g.players[arg]
		if target == nil {
			return Result{Reply: protocol.ErrPlayerNotFound}
		}
		if target.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		target.Invite = p.Group
		leader := p.Group.Members[0]
		return Result{Reply: protocol.OK, Events: []Event{
			{To: []string{target.Name}, Line: "EVT GROUP INVITE " + leader},
		}}

	case "JOIN":
		if p.Group != nil {
			return Result{Reply: protocol.ErrAlreadyInGroup}
		}
		grp := p.Invite
		if grp == nil || len(grp.Members) == 0 || grp.Members[0] != arg {
			return Result{Reply: protocol.ErrNotInvited}
		}
		grp.Members = append(grp.Members, p.Name)
		p.Group, p.Invite = grp, nil
		return Result{Reply: "OK group=" + grp.ID, Events: []Event{
			{To: grp.Members, Line: "EVT GROUP JOIN " + p.Name},
		}}

	case "LEAVE":
		if p.Group == nil {
			return Result{Reply: protocol.ErrNotInGroup}
		}
		return Result{Reply: protocol.OK, Events: g.leaveGroup(p)}
	}
	return Result{Reply: protocol.ErrBadRequest}
}

// leaveGroup removes p from its group. Used by GROUP LEAVE and Disconnect.
func (g *Game) leaveGroup(p *Player) []Event {
	grp := p.Group
	if grp == nil {
		return nil
	}
	grp.Members = slices.DeleteFunc(grp.Members, func(n string) bool { return n == p.Name })
	p.Group = nil
	return []Event{{To: grp.Members, Line: "EVT GROUP LEAVE " + p.Name}}
}

// groupMembersInRoom is used by Role B for shared kill credit.
func (g *Game) groupMembersInRoom(p *Player) []*Player {
	if p.Group == nil {
		return []*Player{p}
	}
	var out []*Player
	for _, name := range p.Group.Members {
		if m := g.players[name]; m != nil && m.Room == p.Room {
			out = append(out, m)
		}
	}
	return out
}
```

**Why:**

- **Groups are reached only through players, so there's no global group list.** An empty group simply stops being referenced and Go's garbage collector frees it.
- **`Members[0]` is the leader,** so leadership passes to the next member automatically when the leader leaves. No extra code is needed.
- **The invite event carries the leader's name,** whoever sent the invite. That's what `GROUP JOIN <leader-name>` in the RFC expects.
- **A player keeps only their last invite.** That's enough for the game, and much simpler than a list of pending invites.

---

## Step 9: Abuse detection and limits

### `internal/server/abuse.go`

```go
package server

import "time"

// window counts events in a fixed time period.
type window struct {
	start time.Time
	n     int
}

// hit records one event and returns the count in the current period.
func (w *window) hit(period time.Duration) int {
	if time.Since(w.start) > period {
		w.start, w.n = time.Now(), 0
	}
	w.n++
	return w.n
}

// admit counts a new connection and reports whether there is room for it.
func (s *Server) admit(ip string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	w := s.recent[ip]
	if w == nil {
		w = &window{}
		s.recent[ip] = w
	}
	if w.hit(time.Minute) == rapidConnLimit+1 {
		s.log.Warn("rapid_connections", "ip", ip)
	}
	if s.conns >= maxClients {
		s.log.Warn("server_full", "ip", ip)
		return false
	}
	s.conns++
	s.log.Info("connect", "ip", ip)
	return true
}
```

**Why:**

- **A fixed window** (count resets every second or minute) is the simplest rate limit there is. It's less precise than a sliding window, which doesn't matter for spotting abuse.
- **Warnings are logged once per window,** at the moment the limit is passed, so a flood doesn't also flood the log.
- **The subject asks to monitor and log rapid connections, so we only warn.** Blocking IPs would risk locking out evaluators testing with several clients from one machine.
- **The connection limit uses the RFC's own `900 CONNECTION_FAILED`.**

---

## Step 10: Logging

You've already written most of the logging in the steps above. This step makes sure it's complete.

**Setup:** one line in `main.go`, which you have from Step 2:

```go
log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
```

This prints one JSON object per line with `time`, `level` and `msg`, plus the fields you add:

```json
{"time":"2026-11-11T14:02:31.512+01:00","level":"INFO","msg":"command","ip":"127.0.0.1","player":"alice","cmd":"CHAT","args":"GLOBAL hello"}
```

**What you log** (Role B logs the game's world changes through the same logger, passed into `game.New`):

| `msg` | Level | When | Subject requirement |
|---|---|---|---|
| `listening` | INFO | Server starts | - |
| `world_invalid` | ERROR | World file fails validation | - |
| `connect` | INFO | New connection, with IP | Connections with timestamps and IPs |
| `disconnect` | INFO | Connection ends, with IP and player | Disconnections |
| `command` | INFO | Every line received | Every command with player name and parameters |
| `reply` | INFO, or WARN for `ERR` | Every reply | Server responses and error codes |
| `event` | INFO | Every event, with recipient count | Broadcasts |
| `flood` | WARN | Over 10 commands in a second | Command flooding |
| `rapid_connections` | WARN | Over 10 connections in a minute from one IP | Rapid connections |
| `server_full` | WARN | Connection refused at 100 | Abuse and limits |
| `line_too_long` | WARN | Line over 1024 bytes | Abuse and malformed input |
| `server_stopped` | ERROR | Listener fails | Errors |

**Why:**

- **`log/slog` ships with Go** and gives JSON, levels and precise timestamps with zero code of our own.
- **Writing to stdout** follows the usual practice for servers: whoever runs the server decides where output goes. To keep a file, run `make run-server | tee server.log`. To read it, use `jq`. To spot abuse, filter with `jq 'select(.level == "WARN")'`.
- **Performance is fine:** one small write per line, for a server with a handful of players.

---

## Step 11: CLI client

We chose the subject's first CLI option: the user types raw RFC commands, which the CLI sends as-is.

### `cmd/cli/main.go`

```go
// Command cli is a terminal client for TAP. Type RFC commands such as LOOK or MOVE north.
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
```

**Why:**

- **Two flows, two goroutines:** one reads the keyboard, one reads the server. That's all "keeps receiving events while waiting for user input" needs.
- **`readline`** redraws your half-typed line after printing an event above it. Plain `fmt.Scanln` would mix incoming events into whatever you're typing.
- **Typing `QUIT` needs no special code.** The server replies `OK bye` and closes, the reader loop ends, and the program exits.
- **Raw RFC commands** make the CLI a perfect test tool for any group's server, and keep the code at about 50 lines. State this choice in the README, as the subject requires.

**Tip:** on Windows, run it in Windows Terminal or PowerShell. Line editing may not work well in Git Bash's default terminal.

---

## Step 12: GUI bridge

Browsers can't open raw TCP connections, so this small Go program sits in between. Each browser tab gets its own TCP connection to the game server.

### `internal/gui/bridge.go`

```go
// Package gui serves the web client and bridges its WebSocket to the TAP server.
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

// Run serves the page on listen and connects each tab to the TAP server.
func Run(listen, server string) error {
	static, err := fs.Sub(webFiles, "web")
	if err != nil {
		return err
	}
	http.Handle("/", http.FileServerFS(static))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		bridge(w, r, server)
	})
	return http.ListenAndServe(listen, nil)
}

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
```

### `cmd/gui/main.go`

```go
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
```

**Why:**

- **The bridge doesn't understand the protocol at all.** It only moves lines, so all GUI logic stays in Role C's JavaScript, and any group's server works.
- **`go:embed`** packs the page into the binary, so `go run ./cmd/gui` works from any folder without copying files.
- **Listening on `127.0.0.1` only** means nobody else on the network can use your bridge.
- **gorilla/websocket allows one reader and one writer at a time,** and the code respects that: one goroutine only writes, the other only reads. `Close` is safe to call from both.
- **When either side hangs up, both sides close.** The browser tab then sees the WebSocket close and can show "disconnected".

**Day 1 note:** `go:embed web` needs at least one file in `internal/gui/web/`, so commit a placeholder `index.html` on Day 1.

---

## Step 13: Tests

Write a few focused tests instead of many trivial ones. Run them with `go test -race ./...`.

### Protocol (table test)

`internal/protocol/protocol_test.go`: check `Split` on `"look"`, `"chat ROOM  hi there"` and `"  MOVE north  "`. Check that `Valid` rejects `"hi\x07"` and invalid UTF-8, and accepts `"CHAT GLOBAL héllo 🙂"`.

### Server (real TCP on a random port)

`internal/server/server_test.go`:

```go
func startServer(t *testing.T) string {
	t.Helper()
	w, err := world.Load("../../data/world.yaml")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.DiscardHandler)
	go New(game.New(w, log), log).Serve(ln)
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String()
}

// dial connects and reads the greeting.
func dial(t *testing.T, addr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	r := bufio.NewReader(conn)
	expect(t, conn, r, "OK hello proto=1")
	return conn, r
}

func expect(t *testing.T, conn net.Conn, r *bufio.Reader, want string) {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	got, err := r.ReadString('\n')
	if err != nil || strings.TrimSuffix(got, "\n") != want {
		t.Fatalf("got %q (err %v), want %q", got, err, want)
	}
}
```

**Tests worth writing:**

| Test | What it proves |
|---|---|
| Connect, then a duplicate name | `OK connected`, then `ERR 201 NAME_IN_USE` |
| `LOOK` before `CONNECT` | `ERR 403 NOT_AUTHENTICATED` |
| Write `"CONN"`, pause, then write `"ECT alice\n"` | Fragmented commands work |
| Write `"WHO\nWHO\n"` in one call | Coalesced commands work |
| Alice sends `CHAT GLOBAL hi` | Both clients receive `EVT GLOBAL CHAT alice hi` |
| Bob's connection closes | Alice receives `EVT ROOM PRESENCE LEAVE bob` and `EVT STATS players=1`, and the name `bob` works again |
| Group: create, invite, join, chat, leave | Each reply and event, and errors `401`, `402` and `407` |

**Manual checks:** connect with the CLI or with `nc localhost 4242` (on Windows, `ncat` from the Nmap installer works). Kill a client while others chat, send 20 commands quickly, and watch the server log.

---

## Step 14: README sections

You write two sections of the README.

### Architecture

Cover these points, using the picture at the top of this file:

- **Command handling:** inline handling with a `switch` instead of a router, and why that's enough for 15 commands.
- **Concurrency model:** one read and one write goroutine per client, buffered outgoing channels, and one server-wide mutex around game calls.
- **Separation:** the server owns sockets and the game owns rules, connected by the `Result` and `Event` types.
- **Disconnects:** the cleanup path, and why state is removed before leave events go out.
- **Framing:** how `bufio.Scanner` handles fragmentation, coalescing and the 1024-byte limit.

### Server Logging

Cover these points:

- **Format:** JSON lines from `log/slog`, with an example line.
- **Events:** the `msg` table from Step 10.
- **Output:** stdout, and how to save it to a file with `tee`.
- **Monitoring:** how to filter warnings with `jq`.
- **Abuse detection:** the limits, and how flood and rapid-connection warnings work.

---

## What we deliberately skip

Each of these was left out because the subject doesn't need it. If an evaluator asks, this is the answer.

| Skipped | Why |
|---|---|
| Graceful shutdown with signal handling | No persistence is required, so Ctrl+C losing state is expected |
| `context.Context` everywhere | Nothing needs cancelling beyond closing a socket |
| Interfaces for testing | Tests use a real server on a random port, which is simpler and tests more |
| A router or command registry | A `switch` over 15 commands is clearer |
| Per-room or per-player locks | One lock is simpler, keeps event order correct, and is fast enough |
| Config files | Two flags per program are enough |
| Sliding-window rate limits | Fixed windows are enough to spot abuse |
| Request IDs or other protocol changes | The RFC has none; replies arrive in order instead |
| Reconnect logic in clients | The user can just start the client again |

---

## Evaluation prep

Be ready to answer these about your own code:

1. **How does a line travel from a client to the game and back?** Scanner, `handleLine`, `Handle`, `reply`, `send`, channel, `writeLoop`.
2. **What happens when a client disconnects in the middle of a broadcast?** `send` never blocks, a full buffer closes that socket, and its read loop runs `cleanup`.
3. **How do you guarantee `OK bye` arrives before the socket closes?** The socket is closed by `writeLoop` only after the channel is drained.
4. **Why one lock?** It's simple, it keeps event order correct, and it's fast enough.
5. **How are fragmented and merged packets handled?** `bufio.Scanner`.
6. **Why can't names have spaces?** Events are split on spaces.
7. **Where does a new command go?** A new `case` in `Handle`, plus a handler function.

**Likely live changes to practise** (each should take a few minutes):

- Change the rate limit from 10 to 5 per second.
- Add a field to every log line, such as the client's port.
- Make `WHO` also list names.
- Change the maximum name length.
