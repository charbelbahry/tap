*This project has been created as part of the 42 curriculum by <login1>, <login2>, <login3>.*

# TAP: The Last Vigil

> Work in progress. Each section below names its owner. Fill it in as your part is finished, and delete this note before submission.

## Description

TAP: The Last Vigil is a small multiplayer text adventure (a MUD) written in Go. It follows RFC 42TAP, a line-based protocol over TCP. Players connect to one shared server, explore rooms, chat, pick up items, fight enemies, form groups and complete quests in real time.

The project has four parts:

| Part | What it does |
|---|---|
| Server (`cmd/server`) | Holds the world and all players, answers every RFC command, sends events, and writes JSON logs |
| CLI client (`cmd/cli`) | Plays the game by typing raw RFC commands in a terminal, with live events |
| GUI client (`cmd/gui`) | Plays the same game in a pixel-art web page in the browser |
| World file (`data/world.yaml`) | Rooms, items, NPCs and quests |

Both clients talk to the same server, so a CLI player and a GUI player can meet in the same room. They also work with other groups' servers.

**The story.** King Aurenth of Morvane feared death, so he rang a great bell to ask the abyss how his kingdom could never die. Something answered. Now nothing in Morvane can truly die or truly live. You are a Vigilant, a corpse woken by the last candle in the Votive Chapel, and you must silence the bell.

## Instructions

**Requirements:** Go 1.27 or newer, and `make` (optional, see below). Linux or macOS is recommended. On Windows, use Windows Terminal or PowerShell, because line editing in the CLI may not work in Git Bash.

**Quickest way to play:**

```sh
git clone <repo-url> tap && cd tap
make install          # download dependencies and the linter
make run-server       # terminal 1: start the server on port 4242
make run-client       # terminal 2: play in the terminal
make run-client-gui   # or: play in the browser at http://127.0.0.1:8080
```

In the CLI, type `CONNECT <name>` first, then any command, for example `LOOK`, `MOVE east` or `CHAT GLOBAL hello`. In the GUI, enter a name and use the buttons.

## Building and Running

| `make` target | Without `make` | Does |
|---|---|---|
| `make install` | `go mod download` | Downloads dependencies and installs the pinned linter |
| `make build` | `go build -o bin/ ./cmd/...` | Builds the server, CLI and GUI into `bin/` |
| `make run-server` | `go run ./cmd/server` | Starts the server |
| `make run-client` | `go run ./cmd/cli` | Starts the CLI client |
| `make run-client-gui` | `go run ./cmd/gui` | Starts the GUI, then open `http://127.0.0.1:8080` |
| `make lint` | `golangci-lint run ./...` | Runs the linter (includes `go vet` and `staticcheck`) |
| `make test` | `go test -race ./...` | Runs all tests with the race detector |
| `make fmt` | `gofmt -w .` | Formats all Go files |
| `make clean` | delete `bin/` | Removes build output |

The race detector needs a C compiler. If Windows has none, run `go test ./...` there and `-race` on Ubuntu.

**Flags**

| Program | Flag | Default |
|---|---|---|
| Server | `-port` | `4242` |
| Server | `-world` | `data/world.yaml` |
| CLI | `-addr` | `localhost:4242` |
| GUI | `-port` (web page) | `8080` |
| GUI | `-server` (TAP server) | `localhost:4242` |

The `make` targets accept the same settings as variables, for example `make run-server PORT=5000` or `make run-client-gui SERVER=10.0.0.5:4242`.

**CLI choice: raw RFC commands.** The CLI sends exactly what you type, such as `MOVE east`, and prints every line the server sends, including events as they arrive. We chose this because it works unchanged with any group's server, it shows the protocol directly (useful for testing and evaluation), and it stays very small.

**GUI toolkit.** The GUI is web-based. A small Go program serves the page and bridges the browser's WebSocket to the server's TCP connection, one WebSocket message per protocol line. JavaScript is only the display layer of this web toolkit.

## Architecture

<!-- Owner: Role A. Inline switch instead of a router, goroutines per client, the single lock, disconnect cleanup, framing. -->

## Protocol Implementation

We follow RFC 42TAP. This section lists every place where we chose between conflicting rules, filled a gap, or added something.

**Subject vs RFC**

| Topic | RFC | Subject example | Our choice |
|---|---|---|---|
| `WHO` reply | `OK players=5` | `OK {"room":[...],"server":5}` | The server sends the RFC form. Room players come from `LOOK`. |
| `TALK` reply | `OK Stay safe.` | `OK {"npc":"guard","dialogue":"..."}` | The server sends the RFC form. |

The subject asks for strict RFC adherence and interchangeable clients, and its snippets are labelled as examples. **Both our clients accept both forms**, so they work with servers that chose differently.

**Gaps in the RFC**

