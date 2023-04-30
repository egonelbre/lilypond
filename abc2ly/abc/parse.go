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
	Book     *TuneBook
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
	tune := &Tune{Raw: content}

	inheader := true

	var noteLength big.Rat
	for _, line := range strings.Split(content, "\n") {
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
					tune.ID = value
				case "T":
					tune.Title = value
				case "L":
					tune.NoteLength = ParseNoteLength(value)
					noteLength = tune.NoteLength
				case "M":
					tune.Meter = ParseMeter(value)
				case "K":
					tune.Key = value
					inheader = false
				}

				tune.Fields = append(tune.Fields, Field{
					Tag:   match[1],
					Value: value,
				})
				continue
			}

			// unknown line without detecting `K:` header
			inheader = false
		}

		if noteLength == (big.Rat{}) {
			noteLength = *big.NewRat(1, 4)
		}
	}

	p.Book.Tunes = append(p.Book.Tunes, tune)
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
	return strings.TrimRight(line, " \t\n")
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
	Kind     Kind
	Text     string
	Duration big.Rat
	Tie      bool
}

type Kind byte

const (
	KindText = Kind(1)
	KindNote = Kind(2)
	KindBar  = Kind(3)
	KindDeco = Kind(4)
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
