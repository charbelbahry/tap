package server

import (
	"bufio"
	"encoding/json"
	"net"
	"reflect"
	"strings"
	"testing"
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
