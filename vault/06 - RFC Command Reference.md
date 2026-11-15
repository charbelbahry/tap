---
title: RFC 42TAP Command & Event Reference
tags:
  - rfc42tap
  - commands
  - events
  - wire-format
created: 2026-11-16
---

# 📡 RFC 42TAP Command & Event Reference

Back to: [[00 - Dashboard]] | Architecture: [[01 - Project Architecture]] | Plan: [[03 - Role B Plan & Milestones]]

---

## 1. Command Matrix (Roles Division)

| Command | Category | Owner | Wire Format Example | Expected Success Reply | Error Codes |
|---|---|---|---|---|---|
| `CONNECT` | Session | Role A | `CONNECT alice` | `OK connected` | `201`, `400`, `402`, `900` |
| `QUIT` | Session | Role A | `QUIT` | `OK bye` (closes connection) | - |
| `LOOK` | Exploration | **Role B** | `LOOK` | `OK {"room":{...},"players":[...],"items":[...],"npcs":[...]}` | `400` |
| `MOVE` | Exploration | **Role B** | `MOVE north` | `OK room=loc.crypt_gate` | `301`, `400` |
| `TAKE` | Items | **Role B** | `TAKE vergers_hammer` | `OK taken=item.vergers_hammer` | `404`, `400` |
| `DROP` | Items | **Role B** | `DROP item.pilgrims_candle` | `OK dropped=item.pilgrims_candle` | `404`, `400` |
| `INVENTORY`| Items | **Role B** | `INVENTORY` | `OK ["item.vergers_hammer"]` | `400` |
| `TALK` | Social/NPC | **Role B** | `TALK sister_ilvane` | `OK Stay safe, Vigilant.` | `404`, `400` |
| `ATTACK` | Combat | **Role B** | `ATTACK rotting_sentry` | `OK {"attacker_hp":92,"target_hp":18,"damage":12,"status":"combat"}` | `404`, `405`, `400` |
| `DEFEND` | Combat | **Role B** | `DEFEND` | `OK {"hp":96,"damage_taken":4,"status":"combat"}` | `408`, `400` |
| `FLEE` | Combat | **Role B** | `FLEE` | `OK room=loc.votive_chapel` | `408`, `400` |
| `STATUS` | Combat | **Role B** | `STATUS` | `OK {"hp":100,"max_hp":100,"status":"healthy"}` | `400` |
| `QUEST` | Quests | **Role B** | `QUEST sister_ilvane` | `OK {"quest_id":"road_without_watchers","reward":"+2 damage","status":"available"}` | `404`, `406`, `400` |
| `QUESTS` | Quests | **Role B** | `QUESTS` | `OK [{"quest_id":"road_without_watchers","status":"active","progress":"0/1"}]` | `400` |
| `CHAT` | Social | Role A | `CHAT ROOM hello` | `OK` | `400`, `401` |
| `WHO` | Social | Role A | `WHO` | `OK players=3` | `400` |
| `GROUP` | Social | Role A | `GROUP CREATE` | `OK group=grp_1` | `400`, `402`, `404`, `407` |

---

## 2. Event Reference & Routing

| Event Wire Format | Trigger | Recipients (`To: []string`) |
|---|---|---|
| `EVT ROOM PRESENCE ENTER <name>` | Player enters a room via `MOVE` or `CONNECT` | Other players currently in that destination room. |
| `EVT ROOM PRESENCE LEAVE <name>` | Player leaves room via `MOVE`, `QUIT`, or disconnect | Other players in that departure room. |
| `EVT ROOM ITEM TAKE <name> <item-id>` | Player takes an item from the floor | Other players currently in that room. |
| `EVT ROOM ITEM DROP <name> <item-id>` | Player drops an item onto the floor | Other players currently in that room. |
| `EVT ROOM COMBAT <name> <npc-id> <json>`| Combat exchange occurs (`ATTACK`, `DEFEND`) | Everyone in that room. |
| `EVT ROOM CHAT <name> <message>` | `CHAT ROOM <msg>` sent | Everyone in the room (including sender). |
| `EVT GLOBAL CHAT <name> <message>` | `CHAT GLOBAL <msg>` sent | Everyone connected on the server. |
| `EVT GROUP CHAT <name> <message>` | `CHAT GROUP <msg>` sent | All active members of that group. |
| `EVT GROUP INVITE <leader>` | `GROUP INVITE <player>` sent | The targeted invitee only. |
| `EVT GROUP JOIN <name>` | Player joins group via `GROUP JOIN` | All existing group members. |
| `EVT GROUP LEAVE <name>` | Player leaves group via `GROUP LEAVE` or disconnect | Remaining group members. |
| `EVT STATS players=<N>` | Server connection count changes | Everyone connected on the server. |
