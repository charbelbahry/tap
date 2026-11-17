---
title: TAP Master Dashboard
tags:
  - moc
  - tap
  - 42project
  - role-b
created: 2026-11-16
---

# 🏰 TAP: The Last Vigil — Master Dashboard

> [!NOTE]
> Welcome to the **TAP: The Last Vigil** Obsidian Knowledge Base. This vault houses the complete technical architecture, protocol specifications, lore, game formulas, test runbooks, and the **Person B (Role B) Master Plan**.

---

## 🗺️ Vault Navigation (Map of Content)

| Document | Description | Primary Owner |
|---|---|---|
| [[01 - Project Architecture]] | The core problem, technical solution, concurrency model, and system flow. | Everyone / Role A |
| [[02 - Protocol & Terminology]] | RFC 42TAP wire format, framing, message types, and complete engineering glossary. | Everyone |
| [[03 - Role B Plan & Milestones]] | **The Master Plan for Person B**: 8 testable milestones with detailed implementations & tests. | **Role B (You)** |
| [[04 - Game Rules & Formulas]] | Combat math, turn order, DEFEND/FLEE logic, respawns, and quest progression rules. | **Role B (You)** |
| [[05 - World Design & Lore]] | Story of Morvane, 11-room ASCII map, NPC index, item catalog, and boss mechanics. | **Role B (You)** |
| [[06 - RFC Command Reference]] | Wire-level reference for all 15 commands, replies, events, and error codes. | Everyone |
| [[07 - Testing & Verification Log]] | Automated unit tests, race detector, manual `nc` testing guides, and 42 peer-review prep. | Everyone / Role C |

---

## 👥 Team Roles & Responsibilities

```
                                      ┌─────────────────────────────────────┐
                                      │         TAP: The Last Vigil         │
                                      └──────────────────┬──────────────────┘
                     ┌───────────────────────────────────┼───────────────────────────────────┐
                     │                                   │                                   │
                     ▼                                   ▼                                   ▼
        ┌─────────────────────────┐         ┌─────────────────────────┐         ┌─────────────────────────┐
        │  Role A: Network & Core │         │ Role B: World & Gameplay│         │ Role C: GUI & Compliance│
        ├─────────────────────────┤         ├─────────────────────────┤         ├─────────────────────────┤
        │ • TCP server & Framing  │         │ • World YAML & Validator│         │ • Pixel Web GUI         │
        │ • Goroutines & Lock     │         │ • Room Navigation (LOOK)│         │ • WebSocket Bridge      │
        │ • Session (CONNECT/QUIT)│         │ • Dynamic Item Engine   │         │ • Makefile & Linter     │
        │ • CHAT, WHO, GROUP      │         │ • Turn Combat & Death   │         │ • Compliance Checklist  │
        │ • JSON Logging & Abuse  │         │ • Quest State Machine   │         │ • README Lead           │
        └─────────────────────────┘         └─────────────────────────┘         └─────────────────────────┘
```

---

## 🎯 Role B Sprint Progress (Milestone Tracker)

Use this checklist to track your milestone execution as you build and test:

- [x] **[[03 - Role B Plan & Milestones#Milestone 1 Static World Data Ingestion & Validation internalworld|Milestone 1: World Data Ingestion & Validation]]** (`internal/world`)
  - [x] Implement `world.Load(path)` using `gopkg.in/yaml.v3`
  - [x] Implement graph & reference validator in `Validate()`
  - [x] Write unit test suite `world_test.go`
  - [x] *Outcome: Unblocked server startup (`cmd/server/main.go`)*
- [x] **[[03 - Role B Plan & Milestones#Milestone 2 Player Spawning Spatial Navigation & Presence internalgameroomgo|Milestone 2: Player Spawning & Navigation]]** (`internal/game/room.go`)
  - [x] Initialize player state in `world.Map.Start` (100 HP)
  - [x] Implement RFC-compliant JSON response for `LOOK`
  - [x] Implement `MOVE <dir>` exit checks and `EVT ROOM PRESENCE` events
  - [x] Write unit test suite `room_test.go`
- [ ] **[[03 - Role B Plan & Milestones#Milestone 3 Dynamic Item Lifecycle Uniqueness & Fuzzy Matching internalgameitemgo|Milestone 3: Dynamic Item Engine & Fuzzy Matching]]** (`internal/game/item.go`)
  - [ ] Implement multi-token fuzzy resolver (full ID, short ID, display name)
  - [ ] Implement `TAKE`, `DROP`, `INVENTORY` with uniqueness guarantees
  - [ ] Implement `dropAllItems(p)` on disconnect/death
  - [ ] Write unit test suite `item_test.go`
- [ ] **[[03 - Role B Plan & Milestones#Milestone 4 NPC Interaction & Dialogue Engine internalgamenpcgo|Milestone 4: NPC Interaction & Dialogue]]** (`internal/game/npc.go`)
  - [ ] Implement `TALK <npc>` with dialogue rotation
  - [ ] Write unit test suite `npc_test.go`
- [ ] **[[03 - Role B Plan & Milestones#Milestone 5 Turn-Based Combat Engine Initiative & Respawn internalgamecombatgo|Milestone 5: Turn-Based Combat & Death Cycle]]** (`internal/game/combat.go`)
  - [ ] Implement `ATTACK` with damage formula + counterattack
  - [ ] Implement `DEFEND` (halves incoming damage) and `FLEE` (random exit + free hit)
  - [ ] Implement death & safe-room respawn at 50 HP (items kept)
  - [ ] Implement 45-second enemy respawn timer
  - [ ] Implement `STATUS` command
  - [ ] Write unit test suite `combat_test.go`
- [ ] **[[03 - Role B Plan & Milestones#Milestone 6 Quest Progression State Machine & Party Credit internalgamequestgo|Milestone 6: Quest Progression & Party Credit]]** (`internal/game/quest.go`)
  - [ ] Implement `QUEST <npc>` state machine (accept, check, complete)
  - [ ] Implement `QUESTS` listing
  - [ ] Connect group kill credit (`g.groupMembersInRoom`)
  - [ ] Handle delivery item return to spawn
  - [ ] Write unit test suite `quest_test.go`
- [ ] **[[03 - Role B Plan & Milestones#Milestone 7 11-Room World Assembly & Boss Integration dataworldyaml|Milestone 7: 11-Room World Assembly & Bosses]]** (`data/world.yaml`)
  - [ ] Construct full 11-room graph with loop and 2 branches
  - [ ] Implement all 9 NPCs, 8 items, 5 quests
  - [ ] Implement Bell Warden boss and Baal secret boss mechanics
- [ ] **[[03 - Role B Plan & Milestones#Milestone 8 Balance Tuning Verification & README Documentation|Milestone 8: README & Evaluation Defense]]**
  - [ ] Write `Combat System`, `Quest System`, and `World Design` sections in `README.md`
  - [ ] Pass `go test -race ./...` and `golangci-lint`
  - [ ] Practice live code change scenarios for peer review

---

## ⚡ Quick Shell Commands

```bash
# Run server
go run ./cmd/server

# Connect with raw TCP (netcat)
nc 127.0.0.1 4242

# Run all unit tests with race detection
go test -v -race ./...

# Run only Role B tests
go test -v ./internal/world/... ./internal/game/...
```
