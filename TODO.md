# Role A TODO

Leftovers and blockers from each phase.

## Open

- [ ] **Run the real server by hand** (`make run-server` + `nc`). Blocked: `world.Load` is still Role B's stub that returns an error.
- [ ] **Slow-client test:** a client that stops reading until its 256-line buffer fills gets disconnected, and the others keep receiving. The code exists, there is no test.
- [ ] **`make lint`:** `golangci-lint` isn't installed on this machine, so it is unverified. Only `go vet` and `gofmt` were run.
- [ ] **Re-test with Role B's real game:** my tests use the Day 1 stubs (every player starts in the same room). Once B's `newPlayer` and `MOVE` work, add a test that `CHAT ROOM` only reaches players in the same room.
- [ ] **Group kill credit:** `groupMembersInRoom` is done, but B has to call it. Check it is used when combat lands.
- [ ] **Add real logging checks** once the server runs: confirm every msg in ROLE_A_PLAN.md step 10 appears with the right level.
- [ ] **CLI:** try it against another group's server. It has no automated test (tested by hand, piped and under a pseudo-terminal). Windows line editing is untested.
- [ ] **Bridge with the real page:** `web/app.js` (Role C) does not open a WebSocket yet. Once it does, check in a browser that two tabs are two players, and that the page handles `ERR 900` and a closed socket.
- [ ] **Bridge in a real browser:** the bridge is tested with a Go WebSocket client only.
- [ ] **Audit:** run the overengineering and complexity check on phases 1-4, then test against Role B and Role C code when it is on `main`.
- [ ] **README:** Architecture and Server Logging sections (Phase 5), plus the CLI choice (raw commands) and the bridge design.

## Done

- [x] Phase 1: server core, CONNECT/QUIT/disconnect, abuse limits, logging
- [x] Phase 2: CHAT, WHO, GROUP, with leadership passing on
- [x] Phase 3: CLI client with `readline` (merged, PR #4)
- [x] Phase 4: GUI bridge, WebSocket to TCP (branch `feature/gui-bridge`)
