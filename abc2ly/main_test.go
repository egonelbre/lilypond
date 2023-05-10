package main

import (
	"bytes"
	_ "embed"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egonelbre/lilypond/abc2ly/abc"
	"github.com/google/go-cmp/cmp"
)

var update = flag.Bool("update", false, "update expected output")

func TestConvert(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.abc")
	if err != nil {
		t.Fatal(err)
	}
	for _, abcpath := range matches {
		t.Run(filepath.Base(abcpath), func(t *testing.T) {
			lypath := strings.TrimSuffix(abcpath, ".abc") + ".ly"

			abcdata, err := os.ReadFile(abcpath)
			if err != nil {
				t.Fatal(err)
			}

			book, warnings := abc.Parse(string(abcdata))
			for _, warn := range warnings {
				t.Error(warn)
			}

			var out bytes.Buffer

			convert := &Convert{Output: &out}
			for _, tune := range book.Tunes {
				convert.Tune(tune)
			}

			converted := out.String()

			lydata, err := os.ReadFile(lypath)
			diff := ""
			if err != nil {
				t.Error(err)
				diff = "<LILYPOND MISSING>"
			} else {
				diff = cmp.Diff(string(lydata), converted)
			}

			if diff != "" {
				t.Error(diff)
				if *update {
					os.WriteFile("testdata/features.ly", out.Bytes(), 0644)
				}
				t.Log("LILYPOND OUTPUT:\n", out.String())
			}
		})
	}

}
