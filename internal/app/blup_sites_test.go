package app

import "testing"

// A vault made in Blup Notes (no settings next to its notes) comes as a
// vault of notes; one with settings keeps the kind they carry.
func TestBroughtAsNotes(t *testing.T) {
	if !broughtAsNotes(false) {
		t.Fatal("a vault from the phone must come as a vault of notes")
	}
	if broughtAsNotes(true) {
		t.Fatal("a vault with settings keeps their kind")
	}
}
