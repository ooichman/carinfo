package main

import "testing"

func TestChipClass(t *testing.T) {
	cases := map[string]string{
		"new":           "chip chip-new",
		"New":           "chip chip-new",
		"Old":           "chip chip-old",
		"mid condition": "chip chip-mid",
	}
	for in, want := range cases {
		if got := chipClass(in); got != want {
			t.Errorf("chipClass(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseCarID(t *testing.T) {
	if id, ok := parseCarID("/cars/12/edit", "edit"); !ok || id != 12 {
		t.Fatalf("edit path: id=%d ok=%v", id, ok)
	}
	if id, ok := parseCarID("/cars/7/delete", "delete"); !ok || id != 7 {
		t.Fatalf("delete path: id=%d ok=%v", id, ok)
	}
	if id, ok := parseCarID("/cars/3", ""); !ok || id != 3 {
		t.Fatalf("update path: id=%d ok=%v", id, ok)
	}
	if _, ok := parseCarID("/cars/new", "edit"); ok {
		t.Fatal("cars/new should not parse as edit id")
	}
}
