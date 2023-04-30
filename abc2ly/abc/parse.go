package abc

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
)

var rxHeader = regexp.MustCompile(`^([a-zA-Z]):(.*)$`)
var rxNewTune = regexp.MustCompile(`(?m)^X:`) // TODO: avoid regex

type Parser struct {
	Book  *TuneBook
	Tune  *Tune
	Stave *Stave

	Warnings []Warning
}

func NewParser() *Parser {
	return &Parser{
		Book: &TuneBook{},
	}
}

type Warning struct {
	Line, Column int
	Message      string
}

func Parse(content string) (*TuneBook, []Warning) {
	p := NewParser()
	p.ParseBook(content)
	return p.Book, p.Warnings
}

func SplitTuneBook(s string) []string {
	var tunes []string

	start := 0
	for _, loc := range rxNewTune.FindAllStringIndex(s, -1) {
		if tune := strings.TrimSpace(s[start:loc[0]]); tune != "" {
			tunes = append(tunes, tune)
		}
		start = loc[0]
	}
	if tune := strings.TrimSpace(s[start:]); tune != "" {
		tunes = append(tunes, tune)
	}

	return tunes
}

func (p *Parser) ParseBook(content string) {
	for _, tune := range SplitTuneBook(content) {
		p.ParseTune(tune)
	}
}

func (p *Parser) ParseTune(content string) {
	p.Tune = &Tune{Raw: content}

	inheader := true

	for _, line := range strings.Split(content, "\n") {
		line = trimComment(line)
		line = trimTrailingWhitespace(line)
		if line == "" {
			continue
		}
		if line[0] == '%' {
			continue
		}

		if inheader {
			match := rxHeader.FindStringSubmatch(line)
			if len(match) > 0 {
				value := strings.TrimSpace(match[2])
				switch match[1] {
				case "X":
					p.Tune.ID = value
				case "T":
					p.Tune.Title = value
				case "L":
					p.Tune.NoteLength = ParseNoteLength(value)
				case "M":
					p.Tune.Meter = ParseMeter(value)
				case "K":
					p.Tune.Key = value
					inheader = false
				}

				p.Tune.Fields = append(p.Tune.Fields, Field{
					Tag:   match[1],
					Value: value,
				})
				continue
			}

			// unknown line without detecting `K:` header
			inheader = false
		}

		p.Stave = &Stave{}
		prevLine := ""
		for prevLine != line {
			prevLine = line

			line = p.TryParseField(line)
			line = p.TryParseDeco(line)
			line = p.TryParseNote(line)
			line = p.TryParseText(line)
			line = p.TryParseBar(line)

			// TODO: handle note groups
			// TODO: handle tuplets
			// TODO: handle slurs

			line = strings.TrimLeft(line, " \t")
		}
		if len(p.Stave.Symbols) > 0 {
			p.Tune.Body.Staves = append(p.Tune.Body.Staves, *p.Stave)
		}
		p.Stave = nil
		if line != "" {
			p.Warnings = append(p.Warnings, Warning{
				Message: fmt.Sprintf("unable to parse %q", line),
			})
		}
	}

	p.Book.Tunes = append(p.Book.Tunes, p.Tune)
}

var rxInlineField = regexp.MustCompile(`^\[([a-zA-Z]):([^\]]*)\]`)

func (p *Parser) TryParseField(line string) string {
	if match := rxInlineField.FindStringSubmatch(line); len(match) > 0 {
		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:  KindField,
			Tag:   match[1],
			Value: strings.TrimSpace(match[2]),
		})
		return strings.TrimLeft(line[len(match[0]):], " ")
	}

	return line
}

var rxDeco = regexp.MustCompile(`^([\.~HLMOPSTuv]|![^!]+!)`)

func (p *Parser) TryParseDeco(line string) string {
	if match := rxDeco.FindStringSubmatch(line); len(match) > 0 {
		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:  KindDeco,
			Value: strings.TrimSpace(match[1]),
		})
		return strings.TrimLeft(line[len(match[0]):], " ")
	}

	return line
}

