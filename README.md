*This project has been created as part of the 42 curriculum by <login1>, <login2>, <login3>.*

# TAP: The Last Vigil

> Work in progress. Each section below names its owner. Fill it in as your part is finished, and delete this note before submission.

## Description

<!-- Owner: Role C. What the project is, its goal, and a short overview of the server, the two clients and the world. -->

## Instructions

<!-- Owner: Role C. Requirements (Go 1.27, make), installation, and the quickest way to start playing. -->

## Building and Running

<!-- Owner: Role C. Every make target and the plain go command for machines without make.
| make | go | Does |
|---|---|---|
| make install | go mod download | ... |
| make run-server | go run ./cmd/server | ... |
| make run-client | go run ./cmd/cli | ... |
| make run-client-gui | go run ./cmd/gui | ... |
| make lint | golangci-lint run ./... | ... |
| make test | go test -race ./... | ... |
| make clean | ... | ... |
Also: flags for each program, and the CLI choice (raw RFC commands) with the reason. -->

## Architecture

<!-- Owner: Role A. Inline switch instead of a router, goroutines per client, the single lock, disconnect cleanup, framing. -->

## Protocol Implementation

<!-- Owner: Role C, from PLAN.md section 5. Every deviation and extension from RFC 42TAP, with the reason. -->

## Combat System

Combat in TAP is strictly **turn-based**. Each command issued by a player represents one discrete combat turn with no real-time timers:

1. **Turn Order & Initiative**:
   - The player acts first (dealing attack damage or raising defenses).
   - If the enemy survives, it counterattacks the player within the same turn.
   - In shared fights, multiple players can strike the same enemy; the enemy retaliates against whichever player just attacked it.

2. **Formulas & Scaling**:
   - **Player Attack**: `8 + rand(0..4) + DamageBonus` (where `DamageBonus` stacks bonuses from carried items like `Verger's Hammer` and quest rewards).
   - **Enemy Counterattack**: Random damage in `[DamageMin, DamageMax]` defined by the NPC's stats in `data/world.yaml`.

3. **Defensive Actions & Retreat**:
   - **`DEFEND`**: Sets a defensive stance; incoming enemy damage on that turn is halved (`dmg / 2`).
   - **`FLEE`**: Attempts to escape through a random room exit. Player has a 50% chance of taking an unmitigated hit before escaping.
   - **`MOVE <dir>` in Combat**: Permitted to ensure clients without a dedicated `FLEE` button never get stuck. The enemy deals a guaranteed free counterattack hit before the player moves.

4. **Death & Respawn**:
   - **Player Death**: When HP drops to $\le 0$, the player immediately respawns in the safe room (`loc.votive_chapel`) with **50 HP**, retaining all carried items. Combat state is cleared.
   - **Safe Room Healing**: Entering the safe room (`loc.votive_chapel`) through normal movement restores HP to full `MaxHP`.
   - **Enemy Respawn**: Dead enemies remain down for **45 seconds**, evaluated lazily whenever the room is inspected or entered (`LOOK`, `MOVE`, `ATTACK`), requiring zero background goroutines or timers.

5. **Boss Mechanics**:
   - **Bell Warden**: Heavy boss guarding the Bell Tower (150 HP, 10–16 damage). Best challenged with quest damage bonuses or in a party.
   - **Baal**: Secret boss in the Hall of the Answer (300 HP, 15–25 damage). Baal is immune to damage unless a player in the room is carrying **The Unspoken Question** (`item.unspoken_question`). Attacks without the item deal 0 damage.

## Quest System

The quest engine is driven through `QUEST <npc>` and `QUESTS`:

1. **State Machine (`QUEST <npc>`)**:
   - **Acceptance**: First call accepts the next available quest from the NPC and replies `status: available`.
   - **Active**: Subsequent calls before fulfilling objectives reply `status: active`.
   - **Completion**: Once objectives are satisfied, claims rewards and replies `status: completed`.
   - **Exhaustion**: If no more quests are available from this NPC, replies `ERR 406 NO_QUEST_AVAILABLE`.

2. **Objective Types**:
   - **Kill Quests**: Target enemy NPC must be slain.
   - **Delivery Quests**: Carried item must be present in player inventory. On delivery, the item is removed from the player and automatically returned to its original world spawn room so other players can complete the quest.

