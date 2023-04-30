package abc

import (
	_ "embed"
	"testing"
)

//go:embed testdata/simple.abc
var simple string

//go:embed testdata/15.abc
var tune15 string

//go:embed testdata/16.abc
var tune16 string

var tunebook = "\n" + tune15 + "\n" + tune16 + "\n"

func TestParse(t *testing.T) {
	book, warnings := Parse(tunebook)
	require(t, 2, len(book.Tunes))
	for _, warn := range warnings {
		t.Error(warn)
	}
}

func TestSimple(t *testing.T) {
	book, warnings := Parse(simple)
	require(t, 1, len(book.Tunes))
	for _, warn := range warnings {
		t.Error(warn)
	}
	for _, tune := range book.Tunes {
		for _, stave := range tune.Body.Staves {
			t.Log("stave")
			for _, sym := range stave.Symbols {
				t.Logf("%v\n", sym)
			}
		}
	}
}

func require[T comparable](t *testing.T, expect, got T) {
	if expect != got {
		t.Helper()
		t.Fatalf("expected %v, got %v", expect, got)
	}
}
