package abc

import (
	_ "embed"
	"testing"
)

//go:embed testdata/15.abc
var tune15 string

//go:embed testdata/16.abc
var tune16 string

var tunebook = "\n" + tune15 + "\n" + tune16 + "\n"

func TestParse(t *testing.T) {
	book, warnings := Parse(tunebook)
	require(t, 2, len(book.Tunes))
	if len(book.Tunes) != 2 {
		t.Fatal("expected 2 tunes")
	}
	t.Log(warnings)
	for _, tune := range book.Tunes {
		t.Logf("%#v\n", tune)
	}
}

func require[T comparable](t *testing.T, expect, got T) {
	if expect != got {
		t.Helper()
		t.Fatalf("expected %v, got %v", expect, got)
	}
}
