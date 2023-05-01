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
	"golang.org/x/exp/slices"
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
	for _, tune := range book.Tunes {
		c.Tune(tune)
	}
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

	const (
		repeatNone      = 0
		repeatNormal    = 1
		repeatAlternate = 2
	)

	repeatMode := repeatNone

	var lastSym abc.Symbol
	barAccidentals := map[string]string{}
	tiedNotePitch := ""

	for i, stave := range tune.Body.Staves {
		if i > 0 {
			c.pf(" \\break\n")
		}
		c.pf("   ")

		symbols := slices.Clone(stave.Symbols)

		// sort notes before decorations and texts
		for i := len(symbols) - 1; i >= 0; i-- {
			if symbols[i].Kind == abc.KindNote || symbols[i].Kind == abc.KindRest {
				p := i - 1
				for p >= 0 && (symbols[p].Kind == abc.KindDeco || symbols[p].Kind == abc.KindText) {
					symbols[p], symbols[p+1] = symbols[p+1], symbols[p]
				}
				i = p + 1
			}
		}

		for _, sym := range symbols {
			switch sym.Kind {
			case abc.KindText:
				c.pf(" ^%q", sym.Value)
			case abc.KindNote:
				dur := calculateDuration(&noteLength, &sym, &lastSym)

				var notePitch string
				if tiedNotePitch != "" {
					notePitch = tiedNotePitch
					tiedNotePitch = ""
				} else {
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
							n += barAccidentals[note.Pitch]
						}

						oct := note.Octave + 1
						for range iter(oct) {
							n += "'"
						}
						for range iter(-oct) {
							n += ","
						}

						notes = append(notes, n)
					}
					for k, v := range nextAccidentals {
						barAccidentals[k] = v
					}

					switch {
					case len(notes) > 1:
						notePitch = "<" + strings.Join(notes, " ") + ">"
					case len(notes) == 1:
						notePitch = notes[0]
					default:
						fmt.Printf("\n\n%#v\n\n", sym)
						panic("invalid notes")
					}
				}

				tie := ""
				if sym.Tie {
					tie = "~"
					tiedNotePitch = notePitch
				}

				c.pf(" %s%s%s", notePitch, durationToString(dur), tie)

			case abc.KindRest:
				tiedNotePitch = ""
				dur := calculateDuration(&noteLength, &sym, &lastSym)

				value := ""
				switch sym.Value {
				case "z":
					value = "r"
					c.pf(" %s%s", value, durationToString(dur))
				case "Z": // this should be full bar rest
					value = "r"
					dur.Mul(&dur, big.NewRat(2, 1)) // TODO: handle correctly
					c.pf(" %s%s", value, durationToString(dur))
				case "y":
				default:
					panic("unhandled rest " + sym.Value)
				}

			case abc.KindBar:
				barAccidentals = map[string]string{}

				// TODO: handle volta
				switch sym.Value {
				case "|":
					c.pf(` |`)
				case "||":
					c.pf(` \bar "||"`)
				case "|]":
					if repeatMode != repeatNone {
						panic("still in repeat")
					}
					c.pf(` \bar "|."`)
				case "|:":
					c.pf(` \repeat volta 2 {`)
					repeatMode = repeatNormal
				case ":|", ":|]", ":]":
					if repeatMode == repeatNone {
						panic("not in repeat")
					}
					c.pf(" }")
					repeatMode = repeatNone
				default:
					panic("unhandled bar " + sym.Value)
				}

				if sym.Volta != "" {
					panic("unhandled volta")
				}

			case abc.KindDeco:
				switch sym.Value {
				case ".":
					c.pf("-.")
				case "!marcato!":
					c.pf("-^")
				default:
					panic("unhandled deco " + sym.Value)
				}

			case abc.KindField:
				switch sym.Tag {
				case abc.FieldRemark.Tag:
					// IGNORE
				case abc.FieldUnitNoteLength.Tag:
					noteLength = abc.ParseNoteLength(sym.Value)
				default:
					panic("unhandled field " + sym.Tag + ":" + sym.Value)
				}
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

	for range iter(sym.Syncopation) {
		dur.Mul(&dur, big.NewRat(3, 2))
	}
	for range iter(-lastNote.Syncopation) {
		dur.Mul(&dur, big.NewRat(3, 2))
	}
	for range iter(-sym.Syncopation) {
		dur.Mul(&dur, big.NewRat(1, 2))
	}
	for range iter(lastNote.Syncopation) {
		dur.Mul(&dur, big.NewRat(1, 2))
	}

	return dur
}

func iter(n int) []struct{} {
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
