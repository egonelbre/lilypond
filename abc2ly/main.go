package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/egonelbre/lilypond/abc2ly/abc"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

func main() {
	filePerTune := flag.Bool("file-per-tune", false, "creates a single file per tune")
	outdir := flag.String("out", "", "output directory")
	flag.Parse()

	if *filePerTune && *outdir == "" {
		fmt.Fprint(os.Stderr, "-out required when using -file-per-tune")
		os.Exit(1)
	}

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

	if *filePerTune {
		paths := []string{}

		os.MkdirAll(*outdir, 0755)

		for _, tune := range book.Tunes {
			if tune.ID == "" {
				// TODO: handle this better
				continue
			}
			out := &bytes.Buffer{}
			c := Convert{Output: out}
			c.Tune(tune)
			p := filepath.Join(*outdir, tune.ID+".ly")
			err := os.WriteFile(p, out.Bytes(), 0o644)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			paths = append(paths, tune.ID+".ly")
		}

		main := &bytes.Buffer{}
		for _, p := range paths {
			fmt.Fprintf(main, "\\include %q\n", p)
		}

		err := os.WriteFile(filepath.Join(*outdir, "_index.ly"), main.Bytes(), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	} else {
		c := Convert{Output: os.Stdout}
		for _, tune := range book.Tunes {
			c.Tune(tune)
		}
	}
}

type Convert struct {
	Output io.Writer
}

func (c *Convert) pf(format string, args ...any) {
	_, _ = fmt.Fprintf(c.Output, format, args...)
}

func (c *Convert) Tune(tune *abc.Tune) {
	c.Score(tune)
}

func (c *Convert) Score(tune *abc.Tune) {
	c.pf("\\score {\n")
	defer c.pf("}\n")

	c.Header(tune)

	c.pf("  \\new Staff{\n")
	c.pf("  \\configureStaff\n")
	defer c.pf("  }\n")

	if meter, ok := tune.Fields.ByTag(abc.FieldMeter.Tag); ok {
		c.pf("    \\time %v", meter.Value)
	}

	noteLength := *big.NewRat(1, 4)
	if f, ok := tune.Fields.ByTag(abc.FieldUnitNoteLength.Tag); ok {
		noteLength = abc.ParseNoteLength(f.Value)
	}

	insideRepeat, insideVolta := false, false

	keySignature, octaveOffset := map[string]string{}, 1
	if k, ok := tune.Fields.ByTag(abc.FieldKey.Tag); ok {
		var key string
		key, keySignature, octaveOffset = parseKeySignature(k.Value, octaveOffset)
		decl, ok := abcKeySignatureToLilypond[key]
		if !ok {
			panic("unhandled signature " + key)
		}
		c.pf(" %s", decl)
	}

	var lastSym abc.Symbol
	barAccidentals := maps.Clone(keySignature)
	tiedNotePitch := ""

	c.pf("\n")
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

						oct := note.Octave + octaveOffset
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
					panic("unhandled rest " + sym.Value + " tune:" + tune.ID)
				}

			case abc.KindBar:
				barAccidentals = maps.Clone(keySignature)

				// TODO: handle volta

				switch sym.Value {
				case "|":
					c.pf(` |`)
					if sym.Volta != "" {
						c.pf(` \set Score.repeatCommands = #'((volta %q))`, sym.Volta)
						insideVolta = true
					} else if sym.CloseVolta {
						c.pf(` \set Score.repeatCommands = #'((volta #f))`)
						insideVolta = false
					}
				case "||":
					c.pf(` \bar "||"`)
					if sym.Volta != "" {
						c.pf(` \set Score.repeatCommands = #'((volta %q))`, sym.Volta)
						insideVolta = true
					} else if insideVolta || sym.CloseVolta {
						c.pf(` \set Score.repeatCommands = #'((volta #f))`)
						insideVolta = false
					}
				case "|]":
					if insideRepeat {
						panic("still in repeat")
					}
					if insideVolta || sym.CloseVolta {
						c.pf(` \set Score.repeatCommands = #'((volta #f))`)
						insideVolta = false
					}
					if sym.Volta != "" {
						panic("did not expect volta on |]")
					}
					c.pf(` \bar "|."`)
				case "::", ":|:", ":||:":
					var commands []string

					commands = append(commands, "end-repeat")
					commands = append(commands, "start-repeat")
					insideRepeat = true

					if sym.Volta != "" {
						commands = append(commands, fmt.Sprintf("(volta %q)", sym.Volta))
						insideVolta = true
					} else if insideVolta || sym.CloseVolta {
						commands = append(commands, fmt.Sprintf("(volta #f)"))
						insideVolta = false
					}

					c.pf(` \set Score.repeatCommands = #'(%s)`, strings.Join(commands, " "))

				case "|:", "||:":
					var commands []string
					if insideRepeat {
						commands = append(commands, "end-repeat")
						insideRepeat = false
					}

					commands = append(commands, "start-repeat")
					insideRepeat = true

					if sym.Volta != "" {
						commands = append(commands, fmt.Sprintf("(volta %q)", sym.Volta))
						insideVolta = true
					} else if insideVolta || sym.CloseVolta {
						commands = append(commands, fmt.Sprintf("(volta #f)"))
						insideVolta = false
					}

					c.pf(` \set Score.repeatCommands = #'(%s)`, strings.Join(commands, " "))

				case ":|", ":||", ":|]", ":]":
					commands := []string{"end-repeat"}
					insideRepeat = false

					if sym.Volta != "" {
						commands = append(commands, fmt.Sprintf("(volta %q)", sym.Volta))
						insideVolta = true
					} else if insideVolta || sym.CloseVolta {
						commands = append(commands, fmt.Sprintf("(volta #f)"))
						insideVolta = false
					}

					c.pf(` \set Score.repeatCommands = #'(%s)`, strings.Join(commands, " "))

				default:
					panic("unhandled bar " + sym.Value + " tune:" + tune.ID)
				}

			case abc.KindDeco:
				switch sym.Value {
				case ".":
					c.pf("-.")
				case "!marcato!":
					c.pf("-^")
				case "!segno!":
					c.pf(` \segnoMark 1 `)
				case "!coda!":
					c.pf(` \codaMark 1 `)
				case "!accent!":
					// TODO:
				default:
					panic("unhandled deco " + sym.Value + " tune:" + tune.ID)
				}

			case abc.KindField:
				switch sym.Tag {
				case abc.FieldRemark.Tag, abc.FieldNotes.Tag:
					// IGNORE
				case abc.FieldUnitNoteLength.Tag:
					noteLength = abc.ParseNoteLength(sym.Value)
				case abc.FieldMeter.Tag:
					// TODO:
				case abc.FieldKey.Tag:
					var key string
					key, keySignature, octaveOffset = parseKeySignature(sym.Value, octaveOffset)
					barAccidentals = maps.Clone(keySignature)

					decl, ok := abcKeySignatureToLilypond[key]
					if !ok {
						panic("unhandled signature " + key)
					}
					c.pf(" %s", decl)
				default:
					panic("unhandled field " + sym.Tag + ":" + sym.Value + " tune:" + tune.ID)
				}
			default:
				panic("unhandled " + sym.Kind.String() + " tune:" + tune.ID)
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

func parseKeySignature(keysig string, octaveOffset int) (key string, accidentals map[string]string, octave int) {
	// TODO: allow only changing octave etc.

	if keysig == "" {
		return "", map[string]string{}, octaveOffset
	}
	accidentals = map[string]string{}

	fields := strings.Fields(keysig)
	if len(fields) > 0 {
		accidentals = lookupAccidentalMap(fields[0])
		key = strings.ToLower(fields[0])
	}

	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "octave=") {
			octs := strings.TrimPrefix(f, "octave=")
			n, err := strconv.Atoi(octs)
			if err != nil {
				panic(err) // TODO: proper handling
			}
			octaveOffset = n + 1
		}
	}

	return key, accidentals, octaveOffset
}

