package abc

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
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

var rxNote = regexp.MustCompile(`^([\_\^=]*[a-gA-G][,']*|\[(?:[\_\^=]*[a-gA-G][,']*)+\]|[yzZxX])([0-9]*)(\/*)([0-9]*)([<>]*)(\-?)`)
var rxNotePitch = regexp.MustCompile(`([\_\^=]*)([a-gA-G])([,']*)`)

func (p *Parser) TryParseNote(line string) string {
	if match := rxNote.FindStringSubmatch(line); len(match) > 0 {
		note := match[1]
		duration := match[2]
		halving := match[3]
		divider := match[4]
		syncopate := match[5]
		tie := match[6]

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

		if isRest(note) {
			p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
				Kind:        KindRest,
				Value:       note,
				Duration:    *dur,
				Tie:         tie != "",
				Syncopation: sync,
			})
			return strings.TrimLeft(line[len(match[0]):], " ")
		}

		var notes []Note
		for _, match := range rxNotePitch.FindAllStringSubmatch(note, -1) {
			note := Note{}

			note.Accidentals = match[1]

			for _, v := range match[3] {
				switch v {
				case ',':
					note.Octave--
				case '\'':
					note.Octave++
				}
			}

			note.Pitch = strings.ToLower(match[2])

			if unicode.IsLower(rune(match[2][0])) {
				note.Octave++
			}

			notes = append(notes, note)
		}

		if len(notes) == 0 {
			panic("failed to parse note " + note)
		}

		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:        KindNote,
			Notes:       notes,
			Duration:    *dur,
			Tie:         tie != "",
			Syncopation: sync,
		})

		return strings.TrimLeft(line[len(match[0]):], " ")
	}

	return line
}

func isRest(v string) bool {
	switch v {
	case "y", "z", "Z", "x", "X":
		return true
	}
	return false
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

// `:|: [1-2`
var rxBar = regexp.MustCompile(`^([\:\|\[\]]*[\:\|\]])(?:(?:\s*\[)?([0-9\-\,]+)|(\])|)`)

func (p *Parser) TryParseBar(line string) string {
	if match := rxBar.FindStringSubmatch(line); len(match) > 0 {
		bar := match[1]
		volta := match[2]
		end := match[3]
		if volta != "" {
			volta = strings.TrimLeft(volta, " [")
		}

		p.Stave.Symbols = append(p.Stave.Symbols, Symbol{
			Kind:  KindBar,
			Value: bar,
			Volta: volta,

			CloseVolta: end != "",
		})

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

	Meter Meter

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

func (fields Fields) ByTag(tag string) (Field, bool) {
	for _, f := range fields {
		if f.Tag == tag {
			return f, true
		}
	}
	return Field{}, false
}

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
	Notes       []Note
	Duration    big.Rat
	Syncopation int

	Tie   bool
	Tag   string
	Volta string

	CloseVolta bool
}

type Note struct {
	Accidentals string
	Pitch       string
	Octave      int
}

type Kind byte

func (k Kind) String() string {
	switch k {
	case KindText:
		return "Text"
	case KindNote:
		return "Note"
	case KindRest:
		return "Rest"
	case KindBar:
		return "Bar"
	case KindDeco:
		return "Deco"
	case KindField:
		return "Field"
	default:
		return fmt.Sprintf("Kind(%d)", k)
	}
}

const (
	KindText  = Kind(1)
	KindNote  = Kind(2)
	KindRest  = Kind(3)
	KindBar   = Kind(4)
	KindDeco  = Kind(5)
	KindField = Kind(6)
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
	FieldTypeUnknown     = FieldType(2)
)

var (
	FieldArea            = FieldDef{"A", "area", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "A:Donegal, A:Bampton"} // outdated information field syntax
	FieldBook            = FieldDef{"B", "book", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "B:O'Neills"}
	FieldComposer        = FieldDef{"C", "composer", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "C:Robert Jones, C:Trad."}
	FieldDiscography     = FieldDef{"D", "discography", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "D:Chieftains IV"}
	FieldFile            = FieldDef{"F", "file url", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "F:http://a.b.c/file.abc"}
	FieldGroup           = FieldDef{"G", "group", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "G:flute"}
	FieldHistory         = FieldDef{"H", "history", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "H:The story behind this tune ..."}
	FieldInstruction     = FieldDef{"I", "instruction", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "I:papersize A4, I:newpage"}
	FieldKey             = FieldDef{"K", "key", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "K:G, K:Dm, K:AMix"}
	FieldUnitNoteLength  = FieldDef{"L", "unit note length", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "L:1/4"}
	FieldMeter           = FieldDef{"M", "meter", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "M:3/4, M:4/4"}
	FieldMacro           = FieldDef{"m", "macro", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "m: ~G2 = {A}G{F}G"}
	FieldNotes           = FieldDef{"N", "notes", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeString, "N:see also O'Neills - 234"}
	FieldOrigin          = FieldDef{"O", "origin", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "O:UK; Yorkshire; Bradford"}
	FieldParts           = FieldDef{"P", "parts", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "P:A, P:ABAC, P:(A2B)3"}
	FieldTempo           = FieldDef{"Q", "tempo", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "Q:\"allegro\" 1/4=120"}
	FieldRhythm          = FieldDef{"R", "rhythm", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeString, "R:R, R:reel"}
	FieldRemark          = FieldDef{"r", "remark", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeUnknown, "r:I love abc"}
	FieldSource          = FieldDef{"S", "source", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "S:collected in Brittany"}
	FieldSymbolLine      = FieldDef{"s", "symbol line", FieldInTuneBody, FieldTypeInstruction, "s: !pp! ** !f!"}
	FieldTuneTitle       = FieldDef{"T", "tune title", FieldInTuneHeader | FieldInTuneBody, FieldTypeString, "T:Paddy O'Rafferty"}
	FieldUserDefined     = FieldDef{"U", "user defined", FieldInFileHeader | FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "U: T = !trill!"}
	FieldVoice           = FieldDef{"V", "voice", FieldInTuneHeader | FieldInTuneBody | FieldInline, FieldTypeInstruction, "V:4 clef=bass"}
	FieldWords           = FieldDef{"W", "words", FieldInTuneHeader | FieldInTuneBody, FieldTypeString, "W:lyrics printed after the end of the tune"}
	FieldWords2          = FieldDef{"w", "words", FieldInTuneHeader | FieldInTuneBody, FieldTypeString, "W:lyrics printed after the end of the tune"}
	FieldReferenceNumber = FieldDef{"X", "reference number", FieldInTuneHeader, FieldTypeInstruction, "X:1, X:2"}
	FieldTranscription   = FieldDef{"Z", "transcription", FieldInFileHeader | FieldInTuneHeader, FieldTypeString, "Z:John Smith, <j.s@example.com>"}
)

var FieldDefs = []FieldDef{
	FieldArea,
	FieldBook,
	FieldComposer,
	FieldDiscography,
	FieldFile,
	FieldGroup,
	FieldHistory,
	FieldInstruction,
	FieldKey,
	FieldUnitNoteLength,
	FieldMeter,
	FieldMacro,
	FieldNotes,
	FieldOrigin,
	FieldParts,
	FieldTempo,
	FieldRhythm,
	FieldRemark,
	FieldSource,
	FieldSymbolLine,
	FieldTuneTitle,
	FieldUserDefined,
	FieldVoice,
	FieldWords,
	FieldWords,
	FieldReferenceNumber,
	FieldTranscription,
}

const (
	AccidentalFlat    = '_'
	AccidentalSharp   = '^'
	AccidentalNatural = '='
)
