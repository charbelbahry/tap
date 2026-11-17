# Role A TODO

Leftovers and blockers from each phase.

## Open

- [ ] **Merge order:** `feature/gui-bridge` and `main` (Role B's `yaml.v3`) both edited `go.mod`. When merging, keep both `require` lines and run `go mod tidy`.
- [ ] **Tests on the real world file:** once the bridge is merged, switch `startServer` in `internal/server/server_test.go` to `world.Load("../../data/world.yaml")`. Checked once in a scratch copy: passes.
- [ ] **`make lint` has one finding left:** `groupMembersInRoom` is unused until Role B calls it for kill credit. Ask Role B to use it, or lint stays red.
- [ ] **Re-test with Role B's real game handlers** (`LOOK`, `MOVE`, `TAKE`, ...; still stubs on `main`). Then add a test that `CHAT ROOM` only reaches players in the same room.
- [ ] **CLI:** try it against another group's server. No automated test (tested by hand, piped and under a pseudo-terminal). Windows line editing is untested.
- [ ] **Bridge with the real page:** `web/app.js` (Role C) does not open a WebSocket yet. Once it does, check in a browser that two tabs are two players, and that the page handles `ERR 900` and a closed socket.
- [ ] **README:** Architecture and Server Logging sections (Phase 5), plus the CLI choice (raw commands) and the bridge design.
- [ ] **Known, not fixed:** `Server.recent` (connections per IP) is never pruned, so it grows with every distinct IP. Harmless for a school project.

## Done

- [x] Phase 1: server core, CONNECT/QUIT/disconnect, abuse limits, logging
- [x] Phase 2: CHAT, WHO, GROUP, with leadership passing on
- [x] Phase 3: CLI client with `readline` (merged, PR #4)
- [x] Phase 4: GUI bridge, WebSocket to TCP (branch `feature/gui-bridge`)
- [x] Audit 1 (phases 1-4): see `refactor/simplify`. Lint 45 issues down to 1, slow-client, limits, boundary and stress tests added, real server run by hand with the real world file.
