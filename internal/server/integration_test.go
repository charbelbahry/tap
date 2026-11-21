package server

import (
	"bufio"
	"encoding/json"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Room-based behaviour: Role A's events and chat against Role B's LOOK and MOVE,
// using the real data/world.yaml. The chapel's east exit leads to the road.

// quiet fails the test if anything unexpected is queued for c: the next line it
// receives must be the reply to WHO, because the server sends each client's lines in order.
func quiet(t *testing.T, c net.Conn, r *bufio.Reader) {
	t.Helper()
	send(t, c, "WHO\n")
	expect(t, c, r, "OK players=2")
}

func TestRoomChatStaysInRoom(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, b, "MOVE east\n")
	expect(t, b, rb, "OK room=loc.pilgrims_road")
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")

	send(t, a, "CHAT ROOM anyone here\n")
	expect(t, a, ra, "OK")
	expect(t, a, ra, "EVT ROOM CHAT alice anyone here")
	quiet(t, b, rb) // bob is in another room and heard nothing

	send(t, b, "CHAT ROOM hello road\n")
	expect(t, b, rb, "OK")
	expect(t, b, rb, "EVT ROOM CHAT bob hello road")
	quiet(t, a, ra)

	// GLOBAL ignores rooms.
	send(t, a, "CHAT GLOBAL still here\n")
	expect(t, a, ra, "OK")
	expect(t, a, ra, "EVT GLOBAL CHAT alice still here")
	expect(t, b, rb, "EVT GLOBAL CHAT alice still here")
}

func TestPresenceEventsFollowMoves(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, b, "MOVE east\n")
	expect(t, b, rb, "OK room=loc.pilgrims_road")
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")

	send(t, b, "MOVE west\n")
	expect(t, b, rb, "OK room=loc.votive_chapel")
	expect(t, a, ra, "EVT ROOM PRESENCE ENTER bob")
	quiet(t, b, rb) // the mover gets no presence event about themselves

	send(t, b, "MOVE up\n")
	expect(t, b, rb, "ERR 301 NO_EXIT")
}

func TestLookShowsWhoIsInTheRoom(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	players := func(c net.Conn, r *bufio.Reader) []string {
		t.Helper()
		send(t, c, "LOOK\n")
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		var look struct {
			Room    struct{ ID string }
			Players []string
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "OK ")), &look); err != nil {
			t.Fatalf("LOOK reply is not valid JSON: %q (%v)", line, err)
		}
		return look.Players
	}

	if got := players(a, ra); !reflect.DeepEqual(got, []string{"alice", "bob"}) {
		t.Fatalf("players = %v, want [alice bob]", got)
	}
	send(t, b, "MOVE east\n")
	expect(t, b, rb, "OK room=loc.pilgrims_road")
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")
	if got := players(a, ra); !reflect.DeepEqual(got, []string{"alice"}) {
		t.Fatalf("players after bob left = %v, want [alice]", got)
	}
	if got := players(b, rb); !reflect.DeepEqual(got, []string{"bob"}) {
		t.Fatalf("bob sees %v, want [bob]", got)
	}
}

// A player who disconnects from another room is announced in that room, not in ours.
func TestDisconnectIsAnnouncedInTheRightRoom(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, b, "MOVE east\n")
	expect(t, b, rb, "OK room=loc.pilgrims_road")
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")

	b.Close()
	expect(t, a, ra, "EVT STATS players=1") // no PRESENCE LEAVE: alice was never in bob's room
}

// Group chat and invites work across rooms.
func TestGroupWorksAcrossRooms(t *testing.T) {
	a, ra, b, rb := twoPlayers(t)

	send(t, a, "GROUP CREATE\n")
	expect(t, a, ra, "OK group=grp.1")
	send(t, a, "GROUP INVITE bob\n")
	expect(t, a, ra, "OK")
	expect(t, b, rb, "EVT GROUP INVITE alice")
	send(t, b, "GROUP JOIN alice\n")
	expect(t, b, rb, "OK group=grp.1")
	expect(t, a, ra, "EVT GROUP JOIN bob")
	expect(t, b, rb, "EVT GROUP JOIN bob")

	send(t, b, "MOVE east\n")
	expect(t, b, rb, "OK room=loc.pilgrims_road")
	expect(t, a, ra, "EVT ROOM PRESENCE LEAVE bob")

	send(t, a, "CHAT GROUP can you hear me\n")
	expect(t, a, ra, "OK")
	expect(t, a, ra, "EVT GROUP CHAT alice can you hear me")
	expect(t, b, rb, "EVT GROUP CHAT alice can you hear me")
}

// TestFullGameplayLifecycleOverTCP verifies end-to-end player lifecycle:
// STATUS, Items (TAKE, INV), Quests (QUEST, QUESTS), Movement, Combat (ATTACK, DEFEND, FLEE), and QUIT.
func TestFullGameplayLifecycleOverTCP(t *testing.T) {
	addr := startServer(t)
	c, r := login(t, addr, "hero")
	expect(t, c, r, "EVT STATS players=1")

	// 1. Initial STATUS
	send(t, c, "STATUS\n")
	expect(t, c, r, `OK {"hp":100,"max_hp":100,"status":"healthy"}`)

	// 2. Initial QUESTS
	send(t, c, "QUESTS\n")
	expect(t, c, r, "OK []")

	// 3. Take Item (pilgrims_candle in starting room loc.votive_chapel)
	send(t, c, "TAKE item.pilgrims_candle\n")
	expect(t, c, r, "OK taken=item.pilgrims_candle")

	// 4. INV shows the candle
	send(t, c, "INV\n")
	line, err := r.ReadString('\n')
	if err != nil || !strings.Contains(line, "item.pilgrims_candle") {
		t.Fatalf("unexpected INV reply: %q", line)
	}

	// 5. Accept quest from Sister Ilvane
	send(t, c, "QUEST npc.sister_ilvane\n")
	line, err = r.ReadString('\n')
	if err != nil || !strings.Contains(line, `"status":"available"`) {
		t.Fatalf("unexpected QUEST reply: %q", line)
	}

	// 6. Check QUESTS now has 1 active quest
	send(t, c, "QUESTS\n")
	line, err = r.ReadString('\n')
	if err != nil || !strings.Contains(line, `"quest_id":"road_without_watchers"`) {
		t.Fatalf("unexpected QUESTS reply: %q", line)
	}

	// 7. Move east to Pilgrim's Road where npc.rotting_sentry is stationed
	time.Sleep(1100 * time.Millisecond) // stay within 10 cmds/sec rate limit
	send(t, c, "MOVE east\n")
	expect(t, c, r, "OK room=loc.pilgrims_road")

	// 8. ATTACK npc.rotting_sentry
	send(t, c, "ATTACK npc.rotting_sentry\n")
	line, err = r.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "OK ") {
		t.Fatalf("unexpected ATTACK reply: %q", line)
	}

	// 9. DEFEND
	send(t, c, "DEFEND\n")
	line, err = r.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "OK ") {
		t.Fatalf("unexpected DEFEND reply: %q", line)
	}

	// 10. FLEE back to adjacent room
	send(t, c, "FLEE\n")
	line, err = r.ReadString('\n')
	if err != nil || !strings.HasPrefix(line, "OK room=") {
		t.Fatalf("unexpected FLEE reply: %q", line)
	}

	// 11. Clean QUIT
	send(t, c, "QUIT\n")
	expect(t, c, r, "OK bye")
}