var abcKeySignatureToLilypond = map[string]string{
	"c#": `\key cis \major`,
	"f#": `\key fis \major`,
	"b":  `\key b \major`,
	"e":  `\key e \major`,
	"a":  `\key a \major`,
	"d":  `\key d \major`,
	"g":  `\key g \major`,
	"c":  `\key c \major`,
	"f":  `\key f \major`,
	"bb": `\key bes \major`,
	"eb": `\key ees \major`,
	"ab": `\key aes \major`,
	"db": `\key des \major`,
	"gb": `\key ges \major`,
	"cb": `\key ces \major`,

	"a#m": `\key ais \minor`,
	"d#m": `\key dis \minor`,
	"g#m": `\key gis \minor`,
	"c#m": `\key cis \minor`,
	"f#m": `\key fis \minor`,
	"bm":  `\key b \minor`,
	"em":  `\key e \minor`,
	"am":  `\key a \minor`,
	"dm":  `\key d \minor`,
	"gm":  `\key g \minor`,
	"cm":  `\key c \minor`,
	"fm":  `\key f \minor`,
	"bbm": `\key bes \minor`,
	"ebm": `\key ees \minor`,
	"abm": `\key aes \minor`,
}

func lookupAccidentalMap(k string) map[string]string {
	const sh = "fcgdaeb"
	const fl = "beadgcf"
	// F♯, C♯, G♯, D♯, A♯, E♯, B♯
	// B♭, E♭, A♭, D♭, G♭, C♭, F♭

	switch strings.ToLower(k) {
	case "c#", "a#m", "g#mix", "d#dor", "e#phr", "f#lyd", "b#loc":
		return makeAccidentalMap(sh[:7], "is")
	case "f#", "d#m", "c#mix", "g#dor", "a#phr", "blyd", "e#loc":
		return makeAccidentalMap(sh[:6], "is")
	case "b", "g#m", "f#mix", "c#dor", "d#phr", "elyd", "a#loc":
		return makeAccidentalMap(sh[:5], "is")
	case "e", "c#m", "bmix", "f#dor", "g#phr", "alyd", "d#loc":
		return makeAccidentalMap(sh[:4], "is")
	case "a", "f#m", "emix", "bdor", "c#phr", "dlyd", "g#loc":
		return makeAccidentalMap(sh[:3], "is")
	case "d", "bm", "amix", "edor", "f#phr", "glyd", "c#loc":
		return makeAccidentalMap(sh[:2], "is")
	case "g", "em", "dmix", "ador", "bphr", "clyd", "f#loc":
		return makeAccidentalMap(sh[:1], "is")
	case "c", "am", "gmix", "ddor", "ephr", "flyd", "bloc":
		return map[string]string{}
	case "f", "dm", "cmix", "gdor", "aphr", "bblyd", "eloc":
		return makeAccidentalMap(fl[:1], "es")
	case "bb", "gm", "fmix", "cdor", "dphr", "eblyd", "aloc":
		return makeAccidentalMap(fl[:2], "es")
	case "eb", "cm", "bbmix", "fdor", "gphr", "ablyd", "dloc":
		return makeAccidentalMap(fl[:3], "es")
	case "ab", "fm", "ebmix", "bbdor", "cphr", "dblyd", "gloc":
		return makeAccidentalMap(fl[:4], "es")
	case "db", "bbm", "abmix", "ebdor", "fphr", "gblyd", "cloc":
		return makeAccidentalMap(fl[:5], "es")
	case "gb", "ebm", "dbmix", "abdor", "bbphr", "cblyd", "floc":
		return makeAccidentalMap(fl[:6], "es")
	case "cb", "abm", "gbmix", "dbdor", "ebphr", "fblyd", "bbloc":
		return makeAccidentalMap(fl[:7], "es")
	default:
		panic("unknown key " + k)
	}
}

func makeAccidentalMap(k string, sig string) map[string]string {
	acc := make(map[string]string)
	for _, r := range k {
		acc[string(r)] += sig
	}
	return acc
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
	c.pf("  \\header {\n")
	defer c.pf("  }\n")

	c.pf("      piece = %q\n", tune.Title)
	for _, field := range tune.Fields {
		switch field.Tag {
		case abc.FieldComposer.Tag:
			c.pf("      composer = %q\n", field.Value)
		case abc.FieldHistory.Tag:
			c.pf("      history = %q\n", field.Value)
		}
	}
}
