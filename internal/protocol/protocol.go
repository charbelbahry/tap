// Package protocol holds the fixed strings and parsing rules of RFC 42TAP.
//
// Owner: shared (written together on Day 1). Change only after agreeing as a team.
package protocol

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Fixed success replies.
const (
	Greeting    = "OK hello proto=1"
	OK          = "OK"
	OKConnected = "OK connected"
	OKBye       = "OK bye"
)

// Error replies defined by RFC 42TAP.
const (
	ErrNameInUse        = "ERR 201 NAME_IN_USE"
	ErrNoExit           = "ERR 301 NO_EXIT"
	ErrNotInGroup       = "ERR 401 NOT_IN_GROUP"
	ErrAlreadyInGroup   = "ERR 402 ALREADY_IN_GROUP"
	ErrItemNotFound     = "ERR 404 ITEM_NOT_FOUND"
	ErrItemNotInInv     = "ERR 404 ITEM_NOT_IN_INVENTORY"
	ErrNPCNotFound      = "ERR 404 NPC_NOT_FOUND"
	ErrNPCNotHostile    = "ERR 405 NPC_NOT_HOSTILE"
	ErrNoQuest          = "ERR 406 NO_QUEST_AVAILABLE"
	ErrConnectionFailed = "ERR 900 CONNECTION_FAILED"
	ErrSendFailed       = "ERR 901 SEND_FAILED"
)

// Error replies we add. Each one is documented in the README.
const (
	ErrBadRequest       = "ERR 400 BAD_REQUEST"
	ErrAlreadyConnected = "ERR 402 ALREADY_CONNECTED"
	ErrNotAuthenticated = "ERR 403 NOT_AUTHENTICATED"
	ErrPlayerNotFound   = "ERR 404 PLAYER_NOT_FOUND"
	ErrNotInvited       = "ERR 407 NOT_INVITED"
	ErrNotInCombat      = "ERR 408 NOT_IN_COMBAT"
	ErrRateLimited      = "ERR 429 RATE_LIMITED"
)

// Split returns the upper-cased command and the rest of the line.
// "chat ROOM  hello there" gives ("CHAT", "ROOM  hello there").
func Split(line string) (cmd, args string) {
	cmd, args, _ = strings.Cut(strings.TrimSpace(line), " ")
	return strings.ToUpper(cmd), strings.TrimSpace(args)
}

// Valid reports whether a line is UTF-8 with no control characters.
func Valid(line string) bool {
	return utf8.ValidString(line) && !strings.ContainsFunc(line, unicode.IsControl)
}
