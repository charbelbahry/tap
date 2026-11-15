---
title: Testing & Verification Matrix
tags:
  - testing
  - verification
  - peer-review
  - compliance
created: 2026-11-16
---

# 🧪 Testing & Verification Matrix

Back to: [[00 - Dashboard]] | Architecture: [[01 - Project Architecture]] | Plan: [[03 - Role B Plan & Milestones]]

---

## 1. Automated Test Commands

Run tests directly from the repository root:

```bash
# 1. Run all unit tests with race detection (Ubuntu / Linux)
go test -v -race ./...

# 2. Run only Role B world loader and validator tests
go test -v ./internal/world/...

# 3. Run only Role B game engine rules tests (room, item, combat, quest, npc)
go test -v ./internal/game/...

# 4. Run static analysis and style verification
go vet ./...
gofmt -l .
```

---

## 2. Role B Automated Test Matrix

| Package | Test File | Target Functionality | Success Criteria |
|---|---|---|---|
| `world` | `world_test.go` | `world.Load("data/world.yaml")` | Loads all 11 rooms, 9 NPCs, 8 items, 5 quests without error. |
| `world` | `world_test.go` | `Validate()` Dangling Exit | Returns descriptive error naming the invalid room reference. |
| `world` | `world_test.go` | `Validate()` Missing Item/NPC | Detects non-existent items or NPCs assigned to a room. |
| `world` | `world_test.go` | `Validate()` Quest Objectives | Rejects quests referencing non-existent NPCs or items. |
| `game` | `room_test.go` | `newPlayer(name)` | Spawns player in `loc.votive_chapel` with 100 HP. |
| `game` | `room_test.go` | `LOOK` | Outputs valid RFC JSON with room ID, exits, items, and NPCs. |
| `game` | `room_test.go` | `MOVE <valid-dir>` | Transitions player room, generates `LEAVE` and `ENTER` events. |
| `game` | `room_test.go` | `MOVE <invalid-dir>` | Returns `ERR 301 NO_EXIT`; player remains in room. |
| `game` | `item_test.go` | `TAKE <display-name>` | Matches multi-word display name (e.g. `"Pilgrim's Candle"`). |
| `game` | `item_test.go` | Item Uniqueness | Second player taking same item receives `ERR 404 ITEM_NOT_FOUND`. |
| `game` | `item_test.go` | `DROP <item>` | Item is removed from inventory and appears in room items. |
| `game` | `item_test.go` | `dropAllItems(p)` | All carried items are placed in room upon player disconnect. |
| `game` | `combat_test.go` | `ATTACK <hostile>` | Player strikes first; enemy counterattacks; broadcast `EVT ROOM COMBAT`. |
| `game` | `combat_test.go` | `ATTACK <friendly>` | Returns `ERR 405 NPC_NOT_HOSTILE`; no damage dealt. |
| `game` | `combat_test.go` | `DEFEND` | Enemy counterattack damage is halved. |
| `game` | `combat_test.go` | Player Death | At 0 HP, player respawns at `loc.votive_chapel` with 50 HP. |
| `game` | `combat_test.go` | Enemy Respawn | Defeated enemy automatically revives after 45 seconds. |
| `game` | `quest_test.go` | Quest Lifecycle | Sequential: `available` $\rightarrow$ `active` $\rightarrow$ `completed`. |
| `game` | `quest_test.go` | Fetch Quest Return | Delivery item is removed from player and restored to starting room. |
| `game` | `quest_test.go` | Group Kill Credit | Party members in the room both receive credit for defeating the boss. |

---

## 3. Manual Interactive Testing Walkthrough (Netcat / Telnet)

### Terminal 1: Start Server
```bash
go run ./cmd/server
```

### Terminal 2: Player Alice
```text
$ nc 127.0.0.1 4242
OK hello proto=1
CONNECT Alice
OK connected
LOOK
OK {"room":{"id":"loc.votive_chapel","name":"Votive Chapel",...},"players":[],"items":["item.pilgrims_candle"],"npcs":["npc.sister_ilvane"]}
TAKE Pilgrim's Candle
OK taken=item.pilgrims_candle
INVENTORY
OK ["item.pilgrims_candle"]
MOVE east
OK room=loc.pilgrims_road
LOOK
OK {"room":{"id":"loc.pilgrims_road",...},"npcs":["npc.rotting_sentry"]}
ATTACK rotting_sentry
OK {"attacker_hp":94,"target_hp":19,"damage":11,"status":"combat"}
```

### Terminal 3: Player Bob (Multiplayer Test)
```text
$ nc 127.0.0.1 4242
OK hello proto=1
CONNECT Bob
OK connected
LOOK
OK {"room":{"id":"loc.votive_chapel",...},"items":[]}
MOVE east
OK room=loc.pilgrims_road
ATTACK rotting_sentry
OK {"attacker_hp":100,"target_hp":8,"damage":11,"status":"combat"}
```

*(Alice receives combat broadcast events in Terminal 2 as Bob strikes the sentry!)*

---

## 4. 42 Peer-Evaluation Defense Guide

During evaluation, evaluators may ask you to explain your code or perform a **live code modification** (typically 5-10 minutes).

### Key Explanations to Master:
1. **How is item uniqueness guaranteed?**
   > *Answer*: Items are never duplicated because they are stored either in `Game.roomItems[roomID]` or in a player's `p.Inventory`. Picking up an item deletes it from the room slice before adding it to the player. The entire transaction is atomic because it executes under the server's single global mutex (`s.mu.Lock()`).
2. **What prevents the server from hanging when a client lags or disconnects?**
   > *Answer*: Each client has an asynchronous outgoing buffered channel (`chan string, 256`). Event delivery drops messages into the channel without blocking. If a client stalls and fills its 256-line buffer, the server drops the slow client immediately and continues serving others.
3. **How does enemy respawn work without race conditions?**
   > *Answer*: Dead enemies record a `RespawnAt = time.Now().Add(45 * time.Second)` timestamp. Rather than running uncoordinated background timer goroutines, the game checks the timestamp lazily when the room is entered or inspected.

### Likely Live Modification Scenarios:
- **Scenario A**: *Add a new flavor item or change an item's damage bonus.*
  - Edit `data/world.yaml`, add the item or adjust `damage_bonus`, run `go test ./...`.
- **Scenario B**: *Modify player respawn health from 50 HP to 75 HP.*
  - Edit `internal/game/combat.go` where `p.HP <= 0` resets `p.HP = 75`.
- **Scenario C**: *Add a new exit between two rooms.*
  - Add bidirectional exit entries in `data/world.yaml` under `exits`.
