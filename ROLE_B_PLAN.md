# Role B Plan & Milestones — TAP: The Last Vigil

> This document is part of the Person B (Role B: World and Gameplay) workspace.
> The full, interactive Obsidian knowledge base is available in [`vault/`](vault/00%20-%20Dashboard.md).

For the complete guide, open Obsidian on the [`vault/`](vault/) directory or read the documents directly:
- **Master Dashboard**: [`vault/00 - Dashboard.md`](vault/00%20-%20Dashboard.md)
- **Project Architecture**: [`vault/01 - Project Architecture.md`](vault/01%20-%20Project%20Architecture.md)
- **Protocol & Terminology**: [`vault/02 - Protocol & Terminology.md`](vault/02%20-%20Protocol%20%26%20Terminology.md)
- **Milestone Breakdown**: [`vault/03 - Role B Plan & Milestones.md`](vault/03%20-%20Role%20B%20Plan%20%26%20Milestones.md)
- **Game Rules & Formulas**: [`vault/04 - Game Rules & Formulas.md`](vault/04%20-%20Game%20Rules%20%26%20Formulas.md)
- **World Design & Lore**: [`vault/05 - World Design & Lore.md`](vault/05%20-%20World%20Design%20%26%20Lore.md)
- **RFC Command Reference**: [`vault/06 - RFC Command Reference.md`](vault/06%20-%20RFC%20Command%20Reference.md)
- **Testing & Verification Matrix**: [`vault/07 - Testing & Verification Log.md`](vault/07%20-%20Testing%20%26%20Verification%20Log.md)

---

## 8 Testable Milestones Summary

1. **Milestone 1: World Data Ingestion & Validation (`internal/world`)**
   - Unblocks `cmd/server/main.go` by implementing `world.Load` with `gopkg.in/yaml.v3` and `Validate()`.
   - Verified by `internal/world/world_test.go`.
2. **Milestone 2: Player Spawning & Room Navigation (`internal/game/room.go`)**
   - Implements `newPlayer`, `LOOK` (RFC JSON), and `MOVE` with exit validation and presence broadcast events.
   - Verified by `internal/game/room_test.go`.
3. **Milestone 3: Dynamic Item Engine & Fuzzy Matching (`internal/game/item.go`)**
   - Implements `TAKE`, `DROP`, `INVENTORY`, and `dropAllItems` with item uniqueness and multi-word fuzzy matching.
   - Verified by `internal/game/item_test.go`.
4. **Milestone 4: NPC Interaction & Dialogue Engine (`internal/game/npc.go`)**
   - Implements `TALK <npc>` with dialogue rotation and absence checks.
   - Verified by `internal/game/npc_test.go`.
5. **Milestone 5: Turn-Based Combat Engine & Respawn (`internal/game/combat.go`)**
   - Implements `ATTACK`, `DEFEND`, `FLEE`, `STATUS`, damage RNG, counterattacks, safe room respawn at 50 HP, and 45s enemy respawns.
   - Verified by `internal/game/combat_test.go`.
6. **Milestone 6: Quest Progression State Machine & Party Credit (`internal/game/quest.go`)**
   - Implements `QUEST <npc>` and `QUESTS` with fetch/kill objectives, prerequisites (`needs`), and group kill credit.
   - Verified by `internal/game/quest_test.go`.
7. **Milestone 7: 11-Room World Assembly & Boss Integration (`data/world.yaml`)**
   - Builds out the complete 11 rooms, loop + 2 branches, 9 NPCs, 8 items, 5 quests, Bell Warden, and Baal.
   - Verified by full world validation tests.
8. **Milestone 8: README Completion & Peer-Evaluation Defense**
   - Fills in `Combat System`, `Quest System`, and `World Design` sections of `README.md`.