3. **Prerequisites & Party Credit**:
   - **Chained Quests (`needs`)**: Quests with prerequisite requirements stay hidden until the required quest is completed.
   - **Shared Group Credit**: Slaying an enemy awards kill credit to the attacker and all group members present in the same room (`g.groupMembersInRoom`).

4. **Quest Listing (`QUESTS`)**:
   - Returns a JSON array of all active and completed quests for the player, including objective completion progress (`0/1` or `1/1`).

## World Design

The world of **Morvane** is a dark, interconnected ruin set around King Aurenth's hubris and the primeval entity Baal:

### Map Topology (11 Rooms)

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

- **Outer Circuit Loop**: `Votive Chapel` $\rightarrow$ `Pilgrim's Road` $\rightarrow$ `Silent Market` $\rightarrow$ `Hanged Garden` $\rightarrow$ `Burned Library` $\rightarrow$ `Throne Ruin` $\rightarrow$ `Ossuary` $\rightarrow$ `Crypt Gate` $\rightarrow$ `Votive Chapel`.
- **Branch 1 (North)**: `Throne Ruin` $\rightarrow$ `Bell Tower` (Bell Warden boss).
- **Branch 2 (West/Down)**: `Ossuary` $\rightarrow$ `Sunken Stair` $\rightarrow$ `Hall of the Answer` (Baal secret boss).

### NPCs (9 Entities)

| NPC ID | Name | Role | Location | Description |
|---|---|---|---|---|
| `npc.sister_ilvane` | Sister Ilvane | `quest_giver` | Votive Chapel | Gentle keeper of the last candle. Gives Quests 1 & 3. |
| `npc.old_corvin` | Old Corvin | `quest_giver` | Crypt Gate | Tired gravedigger. Gives Quest 2. |
| `npc.mute_merchant` | Mute Merchant | `dialogue` | Silent Market | Has not spoken in 90 years; communicates via gestures. |
| `npc.zaza` | Zaza | `quest_giver` | Silent Market | King's former herald who read the question. Gives Quests 4 & 5. |
| `npc.rotting_sentry` | Rotting Sentry | `enemy` | Pilgrim's Road | Undead guard (30 HP, 4–8 dmg). |
| `npc.grave_hound` | Grave Hound | `enemy` | Ossuary | Skeletal hound (40 HP, 6–10 dmg). |
| `npc.fallen_knight` | Fallen Knight | `enemy` | Throne Ruin | Guardian of the empty throne (80 HP, 8–12 dmg). |
| `npc.bell_warden` | Bell Warden | `enemy` | Bell Tower | Boss of the tower (150 HP, 10–16 dmg). |
| `npc.baal` | Baal | `enemy` | Hall of the Answer | Primeval entity (300 HP, 15–25 dmg). |

### Items (8 Artifacts)

| Item ID | Name | Obtainable | Location | Effect / Purpose |
|---|---|---|---|---|
| `item.pilgrims_candle` | Pilgrim's Candle | `true` | Votive Chapel | Flavor item. |
| `item.rusted_censer` | Rusted Censer | `true` | Silent Market | Flavor item. |
| `item.funeral_bread` | Funeral Bread | `true` | Hanged Garden | Flavor item. |
| `item.burial_shroud` | Burial Shroud | `true` | Burned Library | Delivery item for Quest 2. |
| `item.heralds_scroll` | Herald's Scroll | `true` | Sunken Stair | Delivery item for Quest 4. |
| `item.sealed_altar` | Sealed Altar | `false` | Throne Ruin | World fixture (`obtainable: false`). |
| `item.vergers_hammer` | Verger's Hammer | `true` | Quest 2 Reward | $+6$ Attack Damage while carried. |
| `item.unspoken_question` | The Unspoken Question | `true` | Quest 4 Reward | Nullifies Baal's damage immunity. |

## Server Logging

<!-- Owner: Role A. JSON format, event table, output streams, monitoring with jq, abuse detection. -->

## Testing

<!-- Owner: Role C. How to test multiplayer, combat and quests, and how to run the automated tests. -->

## Group Contributions

<!-- Everyone. Each member's responsibilities and what they built. -->

## Resources

<!-- Everyone. References (RFC 2119, RFC 5234, Go docs, MUD history...) and how AI was used: which tasks and which parts of the project. -->
