package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/egonelbre/lilypond/abc2ly/abc"
)

func main() {
	flag.Parse()

	data, err := os.ReadFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	book, warnings := abc.Parse(string(data))

	fmt.Fprintln(os.Stderr, "Parsed", len(book.Tunes), "tunes")
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "\t", warning.Message)
	}
}