var rxNote = regexp.MustCompile(`^([\_\^=]*)([a-gA-G][,']*|\[(?:[a-gA-G][,']*)+\]|[yzZxX])([0-9]*)(\/*)([0-9]*)([<>]*)(\-?)`)

func (p *Parser) TryParseNote(line string) string {
	if match := rxNote.FindStringSubmatch(line); len(match) > 0 {
		accidentals := match[1]
		note := match[2]
		duration := match[3]
		halving := match[4]
		divider := match[5]
		syncopate := match[6]
		tie := match[7]

		dur := big.NewRat(1, 1)
		if duration != "" {
			v, err := strconv.Atoi(duration)
			if err != nil {
				// TODO: fix error handling
				panic(err)
			}
			dur.Mul(dur, big.NewRat(int64(v), 1))
		}
		if len(halving) == 1 && divider != "" {
			div, err := strconv.Atoi(divider)
			if err != nil {
				// TODO: fix error handling
				panic(err)
			}
			dur.Mul(dur, big.NewRat(1, int64(div)))
		} else {
			for k := 0; k < len(halving); k++ {
				dur.Mul(dur, big.NewRat(1, 2))
			}
		}

		sync := 0
		for _, b := range syncopate {
			switch b {
			case '<':
				sync--
			case '>':
				sync++
			}
		}

		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:        KindNote,
			Value:       note,
			Duration:    *dur,
			Accidentals: accidentals,
			Tie:         tie != "",
			Syncopation: sync,
		})

		return strings.TrimLeft(line[len(match[0]):], " ")
	}

	return line
}

var rxText = regexp.MustCompile(`^"([^"]*)"`) // TODO: handle escaping "

func (p *Parser) TryParseText(line string) string {
	if match := rxText.FindStringSubmatch(line); len(match) > 0 {
		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:  KindText,
			Value: match[1],
		})
		return strings.TrimLeft(line[len(match[0]):], " ")
	}

	return line
}

var rxBar = regexp.MustCompile(`^([\:\|\[\]]*[\:\|\]])(?:(?:\s*\[)?([0-9\-\,]+)|(\])|)`)

func (p *Parser) TryParseBar(line string) string {
	if match := rxBar.FindStringSubmatch(line); len(match) > 0 {
		bar := match[1]
		volta := match[2]
		end := match[3]
		if volta != "" {
			volta = strings.TrimLeft(volta, " [")
		}

		switch bar {
		case "::", ":|:", ":||:":
			p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
				Kind:  KindBar,
				Value: ":|",
				Volta: end,
			}, Symbol{
				Kind:  KindBar,
				Value: "|:",
				Volta: volta,
			})
		default:
			p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
				Kind:  KindBar,
				Value: match[1],
				Volta: volta + end,
			})
		}

		return strings.TrimLeft(line[len(match[0]):], " ")
	}
	return line
}

func ParseMeter(s string) (m Meter) {
	beatsPerMeasure, beatLength, ok := strings.Cut(s, "/")
	if !ok {
		panic("invalid meter " + s)
	}

	var err1, err2 error
	m.BeatsPerMeasure, err1 = strconv.Atoi(beatsPerMeasure)
	m.BeatLength, err2 = strconv.Atoi(beatLength)

	if err1 != nil || err2 != nil {
		// TODO: error handling
		panic(fmt.Errorf("%w\n%w", err1, err2))
	}

	return m
}

func ParseNoteLength(s string) (n big.Rat) {
	as, bs, ok := strings.Cut(s, "/")
	if !ok {
		panic("invalid meter " + s)
	}

	a, err1 := strconv.Atoi(as)
	b, err2 := strconv.Atoi(bs)

	if err1 != nil || err2 != nil {
		// TODO: error handling
		panic(fmt.Errorf("%w\n%w", err1, err2))
	}

	return *big.NewRat(int64(a), int64(b))
}

func trimTrailingWhitespace(line string) string {
	return strings.TrimRight(line, " \t\n\r")
}

