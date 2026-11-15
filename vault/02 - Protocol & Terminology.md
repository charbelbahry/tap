---
title: Protocol & Terminology Reference
tags:
  - protocol
  - rfc42tap
  - terminology
  - errors
created: 2026-11-16
---

# 📜 Protocol & Terminology Reference

Back to: [[00 - Dashboard]] | Architecture: [[01 - Project Architecture]]

---

## 1. Technical Terminology Glossary

| Term | Definition & Context in TAP |
|---|---|
| **MUD** | *Multi-User Dungeon*: A real-time, text-based multiplayer virtual world where players share room locations, items, chat, and combat. |
| **RFC 42TAP** | The formal standard defining the wire protocol, ABNF grammar, command set, replies, events, and error codes for The Answer Protocol. |
| **TCP Stream Framing** | The process of parsing an unsegmented byte stream over TCP into discrete, line-delimited protocol messages using `\n` (ignoring optional preceding `\r`), capped at 1024 bytes. |
| **Synchronous Reply** | A direct 1-to-1 reply sent strictly to the issuing client answering their command (prefixed with `OK` or `ERR`). |
| **Asynchronous Event** | An unsolicited server push notification sent to inform one or many players of an action occurring in their vicinity (prefixed with `EVT`). |
| **Single-Lock Architecture** | The concurrency design where all network I/O runs across goroutines, but all state mutations within the `Game` instance happen sequentially under a single server-wide mutex. |
| **Unique Item Instance** | An item that exists strictly in one place at any given instant: either lying on the floor of a specific room or inside one player's inventory. Duplicate instances are impossible. |
| **Fuzzy Matching** | The ability of item and NPC commands to match targets using full ID (`item.vergers_hammer`), short ID (`vergers_hammer`), or case-insensitive multi-word display name (`Verger's Hammer`). |
| **Safe Room** | A designated sanctuary room (`loc.votive_chapel`) where hostile NPCs cannot enter, resting heals players to full 100 HP, and fallen players respawn after death at 50 HP. |
| **Turn-Based Combat** | Discrete combat exchanges where the player initiates action (`ATTACK`, `DEFEND`, `FLEE`), the player acts first, and the target enemy counterattacks immediately within the same command cycle. |
| **Free Hit** | A retaliatory strike automatically dealt by an enemy when an engaged player attempts to move to another room or flees during combat. |
| **Quest State Machine** | The progression of a quest per player: `available` (prerequisites met) $\rightarrow$ `active` (accepted, objectives pending) $\rightarrow$ `completed` (objective verified and reward granted). |
| **Group Kill Credit** | The mechanic wherein defeating an enemy grants quest kill progress to all party members currently present in the same room. |
| **World Graph** | The topological arrangement of interconnected rooms, exits, and branches loaded from YAML, requiring at least one closed loop and one branch. |

---

## 2. Wire Protocol Rules

- **Encoding**: UTF-8. Control characters (except tab) are strictly rejected.
- **Line Endings**: Protocol lines end with `\n` (LF). The server also strips trailing `\r` (CRLF) for compatibility with telnet clients.
- **Maximum Line Length**: 1024 bytes. Messages exceeding 1024 bytes return `ERR 400 BAD_REQUEST` and terminate the connection.
- **Ordering**: The server always transmits the synchronous `Reply` to the issuing client before pushing any resulting `Events`.

---

## 3. Protocol Error Codes Catalog

### Official RFC 42TAP Error Codes
| Code | Identifier | Description |
|---|---|---|
| `201` | `NAME_IN_USE` | The username requested during `CONNECT` is already taken by an active player. |
| `301` | `NO_EXIT` | The requested direction in `MOVE <dir>` does not exist in the current room. |
| `401` | `NOT_IN_GROUP` | Attempting group chat (`CHAT GROUP`) without belonging to a group. |
| `402` | `ALREADY_IN_GROUP` | Attempting to `GROUP CREATE` while already in a group. |
| `404` | `ITEM_NOT_FOUND` | The specified item does not exist in the room (or is not obtainable). |
| `404` | `ITEM_NOT_IN_INVENTORY` | Attempting to `DROP` an item not held in the player's inventory. |
| `404` | `NPC_NOT_FOUND` | The specified NPC does not exist in the current room. |
| `405` | `NPC_NOT_HOSTILE` | Attempting to `ATTACK` a dialogue or quest-giver NPC. |
| `406` | `NO_QUEST_AVAILABLE` | No more quests available from this NPC (or prerequisites unmet). |
| `900` | `CONNECTION_FAILED` | Server is full (100-client limit reached) or connection rejected. |
| `901` | `SEND_FAILED` | Internal socket send error. |

### Extensions Added for Completeness (Documented in README)
| Code | Identifier | Trigger Condition |
|---|---|---|
| `400` | `BAD_REQUEST` | Malformed line, missing arguments, unknown command, invalid name. |
| `402` | `ALREADY_CONNECTED` | Attempting a second `CONNECT` command on an active connection. |
| `403` | `NOT_AUTHENTICATED` | Sending gameplay commands before a successful `CONNECT`. |
| `404` | `PLAYER_NOT_FOUND` | `GROUP INVITE` targeting a non-existent player name. |
| `407` | `NOT_INVITED` | `GROUP JOIN <leader>` called without having received an active invite. |
| `408` | `NOT_IN_COMBAT` | Calling `DEFEND` or `FLEE` when not actively engaged in combat. |
| `429` | `RATE_LIMITED` | Exceeding 10 commands per second (flood protection). |

---

## 4. Subject Conflicts & Our Resolution

| Feature | RFC 42TAP Standard | Subject Example Snippet | Our Resolution |
|---|---|---|---|
| **WHO** | `OK players=5` | `OK {"room":[...],"server":5}` | **Server outputs RFC format** (`OK players=5`). Clients accept **both** formats to guarantee cross-group compatibility. |
| **TALK** | `OK Stay safe.` | `OK {"npc":"guard","dialogue":"..."}` | **Server outputs RFC format** (`OK <dialogue>`). Clients accept **both** formats. |

---

## 5. Protocol Extensions Added

The subject requires combat results to be broadcast and dynamic item states to be tracked in real time. We introduce three non-breaking event extensions:

1. **Combat Broadcast**:
   ```
   EVT ROOM COMBAT <username> <npc-id> {"damage":10,"target_hp":15}
   ```
2. **Item Pickup Broadcast**:
   ```
   EVT ROOM ITEM TAKE <username> <item-id>
   ```
3. **Item Drop Broadcast**:
   ```
   EVT ROOM ITEM DROP <username> <item-id>
   ```

Clients from other groups that follow RFC 42TAP will safely ignore these unrecognized events without errors.

---

## Next Steps
Proceed to the master roadmap: [[03 - Role B Plan & Milestones]].
