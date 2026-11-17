# Role A TODO

Leftovers and blockers from each phase.

## Open

- [ ] **`make lint` has one finding left:** `groupMembersInRoom` is unused until Role B calls it for kill credit. Ask Role B to use it, or lint stays red.
- [ ] **Re-test when Role B's remaining handlers land** (`TAKE`, `DROP`, `INVENTORY`, `TALK`, `ATTACK`, `DEFEND`, `FLEE`, `STATUS`, `QUEST`, `QUESTS`; still stubs on `main`): items dropped on disconnect (`dropAllItems`), `EVT ROOM ITEM` and `EVT ROOM COMBAT` events, shared kill credit, two players racing for one item.
- [ ] **`data/world.yaml` is still the 2-room starter** (needs 11 rooms, 8 items, 9 NPCs, 5 quests). Role B. The room tests only use the chapel and the road.
- [ ] **Role B items to raise:** `go.mod` was changed from `go 1.27` to `go 1.24`; `MOVE` with no direction returns `301` where the plan says `400`; hard-coded fallback start room and `g.world != nil` checks in `room.go`.
- [ ] **Browser test of the GUI:** two tabs are two players, closing a tab shows a leave in the other, and the page copes with `ERR 900` and a closed socket. Needs a human.
- [ ] **CLI:** try it against another group's server. No automated test (tested by hand, piped and under a pseudo-terminal). Windows line editing is untested.
- [ ] **README:** Architecture and Server Logging sections (Phase 5), plus the CLI choice (raw commands), the bridge design, and `PORT=` for a busy port 4242. Waiting for Role C's README (`docs/readme-role-c`) to merge.
- [ ] **Known, not fixed:** `Server.recent` (connections per IP) is never pruned, so it grows with every distinct IP. Harmless for a school project.

## Done

- [x] Phase 1: server core, CONNECT/QUIT/disconnect, abuse limits, logging
- [x] Phase 2: CHAT, WHO, GROUP, with leadership passing on
- [x] Phase 3: CLI client with `readline` (merged, PR #4)
- [x] Phase 4: GUI bridge, WebSocket to TCP (merged)
- [x] Audit 1 (phases 1-4): lint 45 issues down to 1, slow-client, limits, boundary and stress tests, real server run by hand
- [x] Audit 2 (Role B's LOOK/MOVE): server tests now use the real `world.yaml`; room chat, presence, LOOK and disconnect-in-another-room tested end to end
- [x] Blank lines now get `ERR 400 BAD_REQUEST` (one reply per line, which the GUI queue relies on). Mention it in the README.