func trimComment(line string) string {
	// TODO: handle escaping of %
	p := strings.IndexByte(line, '%')
	if p < 0 {
		return line
	}
	return line[:p]
}

type TuneBook struct {
	Tunes []*Tune
}

type Tune struct {
	ID     string
	Title  string
	Key    string
	Fields Fields

	NoteLength big.Rat
	Meter      Meter

	Body TuneBody

	Raw string
}

type Meter struct {
	BeatsPerMeasure int
	BeatLength      int
}

type TuneBody struct {
	Staves []Stave
}

type Fields []Field

type Field struct {
	Tag   string
	Value string
}

type Stave struct {
	Symbols []Symbol
}

type Symbol struct {
	Kind        Kind
	Value       string
	Duration    big.Rat
	Syncopation int

	Accidentals string
	Tie         bool
	Tag         string
	Volta       string
}

type Kind byte

func (k Kind) String() string {
	switch k {
	case KindText:
		return "Text"
	case KindNote:
		return "Note"
	case KindBar:
		return "Bar"
	case KindDeco:
		return "Deco"
	case KindField:
		return "Field"
	case KindVolta:
		return "Volta"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

const (
	KindText  = Kind(1)
	KindNote  = Kind(2)
	KindBar   = Kind(3)
	KindDeco  = Kind(4)
	KindField = Kind(5)
	KindVolta = Kind(6)
)

type FieldDef struct {
	Tag     string
	Full    string
	Flags   FieldFlags
	Type    FieldType
	Example string
}

type FieldFlags byte

const (
	FieldInFileHeader = FieldFlags(1)
	FieldInTuneHeader = FieldFlags(2)
	FieldInTuneBody   = FieldFlags(4)
	FieldInline       = FieldFlags(8)
)

type FieldType byte

const (
	FieldTypeString      = FieldType(0)
	FieldTypeInstruction = FieldType(1)
	FieldTypeAny         = FieldType(2)
)

var FieldDefs = []FieldDef{
	{"A", "area", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "A:Donegal, A:Bampton"}, // outdated information field syntax
	{"B", "book", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "B:O'Neills"},
	{"C", "composer", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "C:Robert Jones, C:Trad."},
	{"D", "discography", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "D:Chieftains IV"},
	{"F", "file url", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "F:http://a.b.c/file.abc"},
	{"G", "group", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "G:flute"},
	{"H", "history", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "H:The story behind this tune ..."},
	{"I", "instruction", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "I:papersize A4, I:newpage"},
	{"K", "key", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "K:G, K:Dm, K:AMix"},
	{"L", "unit note length", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "L:1/4"},
	{"M", "meter", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "M:3/4, M:4/4"},
	{"m", "macro", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "m: ~G2 = {A}G{F}G"},
	{"N", "notes", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeString, "N:see also O'Neills - 234"},
	{"O", "origin", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "O:UK; Yorkshire; Bradford"},
	{"P", "parts", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "P:A, P:ABAC, P:(A2B)3"},
	{"Q", "tempo", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "Q:\"allegro\" 1/4=120"},
	{"R", "rhythm", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeString, "R:R, R:reel"},
	{"r", "remark", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeAny, "r:I love abc"},
	{"S", "source", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "S:collected in Brittany"},
	{"s", "symbol line", FieldInTuneBody, FieldTypeInstruction, "s: !pp! ** !f!"},
	{"T", "tune title", FieldInTuneHeader | FieldInTuneBody, FieldTypeString, "T:Paddy O'Rafferty"},
	{"U", "user defined", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "U: T = !trill!"},
	{"V", "voice", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "V:4 clef=bass"},
	{"W", "words", FieldInTuneHeader | FieldInTuneBody, FieldTypeString, "W:lyrics printed after the end of the tune"},
	{"w", "words", FieldInTuneBody, FieldTypeString, "w:lyrics printed aligned with the notes of a tune"},
	{"X", "reference number", FieldInTuneHeader, FieldTypeInstruction, "X:1, X:2"},
	{"Z", "transcription", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "Z:John Smith, <j.s@example.com>"},
}
