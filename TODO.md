# Role A TODO

Leftovers and blockers from each phase. Nothing here is committed yet.

## Open

- [ ] **Run the real server by hand** (`make run-server` + `nc`). Blocked: `world.Load` is still Role B's stub that returns an error.
- [ ] **Slow-client test:** a client that stops reading until its 256-line buffer fills gets disconnected, and the others keep receiving. The code exists, there is no test.
- [ ] **`make lint`:** `golangci-lint` isn't installed on this machine, so it is unverified. Only `go vet` and `gofmt` were run.
- [ ] **Re-test with Role B's real game:** my tests use the Day 1 stubs (every player starts in the same room). Once B's `newPlayer` and `MOVE` work, add a test that `CHAT ROOM` only reaches players in the same room.
- [ ] **Group kill credit:** `groupMembersInRoom` is done, but B has to call it. Check it is used when combat lands.
- [ ] **Add real logging checks** once the server runs: confirm every msg in ROLE_A_PLAN.md step 10 appears with the right level.
- [ ] **Placeholder `internal/gui/web/index.html`** exists already, so `go:embed` works. Role C replaces it.

## Done

- [x] Phase 1: server core, CONNECT/QUIT/disconnect, abuse limits, logging, 7 tests
- [x] Phase 2: CHAT, WHO, GROUP (CREATE/INVITE/JOIN/LEAVE), leader passes on when the leader leaves or disconnects, 3 more tests
