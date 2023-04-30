package main

import (
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"

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

	c := Convert{Output: os.Stdout}
	c.Tune(book.Tunes[0])
}

type Convert struct {
	Output io.Writer
}

func (c *Convert) pf(format string, args ...any) {
	_, _ = fmt.Fprintf(c.Output, format, args...)
}

func (c *Convert) Tune(tune *abc.Tune) {
	c.Header(tune)
	c.Score(tune)
}
func (c *Convert) Score(tune *abc.Tune) {
	c.pf("\\score {\n")
	defer c.pf("}\n")

	c.pf("  \\new Staff{\n")
	c.pf("  \\accidentalStyle modern\n")
	defer c.pf("  }\n")

	if meter, ok := tune.Fields.ByTag(abc.FieldMeter.Tag); ok {
		c.pf("    \\time %v\n", meter.Value)
	}

	noteLength := *big.NewRat(1, 4)
	if f, ok := tune.Fields.ByTag(abc.FieldUnitNoteLength.Tag); ok {
		noteLength = abc.ParseNoteLength(f.Value)
	}

	for i, stave := range tune.Body.Staves {
		if i > 0 {
			c.pf(" \\break\n")
		}
		c.pf("   ")

		var lastSym abc.Symbol
		accidentals := map[string]string{}

		for _, sym := range stave.Symbols {
			switch sym.Kind {
			case abc.KindText:
				// TODO:
			case abc.KindNote:
				dur := calculateDuration(&noteLength, &sym, &lastSym)

				var notes []string
				nextAccidentals := map[string]string{}
				for _, note := range sym.Notes {
					n := note.Pitch
					if note.Accidentals != "" {
						suffix := ""
						for _, acc := range note.Accidentals {
							switch acc {
							case abc.AccidentalFlat:
								suffix += "es"
							case abc.AccidentalSharp:
								suffix += "is"
							case abc.AccidentalNatural:
								suffix = ""
							}
						}
						n += suffix
						nextAccidentals[note.Pitch] = suffix
					} else {
						n += accidentals[note.Pitch]
					}
					oct := note.Octave + 1
					for range repeat(oct) {
						n += "'"
					}
					for range repeat(-oct) {
						n += ","
					}

					notes = append(notes, n)
				}
				accidentals = nextAccidentals

				value := ""
				switch {
				case len(notes) > 1:
					value = "<" + strings.Join(notes, " ") + ">"
				case len(notes) == 1:
					value = notes[0]
				default:
					fmt.Printf("\n\n%#v\n\n", sym)
					panic("invalid notes")
				}

				c.pf(" %s%s", value, durationToString(dur))

			case abc.KindRest:
				dur := calculateDuration(&noteLength, &sym, &lastSym)

				value := ""
				switch sym.Value {
				case "z":
					value = "r"
				default:
					panic("unhandled rest " + sym.Value)
				}

				c.pf(" %s%s", value, durationToString(dur))

			case abc.KindBar:
				switch sym.Value {
				case "|":
					c.pf(` |`)
				case "||":
					c.pf(` \bar "||"`)
				case "|]":
					c.pf(` \bar "|."`)
				default:
					panic("unhandled bar " + sym.Value)
				}

			case abc.KindDeco:

			case abc.KindField:

			default:
				panic("unhandled " + sym.Kind.String())
			}

			if sym.Kind == abc.KindNote || sym.Kind == abc.KindRest {
				lastSym = sym
			} else if sym.Kind == abc.KindBar {
				lastSym = abc.Symbol{}
			}
		}
	}
	c.pf("\n")
}

func calculateDuration(noteLength *big.Rat, sym, lastNote *abc.Symbol) big.Rat {
	dur := *noteLength
	dur.Mul(&dur, &sym.Duration)

	for range repeat(sym.Syncopation) {
		dur.Mul(&dur, big.NewRat(3, 2))
	}
	for range repeat(-lastNote.Syncopation) {
		dur.Mul(&dur, big.NewRat(3, 2))
	}
	for range repeat(-sym.Syncopation) {
		dur.Mul(&dur, big.NewRat(1, 2))
	}
	for range repeat(lastNote.Syncopation) {
		dur.Mul(&dur, big.NewRat(1, 2))
	}

	return dur
}

func repeat(n int) []struct{} {
	if n > 0 {
		return make([]struct{}, n)
	}
	return nil
}

func durationToString(dur big.Rat) string {
	num := dur.Num().Int64()
	denom := dur.Denom().Int64()

	if num == 1 {
		return strconv.Itoa(int(denom))
	}
	if num == 3 {
		return strconv.Itoa(int(denom/2)) + "."
	}

	panic("unhandled duration " + dur.RatString())
}

func (c *Convert) Header(tune *abc.Tune) {
	c.pf("\\header {\n")
	defer c.pf("}\n")

	c.pf("    piece = %q\n", tune.Title)
	for _, field := range tune.Fields {
		switch field.Tag {
		case abc.FieldComposer.Tag:
			c.pf("    composer = %q\n", field.Value)
		case abc.FieldHistory.Tag:
			c.pf("    history = %q\n", field.Value)
		}
	}
}
