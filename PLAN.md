# TAP: The Last Vigil - Project Plan

A multiplayer dark fantasy text adventure in Go, built for the 42 TAP project and RFC 42TAP.

**Team:** 3 people. **Language:** Go. **Goal:** finish quickly, stay simple, and comply fully with the subject and RFC.

**Contents**

1. [Project explanation](#1-project-explanation)
2. [Implementation plan](#2-implementation-plan)
3. [Folder structure](#3-folder-structure)
4. [Packages](#4-packages)
5. [Protocol decisions](#5-protocol-decisions)
6. [Game rules](#6-game-rules)
7. [Story and world](#7-story-and-world)
8. [Roles](#8-roles)
9. [Working simultaneously](#9-working-simultaneously)
10. [Compliance checklist](#10-compliance-checklist)
11. [README requirements](#11-readme-requirements)

---

## 1. Project explanation

TAP is a small MUD (Multi-User Dungeon): an online game whose world is made of text. Players connect to one shared server, explore rooms, chat, pick up items, fight enemies, form groups and complete quests in real time.

We deliver four things:

| Part | What it does |
|---|---|
| **Server** | Holds the world and all players, implements every RFC command and event, and writes JSON logs |
| **CLI client** | Plays the game by typing commands in a terminal |
| **GUI client** | Plays the same game through a pixel-art web page with panels and buttons |
| **World file** | YAML file defining rooms, items, NPCs and quests |

The CLI and GUI are both ways to play. They connect to the same server, so a CLI player and a GUI player can meet in the same room. Server logs appear in the server's own output, not in a client.

**Hard requirements**

- We use Go, one of the allowed languages. Using Python is an automatic failure. The web GUI's JavaScript is the display layer of the "web-based" toolkit the subject allows, and the README must say so.
- TCP, UTF-8, one message per line, ending in `\n`.
- Our clients must work with other groups' servers, and our server with other groups' clients.
- The world needs at least 8 rooms with a loop and a branch, 3 NPC roles, 4 items (2 obtainable) and 2 quests.
- Any deviation from the RFC must be documented in the README.
- Every member must have commits and be able to explain the project. A live code change may be requested during evaluation.

---

## 2. Implementation plan

### Architecture

```
Terminal: CLI client  ------------- TCP -------------+
                                                     v
Browser: page <-- WebSocket --> Go bridge -- TCP --> Game server --> JSON logs
                                                     |
                                                     v
                                              data/world.yaml
```

### Design choices

| Area | Choice | Why |
|---|---|---|
| Concurrency | One read goroutine and one write goroutine per client, and **one global mutex** around game state | Simple and hard to get wrong |
| Slow or dead clients | Each client has a buffered outgoing channel (256 lines). If it's full, that client is disconnected | A broadcast can never get stuck on one client |
| Commands | Plain `switch` on the command name | The subject allows inline handling |
| Framing | `bufio.Scanner` with a 1024-byte limit | Handles split and merged TCP packets |
| Game logic | Pure Go with no networking | Testable without a server, and lets people work separately |
| Logging | Standard library `log/slog` with JSON output to stdout | Gives levels and timestamps for free |
| World data | One YAML file, validated at startup; the server exits on any bad reference | Comments are allowed, and tuning needs no code change |
| GUI | One HTML/CSS/JS page embedded in the Go binary, with no JS framework or build step | No C compiler, OpenGL or `npm` needed; runs the same on Ubuntu and Windows |
| GUI connection | Each browser tab opens one WebSocket, and the bridge opens one TCP connection for it | Two tabs means two players, which makes testing easy |

### The contract between server and game

We agree this on Day 1. It's what allows independent work:

```go
package game

// Event is one line to push to a set of players.
type Event struct {
    To   []string // player names
    Line string   // full line, such as "EVT ROOM CHAT alice hello"
}

// Result is what a command produces.
// The server sends Reply first, then Events.
type Result struct {
    Reply  string
    Events []Event
    Close  bool // true after QUIT
}

func (g *Game) Connect(name string) Result              // validates the name, spawns the player
func (g *Game) Handle(player, cmd, args string) Result  // runs one command
func (g *Game) Disconnect(player string) []Event        // removes state first, then builds leave events
```

The server deals with sockets, and the game deals with rules. Neither reaches into the other. The server calls these methods one at a time under a single lock, so game code needs no locking of its own.

### Phases

The project has no fixed deadline, so these are rough targets.

| Phase | When | Who | Output |
|---|---|---|---|
| Kickoff | Day 1 | Everyone | Repo, skeleton, contract, stub `Handle`, agreed protocol decisions |
| Build | Days 2-4 | Each person alone | Their role's features, merged often |
| Integration | Day 5 | Everyone | Full game running, checklist tested, bugs fixed, balance tuned |
| Finish | Day 6 | Everyone | README complete, everyone can explain each part |

---

## 3. Folder structure

This follows standard Go layout: `cmd/` for executables, and `internal/` for private packages.

```
tap/
├── cmd/
│   ├── server/main.go          # Starts the server (flags: -port, -world)
│   ├── cli/main.go             # CLI client (flags: -addr)
│   └── gui/main.go             # Starts the GUI bridge (flags: -port, -server)
│
├── internal/
│   ├── protocol/               # Shared by server and game
│   │   ├── errors.go           # Error replies such as ErrNoExit = "ERR 301 NO_EXIT"
│   │   └── parse.go            # Splits a line into command and arguments
│   │
│   ├── server/                 # TCP layer: no game rules here
│   │   ├── server.go           # Accept loop, connection limit
│   │   ├── client.go           # Read and write goroutines per client
│   │   ├── abuse.go            # Flood and rapid-connection detection
│   │   └── *_test.go
│   │
│   ├── game/                   # Game rules: no networking here
│   │   ├── game.go             # Game struct, mutex, Handle switch (shared file)
│   │   ├── session.go          # Connect, Disconnect, QUIT
│   │   ├── chat.go             # CHAT, WHO
│   │   ├── group.go            # GROUP CREATE, INVITE, JOIN, LEAVE
│   │   ├── room.go             # LOOK, MOVE
│   │   ├── item.go             # TAKE, DROP, INVENTORY, name matching
│   │   ├── npc.go              # TALK, NPC respawn
│   │   ├── combat.go           # ATTACK, DEFEND, FLEE, STATUS
│   │   ├── quest.go            # QUEST, QUESTS, rewards
│   │   └── *_test.go
│   │
│   ├── world/                  # Loads and validates world data
│   │   ├── world.go            # Structs and YAML loading
│   │   ├── validate.go         # Checks exits and all references
│   │   └── *_test.go
│   │
│   └── gui/                    # Web client
│       ├── bridge.go           # Serves the page, WebSocket <-> TCP
│       └── web/                # Embedded with go:embed
│           ├── index.html
│           ├── style.css       # Pixel components
│           ├── app.js          # Protocol handling and rendering
│           └── assets/         # Fonts, sprites, room art
│
├── data/
│   └── world.yaml              # All rooms, items, NPCs and quests
│
├── go.mod
├── go.sum
├── Makefile
├── .gitignore
└── README.md
```

**File ownership rule:** each file has one owner (see [Roles](#8-roles)). `game.go` is the only shared game file, and its `switch` is written fully on Day 1 so nobody needs to edit it later.

### Build commands

| Command | Does | Without `make` (such as on Windows) |
|---|---|---|
| `make install` | Downloads dependencies and the linter | `go mod download` |
| `make run-server` | Starts the server | `go run ./cmd/server` |
| `make run-client` | Starts the CLI | `go run ./cmd/cli` |
| `make run-client-gui` | Starts the GUI at `http://127.0.0.1:8080` | `go run ./cmd/gui` |
| `make lint` | Runs `golangci-lint`, including `gofmt` and `go vet` checks | `golangci-lint run` |
| `make test` | Runs tests with the race detector | `go test -race ./...` |
| `make clean` | Removes built binaries | Delete `bin/` |

The race detector needs a C compiler. If one isn't installed on Windows, use `go test ./...` there and run `-race` on Ubuntu.

---

## 4. Packages

The subject permits appropriate libraries, as long as we can explain them.

| Package | Used for | Owner |
|---|---|---|
| `gopkg.in/yaml.v3` | Loading `world.yaml` | B |
| `github.com/gorilla/websocket` | Browser-to-bridge connection | A |
| `github.com/chzyer/readline` | CLI input that stays intact when events arrive | A |
| `golangci-lint` (tool) | `make lint` | C |

**Standard library covers the rest:** `net`, `bufio`, `log/slog`, `encoding/json`, `embed`, `net/http`, `sync`, `unicode/utf8`, `testing`.

**Frontend:** plain HTML, CSS and JavaScript. Pixel fonts such as Jacquard 12, Pixelify Sans, Silkscreen and VT323 are saved in `web/assets/` instead of loaded from the internet, since evaluation machines may not have reliable access. Check each font's license when adding it.

**Avoid:**

- MUD or game frameworks, because the server is the point of the project
- React or other frameworks needing a build step
- CDN links
- Fyne and other GUI toolkits needing a C compiler and OpenGL

---

## 5. Protocol decisions

The RFC is our reference, and every item here goes in the README's "Protocol Implementation" section.

### Conflicts between the subject and the RFC

| Topic | RFC | Subject example | Our decision |
|---|---|---|---|
| WHO | `OK players=5` | `OK {"room":[...],"server":5}` | Server sends the RFC form. Room players come from `LOOK`, and the server count from `WHO` and `EVT STATS` |
| TALK | `OK Stay safe.` | `OK {"npc":"guard","dialogue":"..."}` | Server sends the RFC form. The GUI already knows which NPC it addressed |

**Why:** the subject requires strict RFC adherence and interchangeable clients, while its snippets are labelled examples. One reply line can't match both formats. **Our clients accept both**, so they also work with servers that chose differently.

### Other RFC problems

| Problem | Decision |
|---|---|
| ABNF `1*VCHAR` forbids spaces and Unicode, but the prose requires both | Follow the prose |
| CONNECT's ABNF uses CRLF, everything else uses LF | Server sends LF, and strips a trailing `\r` from input |
| `EVT STATS players=N` has no type field | Send as written; clients treat it as a special case |
| Replies have no request ID | Server replies in order; the GUI keeps a queue of sent commands to match each reply |
| LOOK returns IDs only | IDs are based on names, so the GUI can derive a label (`item.healing_potion` becomes "Healing Potion") |
| Section numbers in the RFC are duplicated | Refer to sections by title |

### Extensions we add

These are additions, not changes. Every client must **ignore unknown events** and handle unknown errors without crashing.

**Error codes**

| Code | Message | When |
|---|---|---|
| 400 | `BAD_REQUEST` | Unknown command, missing arguments, bad scope, invalid name, control characters, invalid UTF-8, or a line that's too long (this last case also closes the connection) |
| 403 | `NOT_AUTHENTICATED` | Any command before `CONNECT` |
| 402 | `ALREADY_CONNECTED` | A second `CONNECT` on one connection |
| 404 | `PLAYER_NOT_FOUND` | Inviting an unknown player |
| 407 | `NOT_INVITED` | `GROUP JOIN` without an invite |
| 408 | `NOT_IN_COMBAT` | `DEFEND` or `FLEE` outside combat |
| 429 | `RATE_LIMITED` | Too many commands or chat messages |

We use the RFC's `900 CONNECTION_FAILED` when the server is full, then close the connection.

**Events**

```
EVT ROOM COMBAT <username> <npc-id> {"damage":10,"target_hp":15}
EVT ROOM ITEM TAKE <username> <item-id>
EVT ROOM ITEM DROP <username> <item-id>
```

The subject requires combat results to be broadcast and the GUI to show live item changes, but the RFC has no event for either.

**Commands**

| Command | Reply |
|---|---|
| `DEFEND` | `OK {"hp":80,"damage_taken":5,"status":"combat"}` |
| `FLEE` | `OK room=<room-id>` |

### Who receives each event

| Event | Sent to |
|---|---|
| `ROOM PRESENCE ENTER` / `LEAVE` | Others in that room, not the player moving |
| `ROOM CHAT` | Everyone in the room, including the sender |
| `GLOBAL CHAT` | Everyone, including the sender |
| `GROUP CHAT`, `JOIN`, `LEAVE` | Group members |
| `GROUP INVITE <leader>` | The invited player |
| `STATS players=N` | Everyone, on each connect and disconnect |
| `ROOM COMBAT`, `ROOM ITEM` | Everyone in the room |

**Ordering:** the reply is always queued before the events it causes.

### Limits

These are starting values:

| Limit | Value |
|---|---|
| Line length | 1024 bytes |
| Username | 1-32 characters, Unicode allowed, no whitespace or control characters |
| Connections | 100 in total |
| Commands | Warn and return `429` above 10 per second |
| Chat | Covered by the command limit, so no separate chat limit |
| Rapid connections | Warn above 10 per minute from one IP |

Inventory size needs no separate limit, because only 8 unique items exist in the world.

---

## 6. Game rules

The owner of game logic (Role B) makes the final call on these. All numbers live in `world.yaml` where possible and should be tuned during play.

### Items

- Each item exists exactly once: in a room, or held by one player.
- Items match by full ID, short ID or display name, ignoring case (`item.vergers_hammer`, `vergers_hammer`, `Verger's Hammer`). The rest of the line is the argument, so multi-word names work.
- On disconnect, a player's items drop into their current room, so quest items can't be lost.
- Items can carry stat bonuses in the YAML (`damage_bonus`, `max_hp_bonus`) while held. **Role B can add more bonuses to make the game easier to beat.**

### Combat (turn-based)

- Players start at 100 HP.
- Each command is one turn, with no timers. The player acts first, then the NPC hits back, and simultaneous attacks are ordered by which the server receives first.
- **Damage:** `8 + rand(0..4) + bonuses`. NPC damage comes from its YAML stats.
- **Shared fights:** enemy HP is shared between attackers, and an enemy hits back at whoever attacked it.
- **`DEFEND`:** the NPC's hit that turn is halved.
- **`FLEE`:** moves through a random exit, with a 50% chance of taking a hit.
- **`MOVE` in combat:** allowed, but the NPC gets a free hit. Players on another group's client, with no `FLEE` button, can therefore never get stuck.
- **Death:** at 0 HP, respawn in the Votive Chapel with 50 HP, keeping items.
- **Healing:** entering the Votive Chapel normally restores full HP.
- **Enemy respawn:** dead enemies return after 45 seconds, checked when the room is read, so no timers are needed.

### Quests

`QUEST <npc>` does everything, so no new commands are needed:

1. **First call:** accepts the quest and returns `status: available`.
2. **Objective not met:** returns `status: active`.
3. **Objective met:** completes it, gives the reward and returns `status: completed`.
4. **Nothing left for you:** returns `ERR 406 NO_QUEST_AVAILABLE`.

More rules:

- Progress is per player.
- Kill credit goes to group members in the same room.
- Delivered items return to their starting room, so others can do the quest.
- A quest with a `needs` field stays hidden until the required quest is complete.
- A reward item moves to the player unless another player is carrying it. In that case, the quest still completes and the carrier must hand it over with `DROP` and `TAKE`.

### Groups

- Any member can invite. The invite event carries the leader's name, so `GROUP JOIN <leader>` works directly.
- Joining requires an invite.
- If the leader leaves, the next member becomes leader. An empty group is removed.

### STATUS values

| Value | Meaning |
|---|---|
| `combat` | Currently fighting |
| `healthy` | At or above 60% HP |
| `wounded` | Below 60% HP |

---

## 7. Story and world

The mood is inspired by dark fantasy games such as Dark Souls, but the world, names and lore are our own.

### Premise

King Aurenth of Morvane feared death. He ordered a great bell cast and rung to ask the abyss one question: *how can my kingdom never die?*

Something answered. Now nothing in Morvane can truly die, or truly live. The sun stopped rising, and the dead still walk.

You are a **Vigilant**, a corpse woken by the last candle in the Votive Chapel. Every time you fall, its flame drags you back. To end this, you must silence the bell.

### The hidden truth

The bell never reached the abyss. It woke **Baal**, buried beneath Morvane. The kingdom's curse is Baal's answer, and only by taking the question back to Baal can it be undone.

### Map

Eleven rooms. The outer ring is the full-circuit loop. The Bell Tower is an optional branch, and the Sunken Stair leads to the secret area.

```
                                      [Bell Tower]
                                           |
[Sunken Stair] --- [Ossuary] ------- [Throne Ruin] ------- [Burned Library]
      | (down)         |                                          |
[Hall of the       [Crypt Gate]                            [Hanged Garden]
  Answer]              |                                          |
                   [Votive Chapel] -- [Pilgrim's Road] ---- [Silent Market]
                     (safe room)
```

**Loop:** Votive Chapel → Pilgrim's Road → Silent Market → Hanged Garden → Burned Library → Throne Ruin → Ossuary → Crypt Gate → Votive Chapel.

| Room | Description |
|---|---|
| **Votive Chapel** | Candles burn without wax beneath a roof that fell in long ago. Nothing hostile crosses this threshold. |
| **Pilgrim's Road** | Grave markers line a road to nowhere. A sentry still walks its post, though most of it has rotted away. |
| **Silent Market** | Empty stalls under torn awnings. It would be silent, if not for Zaza. |
| **Hanged Garden** | Black trees bow under the weight of what hangs from them. Bread rots on a funeral table. |
| **Burned Library** | Ash falls from the shelves like slow snow. One shroud lies untouched among the cinders. |
| **Throne Ruin** | King Aurenth's throne stands empty. A knight still guards it, and a stone stair climbs north to the bell. |
| **Bell Tower** | The great bell hangs above you, ringing with no hand on the rope. Its Warden turns to face you. |
| **Ossuary** | Skulls fill the walls from floor to vault. A cold draught rises from a narrow gap to the west. |
| **Crypt Gate** | A rusted portcullis, half raised. Old Corvin keeps digging, though the ground stopped accepting bodies years ago. |
| **Sunken Stair** | Steps cut for something larger than a man spiral into black water. Writing covers every wall, the same question repeated. |
| **Hall of the Answer** | A vault older than Morvane. This is where the bell's call truly went, and what answered it is still here. |

### Characters

| NPC | Room | Role | Personality |
|---|---|---|---|
| **Sister Ilvane** | Votive Chapel | Quest-giver | Gentle and sorrowful; keeper of the last candle |
| **Old Corvin** | Crypt Gate | Quest-giver | Tired gravedigger with bleak humor |
| **Mute Merchant** | Silent Market | Dialogue | Hasn't spoken in ninety years; communicates in gestures |
| **Zaza** | Silent Market | Dialogue, quest-giver | The king's former herald. He read the question aloud, and the answer broke his mind. He rants endlessly and can't decide anything, and his rants hide clues to the Sunken Stair. |
| **Rotting Sentry** | Pilgrim's Road | Enemy | Still guarding a road nobody uses |
| **Grave Hound** | Ossuary | Enemy | Growls from somewhere behind its ribs |
| **Fallen Knight** | Throne Ruin | Enemy | Sworn to a king who no longer exists |
| **Bell Warden** | Bell Tower | Main boss | Guardian of the bell |
| **Baal** | Hall of the Answer | Bonus boss | The thing that answered. Calm, ancient and amused |

**Enemy stats (starting values)**

| Enemy | HP | Damage per hit |
|---|---|---|
| Rotting Sentry | 30 | 4-8 |
| Grave Hound | 40 | 6-10 |
| Fallen Knight | 80 | 8-12 |
| Bell Warden | 150 | 10-16 |
| Baal | 300 | 15-25 |

**Sample dialogue**

- **Sister Ilvane:** "Every time you fall, the candle pulls you back. It has never asked if you wanted that."
- **Old Corvin:** "Forty years digging and not one of them stays down."
- **Mute Merchant:** "(The merchant holds up a coin, studies it for a long time, and puts it away.)"
- **Zaza:** "Should I go to the tower? The tower is awful. Should I stay? Staying is awful. Should I sit? I tried sitting during the second eclipse. Terrible. Would not sit again."
- **Zaza:** "The merchant has not said a word in ninety years and still wins every argument. Ninety years! I was counting. I lost count. That is the merchant's fault too."
- **Zaza:** "There was something under the bell. Under it! And I read it the question out loud, and it answered, and now I cannot decide which boot goes first. Do NOT ask me about the stair."
- **Bell Warden:** "It has been asked. It will not be asked again."
- **Baal:** "All of you come to ask. I gave your king exactly what he wanted."

### Items

| Item | ID | Found | Effect |
|---|---|---|---|
| Pilgrim's Candle | `item.pilgrims_candle` | Votive Chapel | Flavor |
| Rusted Censer | `item.rusted_censer` | Silent Market | Flavor |
| Funeral Bread | `item.funeral_bread` | Hanged Garden | Flavor |
| Burial Shroud | `item.burial_shroud` | Burned Library | Needed for A Proper Burial |
| Herald's Scroll | `item.heralds_scroll` | Sunken Stair | Needed for What Did I Come Here For? |
| Sealed Altar | `item.sealed_altar` | Throne Ruin | Can't be taken (`obtainable: false`) |
| Verger's Hammer | `item.vergers_hammer` | Reward | +6 damage while carried |
| The Unspoken Question | `item.unspoken_question` | Reward | Makes Baal able to be wounded |

Role B can add stat bonuses to the flavor items to make the game easier.

### Quests

| # | Quest | Given by | Objective | Requires | Reward |
|---|---|---|---|---|---|
| 1 | A Road Without Watchers | Sister Ilvane | Destroy the Rotting Sentry | - | +2 damage |
| 2 | A Proper Burial | Old Corvin | Deliver the Burial Shroud | - | Verger's Hammer, +20 max HP |
| 3 | The Last Toll | Sister Ilvane | Kill the Bell Warden | Quest 1 | +5 damage, title "Unbound" |
| 4 | What Did I Come Here For? | Zaza | Bring the Herald's Scroll | Quest 3 | The Unspoken Question |
| 5 | The Unanswering | Zaza | Destroy Baal | Quest 4 | Title "The Unanswered", and the story ending |

### Boss design

**Bell Warden** can be beaten in two ways. These figures use average damage, come from calculation only, and need playtesting.

| Approach | Result |
|---|---|
| Solo, unprepared | About 182 damage taken against 100 HP: lose |
| Two players | About 91 damage each: narrow win |
| Solo with quests 1 and 2 | About 104 damage against 120 HP: narrow win |

**Baal** is the bonus boss:

- He can't be damaged unless a player carrying The Unspoken Question is in the room. Without it, hits do 0 damage and he still hits back.
- The bearer can stay in the room while others attack, so a group can plan around it.
- He should need about three fully prepared players.
- Defeating him ends the story: the sky lightens and the dead of Morvane finally lie down.

### World file format

```yaml
world:
  start: loc.votive_chapel
  safe_room: loc.votive_chapel
  rooms:
    loc.votive_chapel:
      name: "Votive Chapel"
      description: "Candles burn without wax..."
      exits: { north: loc.crypt_gate, east: loc.pilgrims_road }
      items: [item.pilgrims_candle]
      npcs: [npc.sister_ilvane]

items:
  item.vergers_hammer:
    name: "Verger's Hammer"
    description: "..."
    obtainable: true
    damage_bonus: 6

npcs:
  npc.bell_warden:
    name: "Bell Warden"
    role: enemy
    dialogue: ["It has been asked. It will not be asked again."]
    stats: { hp: 150, damage_min: 10, damage_max: 16 }

quests:
  last_toll:
    title: "The Last Toll"
    giver: npc.sister_ilvane
    description: "Kill the Bell Warden and silence the bell."
    objective: { kill: npc.bell_warden }
    needs: road_without_watchers
    reward: { damage_bonus: 5, title: "Unbound" }
```

---

## 8. Roles

Each role is a complete piece of the project, built against the protocol and contract instead of against each other's unfinished code.

### Role A: Network and social

**Assigned to:** `<login1>` (replace with the server owner's login)

| Area | Work |
|---|---|
| `internal/server` | TCP accept loop, read and write goroutines, line limit, UTF-8 and control character checks, clean disconnects |
| `internal/protocol` | Line parsing and error constants |
| `internal/game` (`session.go`, `chat.go`, `group.go`) | CONNECT, QUIT, CHAT, WHO, all GROUP commands, name uniqueness, presence and stats events |
| Logging | Connections with IP, commands, replies, errors, and flood and rapid-connection warnings |
| `cmd/cli` | CLI sending raw RFC commands with live events, about 60 lines |
| `internal/gui/bridge.go` | WebSocket to TCP bridge, about 100 lines |
| README | Architecture, Server Logging |

**Implementation steps**

1. Accept connections and send the greeting.
2. Add read and write goroutines, then parse lines and call `Handle`.
3. Make disconnects clean: stop the client, call `Disconnect`, broadcast the events.
4. Add session, chat and group commands.
5. Add logging, then abuse detection.
6. Write the CLI and bridge.

**Watch out for:** `OK bye` must be fully sent before the socket closes, and a client dying mid-broadcast mustn't affect the others.

### Role B: World and gameplay

**Assigned to:** to be confirmed

| Area | Work |
|---|---|
| `internal/world` | YAML loader and validator for exits, items, NPCs and quests |
| `data/world.yaml` | The complete world from [section 7](#7-story-and-world) |
| `internal/game` (`room.go`, `item.go`, `npc.go`, `combat.go`, `quest.go`) | LOOK, MOVE, TAKE, DROP, INVENTORY, TALK, ATTACK, DEFEND, FLEE, STATUS, QUEST, QUESTS |
| Tests | Unit tests for items, combat and quests |
| Balance | Damage, HP and item stat bonuses |
| Logging calls | World changes, combat results and quest progress, using the logger passed into `Game` |
| README | Combat System, Quest System, World Design |

**Implementation steps**

1. Define the structs, write the loader and validator, and build the world file.
2. Do LOOK and MOVE, then items and name matching.
3. Do TALK, then combat, then quests.
4. Playtest and tune.

**Watch out for:** two players taking the same item, items being duplicated, and items lost when a holder disconnects.

### Role C: GUI and compliance

**Assigned to:** to be confirmed

| Area | Work |
|---|---|
| `internal/gui/web/index.html` | Layout based on the mockup |
| `internal/gui/web/style.css` | Custom pixel components: buttons, panels, HP bars and tabs |
| `internal/gui/web/app.js` | WebSocket connection, pending command queue, rendering, all buttons |
| `internal/gui/web/assets/` | Fonts, plus AI-generated character sprites and room art |
| `Makefile` | All required targets |
| Compliance | Owns and runs the [checklist](#10-compliance-checklist), and reports bugs to A or B |
| README | Assembles the full README; writes Building and Running, Testing, Protocol Implementation |

**GUI must include** (from the subject)

- Room details, items, NPCs and exits, updating in real time
- Inventory, with Take and Drop buttons
- Item IDs and display names both handled
- Room view refreshing after TAKE and DROP
- Chat (Global, Room, Group) separate from the log view
- Buttons for LOOK, MOVE, TAKE, DROP, TALK, ATTACK, STATUS, QUEST, QUESTS, WHO, GROUP, QUIT
- Counters for players in the room and on the server
- NPC dialogue display

**Implementation steps**

1. Build the static layout and pixel components, using fake data.
2. Connect the WebSocket and display raw lines.
3. Add the pending queue and reply handling. Lines starting with `EVT` are events; anything else answers the oldest pending command.
4. Render room, inventory and chat, and connect every button.
5. Send `LOOK` after room events and every 5 seconds, so the room stays current on other groups' servers. Hide those automatic replies from the log.
6. Generate and add the art.

**Watch out for:** escape all text from the server before displaying it, using `textContent`, so another player's chat can't inject HTML.

**Art note:** the mockup's characters are placeholder squares. The final sprites should be AI-generated pixel art, saved as PNG files. Use a generic fallback scene when a room ID is unknown, since other groups' servers have different rooms. List AI art in the README's Resources section.

**Design reference:** [clickable GUI mockup](https://claude.ai/artifact/RqLcMRGdv1nH2nfQcXeJ5j). The owner must share it from the page's Share menu before teammates can open it.

---

## 9. Working simultaneously

### Day 1 (together, about 2 hours)

1. Create the Git repository, `go.mod` and the full folder structure.
2. Commit `game.go` with the `Event` and `Result` types and the complete `switch`.
3. Add stubs in every game file that return valid fake RFC replies.
4. Commit `protocol/errors.go`.
5. Agree on [section 5](#5-protocol-decisions) together.

After Day 1, `go run ./cmd/server` works with fake replies, which means nobody waits for anyone.

### How each person works alone

| Role | Tests against | Waits for |
|---|---|---|
| **A** | `nc` and the stub game | Nothing; replaces own stubs |
| **B** | `go test`, calling `Handle` directly | Nothing; no network needed |
| **C** | The stub server, plus fake data at first | Nothing; real replies appear as A and B merge |

### Connection points

These are the only places roles touch, and all are settled on Day 1:

| Between | Connection | Defined by |
|---|---|---|
| A and B | `Connect`, `Handle` and `Disconnect` | The contract in section 2 |
| A and B | Group kill credit: B calls `g.groupMembersInRoom(player)` | A writes it, and the Day 1 stub returns only the player |
| A and C | Bridge passes raw lines over a WebSocket | One text message per protocol line |
| B and C | Reply formats | RFC plus section 5 |

### Git workflow

- `main` always builds.
- One branch per feature, such as `feature/combat` or `feature/chat`.
- Clear commit messages, such as `Add DEFEND command`.
- Merge small and often, at least daily.
- Each pull request gets a quick review from a teammate, which also spreads understanding of the code.
- Only touch files you own. If you need a change elsewhere, ask the owner.

### Understanding each other's work

The subject warns that being unable to explain code, including AI-generated code, can fail the project. Before evaluation, each person walks the others through their part. Everyone should be able to answer:

- How does a line get from a client to the game and back?
- What happens when a client disconnects mid-broadcast?
- How does the GUI know which command a reply belongs to?
- How does an item stay unique?
- How are damage and quest completion calculated?

---

## 10. Compliance checklist

Each item should be tested before submission. This comes from the subject and RFC. The evaluation guidelines the subject mentions aren't included, so check them too if you can get them.

### Connection

- [ ] Greeting `OK hello proto=1` arrives first
- [ ] Duplicate name gets `ERR 201 NAME_IN_USE`
- [ ] A name is free again after its player disconnects
- [ ] Commands before `CONNECT` are rejected
- [ ] A second `CONNECT` is rejected
- [ ] Empty names and names with spaces are rejected
- [ ] Unicode names work
- [ ] `QUIT` returns `OK bye`, then closes

### Framing and bad input

- [ ] One command split across several packets works
- [ ] Several commands in one packet all work
- [ ] Lines over 1024 bytes are handled
- [ ] Invalid UTF-8 and control characters are handled
- [ ] CRLF line endings work
- [ ] Lowercase commands work
- [ ] Unknown commands, missing arguments and bad chat scopes return errors, and never crash the server

### Disconnects

- [ ] Killing a client mid-send doesn't interrupt broadcasts to others
- [ ] Player state is removed before the leave event is sent
- [ ] Their items drop into the room
- [ ] They leave their group, and the leader role passes on if needed
- [ ] `EVT STATS` updates

### Commands and events

- [ ] All 15 RFC commands work: CONNECT, LOOK, MOVE, CHAT, TAKE, DROP, INVENTORY, TALK, ATTACK, STATUS, QUEST, QUESTS, WHO, GROUP, QUIT
- [ ] All GROUP subcommands work: CREATE, INVITE, JOIN, LEAVE
- [ ] All RFC events are sent: ROOM PRESENCE ENTER and LEAVE, ROOM CHAT, GLOBAL CHAT, GROUP INVITE, JOIN, LEAVE and CHAT, STATS
- [ ] All RFC error codes appear where specified
- [ ] Replies match the RFC's formats exactly

### Items

- [ ] Two players racing for one item: only one gets it
- [ ] Dropped items can be taken by others
- [ ] ID, display name, any letter case and multi-word names all work
- [ ] Non-obtainable items can't be taken
- [ ] `DROP` on an unheld item gets `ERR 404 ITEM_NOT_IN_INVENTORY`

### Combat

- [ ] Players start at 100 HP
- [ ] Attacking a friendly NPC gets `ERR 405 NPC_NOT_HOSTILE`
- [ ] Attacking an absent NPC gets `ERR 404 NPC_NOT_FOUND`
- [ ] Counterattacks work
- [ ] Two players can fight one enemy
- [ ] At 0 HP, the player respawns in the safe room with reduced HP
- [ ] Combat results are broadcast and logged
- [ ] `DEFEND` and `FLEE` work, and return `408` outside combat
- [ ] `STATUS` shows health and combat state

### Quests

- [ ] Accepting, progress, completion and rewards all work
- [ ] A repeated quest returns `ERR 406 NO_QUEST_AVAILABLE`
- [ ] Delivery and defeat quests both work
- [ ] `QUESTS` lists active and completed quests
- [ ] Progress and completion are logged

### Groups

- [ ] Group chat with no group returns `401`
- [ ] Creating a second group returns `402`
- [ ] Joining without an invite returns `407`
- [ ] The leader leaving or disconnecting is handled

### World

- [ ] At least 8 rooms, with a loop and a branch
- [ ] A full circuit is possible
- [ ] At least 3 NPC roles, 4 items (2 obtainable) and 2 quests
- [ ] A bad exit or reference stops the server at startup

### Logging

- [ ] Connections and disconnections, with timestamps and IP addresses
- [ ] Every command, with player name and parameters
- [ ] All responses and error codes
- [ ] Item movements, NPC interactions and combat results
- [ ] Quest progress and completion
- [ ] JSON format, with INFO, WARN and ERROR levels
- [ ] Flood and rapid-connection warnings

### Clients

- [ ] CLI shows events while waiting for input
- [ ] GUI stays responsive during events
- [ ] Every required GUI feature in Role C's list works
- [ ] Both clients ignore unknown events without crashing
- [ ] Both clients accept both WHO and TALK formats
- [ ] Tested against another group's server, if possible

### Build and submission

- [ ] `make install`, `run-server`, `run-client`, `run-client-gui`, `lint` and `clean` all work
- [ ] `make lint` passes
- [ ] `go test -race ./...` passes on Ubuntu
- [ ] Each member has commits
- [ ] A fresh clone builds and runs from the README alone

---

## 11. README requirements

The README must be in English.

**The first line must be italic:**

```
*This project has been created as part of the 42 curriculum by <login1>, <login2>, <login3>.*
```

| Section | Owner |
|---|---|
| Description | C |
| Instructions | C |
| Resources (references, and how AI was used, for which tasks and parts) | Everyone |
| Architecture | A |
| Protocol Implementation | C, using section 5 of this plan |
| Combat System | B |
| Quest System | B |
| World Design | B |
| Server Logging | A |
| Group Contributions | Everyone |
| Building and Running | C |
| Testing | C |

The README must also state which CLI approach we chose (raw RFC commands) and why, as the subject requires.