| Problem | Our choice |
|---|---|
| The ABNF `1*VCHAR` forbids spaces and Unicode, but the prose requires both | We follow the prose |
| `CONNECT` uses CRLF in the ABNF, everything else uses LF | The server sends LF and strips a trailing `\r` from input |
| `EVT STATS players=N` has no type field | Sent as written; clients handle it as a special case |
| Replies have no request ID | The server replies in order; the GUI keeps a queue of sent commands to match each reply |
| `LOOK` returns IDs only | The GUI derives a label from the ID (`item.pilgrims_candle` becomes "Pilgrims Candle") |

**Extensions** (additions only; clients ignore unknown events)

| Kind | Addition | Why |
|---|---|---|
| Error | `400 BAD_REQUEST` | Unknown command, missing arguments, bad scope, invalid name or bad input |
| Error | `403 NOT_AUTHENTICATED` | Any command before `CONNECT` |
| Error | `402 ALREADY_CONNECTED` | A second `CONNECT` |
| Error | `404 PLAYER_NOT_FOUND` | Inviting an unknown player |
| Error | `407 NOT_INVITED` | `GROUP JOIN` without an invite |
| Error | `408 NOT_IN_COMBAT` | `DEFEND` or `FLEE` outside combat |
| Error | `429 RATE_LIMITED` | Too many commands |
| Error | `900 CONNECTION_FAILED` (from the RFC) | Used when the server is full |
| Event | `EVT ROOM COMBAT <name> <npc-id> {"damage":10,"target_hp":15}` | The subject requires combat results to be broadcast |
| Event | `EVT ROOM ITEM TAKE <name> <item-id>` and `... DROP ...` | The GUI must show item changes live |
| Command | `DEFEND` → `OK {"hp":80,"damage_taken":5,"status":"combat"}` | Turn-based combat |
| Command | `FLEE` → `OK room=<room-id>` | Turn-based combat |

**Who receives each event**

| Event | Sent to |
|---|---|
| `ROOM PRESENCE ENTER` / `LEAVE` | Others in that room, not the player moving |
| `ROOM CHAT` | Everyone in the room, including the sender |
| `GLOBAL CHAT` | Everyone, including the sender |
| `GROUP CHAT`, `JOIN`, `LEAVE` | Group members |
| `GROUP INVITE <leader>` | The invited player |
| `STATS players=N` | Everyone, on each connect and disconnect |
| `ROOM COMBAT`, `ROOM ITEM` | Everyone in the room |

A reply is always sent before the events it causes.

**Limits:** lines up to 1024 bytes; usernames of 1 to 32 characters with Unicode allowed and no whitespace; 100 connections; more than 10 commands per second returns `429`.

## Combat System

<!-- Owner: Role B. Turn order, damage formulas, DEFEND, FLEE, death and respawn, enemy respawn. -->

## Quest System

<!-- Owner: Role B. QUEST flow (accept, check, complete), progress tracking, prerequisites, rewards. -->

## World Design

<!-- Owner: Role B. Story, map with the loop and branches, NPC roles, item placement. -->

## Server Logging

<!-- Owner: Role A. JSON format, event table, output streams, monitoring with jq, abuse detection. -->

## Testing

**Automated tests**

```sh
make test    # go test -race ./...
make lint
```

**Talking to the server by hand.** Any raw TCP tool works, for example `nc 127.0.0.1 4242`. The server greets with `OK hello proto=1`; then type `CONNECT alice`, `LOOK`, and so on.

**Multiplayer.** Start the server, then open several clients at once: two terminals with `make run-client`, and one or more browser tabs on the GUI. Connect with different names and check that:

- each player sees the others enter and leave the room;
- room chat only reaches players in that room, and global chat reaches everyone;
- when one player takes an item, the others can no longer take it, and see it disappear;
- closing a client drops its items in the room and updates the player count for everyone.

**Combat.** Walk to a room with an enemy, then `ATTACK <npc>` until it dies. Also check `DEFEND`, `FLEE`, attacking a friendly NPC (`ERR 405`), and dying (you return to the Votive Chapel with reduced HP). Two players attacking the same enemy should both see the results.

**Quests.** Use `QUEST <npc>` to accept, then check progress with `QUESTS`. Finish a delivery quest and a defeat quest, then ask again to get `ERR 406`.

**Other groups' servers.** Run our CLI or GUI with `-addr` / `-server` pointing at another group's server. Unknown rooms, NPCs and items show a generic picture in the GUI.

## Group Contributions

<!-- Everyone. Each member's responsibilities and what they built. -->

## Resources

<!-- Everyone. References (RFC 2119, RFC 5234, Go docs, MUD history...) and how AI was used: which tasks and which parts of the project. -->

- **GUI art:** every image in `internal/gui/web/assets/` (rooms, characters, items and the player) was generated with Google Gemini from our own descriptions, then scaled down to its native pixel size and reduced in colours.
