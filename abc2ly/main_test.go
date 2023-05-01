package main

import (
	"bytes"
	_ "embed"
	"flag"
	"os"
	"testing"

	"github.com/egonelbre/lilypond/abc2ly/abc"
	"github.com/google/go-cmp/cmp"
)

//go:embed testdata/features.abc
var featuresABC string

//go:embed testdata/features.ly
var featuresLy string

var update = flag.Bool("update", false, "update expected output")

func TestConvert(t *testing.T) {
	book, warnings := abc.Parse(featuresABC)
	for _, warn := range warnings {
		t.Error(warn)
	}

	var out bytes.Buffer
	convert := &Convert{Output: &out}
	for _, tune := range book.Tunes {
		convert.Tune(tune)
	}

	diff := cmp.Diff(featuresLy, out.String())
	if diff != "" {
		t.Error(diff)
		if *update {
			os.WriteFile("testdata/features.ly", out.Bytes(), 0644)
		}
		t.Log("LILYPOND OUTPUT:\n", out.String())
	}
}
