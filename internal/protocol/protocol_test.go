package protocol

import "testing"

func TestSplit(t *testing.T) {
	tests := []struct {
		line, cmd, args string
	}{
		{"look", "LOOK", ""},
		{"  MOVE north  ", "MOVE", "north"},
		{"chat ROOM  hi there", "CHAT", "ROOM  hi there"},
		{"TAKE Verger's Hammer", "TAKE", "Verger's Hammer"},
		{"", "", ""},
	}
	for _, tt := range tests {
		cmd, args := Split(tt.line)
		if cmd != tt.cmd || args != tt.args {
			t.Errorf("Split(%q) = (%q, %q), want (%q, %q)", tt.line, cmd, args, tt.cmd, tt.args)
		}
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"CHAT GLOBAL héllo 🙂", true},
		{"CONNECT алиса", true},
		{"hi\x07", false},
		{"tab\there", false},
		{"bad \xff utf8", false},
	}
	for _, tt := range tests {
		if got := Valid(tt.line); got != tt.want {
			t.Errorf("Valid(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}
