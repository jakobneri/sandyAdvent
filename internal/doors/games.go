package doors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Params of the game templates (M2). Every game has an optional intro and a
// final reveal text shown after completion. Asset paths are relative to
// assets_dir and must be listed in the door's "assets".

// decodeParams decodes raw params strictly (unknown fields are errors, so
// typos like "qestion" do not silently disappear).
func decodeParams(raw json.RawMessage, v any) error {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("params fehlen")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("params: %w", err)
	}
	return nil
}

// checker collects validation errors.
type checker struct{ errs []error }

func (c *checker) add(format string, a ...any) { c.errs = append(c.errs, fmt.Errorf(format, a...)) }

func (c *checker) need(name, v string) {
	if strings.TrimSpace(v) == "" {
		c.add("%s fehlt", name)
	}
}

func (c *checker) asset(name, v string) {
	if v == "" {
		return
	}
	if _, err := CleanAssetPath(v); err != nil {
		c.add("%s %q: %v (relativer Pfad im assets_dir erwartet)", name, v, err)
	}
}

func (c *checker) maxLen(name, v string, n int) {
	if utf8.RuneCountInString(v) > n {
		c.add("%s ist zu lang (max. %d Zeichen)", name, n)
	}
}

func (c *checker) err() error { return errors.Join(c.errs...) }

// ---------------------------------------------------------------------------
// quiz

// QuizParams are the params of template "quiz".
type QuizParams struct {
	Intro            string         `json:"intro,omitempty"`
	Questions        []QuizQuestion `json:"questions"`
	ShuffleQuestions bool           `json:"shuffle_questions,omitempty"`
	ShuffleOptions   bool           `json:"shuffle_options,omitempty"`
	Results          []QuizResult   `json:"results,omitempty"`
	ClosingText      string         `json:"closing_text,omitempty"`
}

// QuizQuestion is one multiple-choice question. For "Wer hat's gesagt?" the
// quote is shown as a quote card and question may be left out.
type QuizQuestion struct {
	Question string   `json:"question,omitempty"`
	Quote    string   `json:"quote,omitempty"`
	Image    string   `json:"image,omitempty"`
	Options  []string `json:"options"`
	Answer   *int     `json:"answer"`
	Explain  string   `json:"explain,omitempty"`
}

// QuizResult is the closing text for scores >= MinScore.
type QuizResult struct {
	MinScore int    `json:"min_score"`
	Text     string `json:"text"`
}

// DefaultQuoteQuestion is used when a quote question has no question text.
const DefaultQuoteQuestion = "Wer hat's gesagt?"

// ParseQuiz decodes and validates quiz params.
func ParseQuiz(raw json.RawMessage) (*QuizParams, error) {
	p := &QuizParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	if len(p.Questions) == 0 {
		c.add("questions ist leer – mindestens eine Frage nötig")
	}
	for i, q := range p.Questions {
		n := fmt.Sprintf("Frage %d", i+1)
		if strings.TrimSpace(q.Question) == "" && strings.TrimSpace(q.Quote) == "" {
			c.add("%s: question (oder quote) fehlt", n)
		}
		c.asset(n+": image", q.Image)
		if len(q.Options) < 2 || len(q.Options) > 6 {
			c.add("%s: options braucht 2 bis 6 Antworten (hat %d)", n, len(q.Options))
		}
		seen := map[string]bool{}
		for j, o := range q.Options {
			o = strings.TrimSpace(o)
			if o == "" {
				c.add("%s: options[%d] ist leer", n, j)
				continue
			}
			c.maxLen(fmt.Sprintf("%s: options[%d]", n, j), o, 120)
			if seen[strings.ToLower(o)] {
				c.add("%s: Antwort %q doppelt", n, o)
			}
			seen[strings.ToLower(o)] = true
		}
		switch {
		case q.Answer == nil:
			c.add("%s: answer fehlt (Index der richtigen Antwort, 0 = erste)", n)
		case *q.Answer < 0 || *q.Answer >= len(q.Options):
			c.add("%s: answer %d liegt außerhalb von options (0 bis %d)", n, *q.Answer, len(q.Options)-1)
		}
	}
	if len(p.Results) > 0 {
		hasZero := false
		seen := map[int]bool{}
		for i, r := range p.Results {
			if r.MinScore < 0 || r.MinScore > len(p.Questions) {
				c.add("results[%d]: min_score %d muss zwischen 0 und %d (Anzahl Fragen) liegen", i, r.MinScore, len(p.Questions))
			}
			if seen[r.MinScore] {
				c.add("results[%d]: min_score %d doppelt", i, r.MinScore)
			}
			seen[r.MinScore] = true
			hasZero = hasZero || r.MinScore == 0
			c.need(fmt.Sprintf("results[%d].text", i), r.Text)
		}
		if !hasZero {
			c.add("results braucht einen Eintrag mit min_score 0 (sonst fehlt bei 0 Punkten der Abschlusstext)")
		}
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// AssetRefs returns the referenced images.
func (p *QuizParams) AssetRefs() []string {
	var refs []string
	for _, q := range p.Questions {
		if q.Image != "" {
			refs = append(refs, q.Image)
		}
	}
	return refs
}

// ResultText returns the text of the result with the highest min_score <= score.
func (p *QuizParams) ResultText(score int) string {
	best, text := -1, ""
	for _, r := range p.Results {
		if r.MinScore <= score && r.MinScore > best {
			best, text = r.MinScore, r.Text
		}
	}
	return text
}

// ---------------------------------------------------------------------------
// memory

// Memory limits.
const (
	MemoryMinPairs = 2
	MemoryMaxPairs = 18
	MemoryMaxPeek  = 10000
)

// MemoryParams are the params of template "memory".
type MemoryParams struct {
	Intro     string   `json:"intro,omitempty"`
	Pairs     int      `json:"pairs"`
	Images    []string `json:"images"`
	CardBack  string   `json:"card_back,omitempty"`
	PeekMS    int      `json:"peek_ms,omitempty"`
	ShowMoves *bool    `json:"show_moves,omitempty"`
	ShowTimer bool     `json:"show_timer,omitempty"`
	WinText   string   `json:"win_text,omitempty"`
}

// ParseMemory decodes and validates memory params.
func ParseMemory(raw json.RawMessage) (*MemoryParams, error) {
	p := &MemoryParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	if p.Pairs < MemoryMinPairs || p.Pairs > MemoryMaxPairs {
		c.add("pairs muss zwischen %d und %d liegen (ist %d)", MemoryMinPairs, MemoryMaxPairs, p.Pairs)
	}
	if len(p.Images) < p.Pairs {
		c.add("images braucht mindestens so viele Bilder wie pairs (%d), hat %d", p.Pairs, len(p.Images))
	}
	seen := map[string]bool{}
	for i, img := range p.Images {
		if img == "" {
			c.add("images[%d] ist leer", i)
			continue
		}
		c.asset(fmt.Sprintf("images[%d]", i), img)
		if seen[img] {
			c.add("images[%d]: %q doppelt (jedes Paar braucht ein eigenes Bild)", i, img)
		}
		seen[img] = true
	}
	c.asset("card_back", p.CardBack)
	if p.PeekMS < 0 || p.PeekMS > MemoryMaxPeek {
		c.add("peek_ms muss zwischen 0 und %d liegen", MemoryMaxPeek)
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// Moves reports whether the move counter is shown (default true).
func (p *MemoryParams) Moves() bool { return p.ShowMoves == nil || *p.ShowMoves }

// AssetRefs returns the referenced images.
func (p *MemoryParams) AssetRefs() []string {
	refs := append([]string(nil), p.Images...)
	if p.CardBack != "" {
		refs = append(refs, p.CardBack)
	}
	return refs
}

// ---------------------------------------------------------------------------
// wordle

// Wordle limits.
const (
	WordleMinLen     = 3
	WordleMaxLen     = 12
	WordleMaxRounds  = 10
	WordleDefGuesses = 6
)

// WordleParams are the params of template "wordle".
type WordleParams struct {
	Intro         string       `json:"intro,omitempty"`
	Words         []WordleWord `json:"words"`
	MaxGuesses    int          `json:"max_guesses,omitempty"`
	AllowAnyGuess *bool        `json:"allow_any_guess,omitempty"`
	WinText       string       `json:"win_text,omitempty"`
}

// WordleWord is one round. Word is normalized (NFC, upper case, ß stays ß).
type WordleWord struct {
	Word string `json:"word"`
	Hint string `json:"hint,omitempty"`
}

// ParseWordle decodes and validates wordle params; words are normalized.
func ParseWordle(raw json.RawMessage) (*WordleParams, error) {
	p := &WordleParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	if len(p.Words) == 0 {
		c.add("words ist leer – mindestens ein Wort nötig")
	}
	if len(p.Words) > WordleMaxRounds {
		c.add("words: höchstens %d Runden", WordleMaxRounds)
	}
	for i := range p.Words {
		w := NormalizeWord(p.Words[i].Word)
		p.Words[i].Word = w
		n := utf8.RuneCountInString(w)
		switch {
		case w == "":
			c.add("words[%d].word fehlt", i)
		case n < WordleMinLen || n > WordleMaxLen:
			c.add("words[%d]: %q hat %d Buchstaben, erlaubt sind %d bis %d", i, w, n, WordleMinLen, WordleMaxLen)
		}
		for _, r := range w {
			if !IsWordleLetter(r) {
				c.add("words[%d]: %q enthält %q – nur Buchstaben A–Z, Ä, Ö, Ü und ß (keine Leerzeichen/Bindestriche)", i, w, string(r))
				break
			}
		}
	}
	if p.MaxGuesses == 0 {
		p.MaxGuesses = WordleDefGuesses
	}
	if p.MaxGuesses < 1 || p.MaxGuesses > 10 {
		c.add("max_guesses muss zwischen 1 und 10 liegen (Standard %d)", WordleDefGuesses)
	}
	if p.AllowAnyGuess != nil && !*p.AllowAnyGuess {
		c.add("allow_any_guess: false wird nicht unterstützt – es gibt kein Wörterbuch (Insider-Wörter), jeder Versuch mit passender Länge zählt")
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// IsWordleLetter reports whether r is on the wordle keyboard (A–Z, ÄÖÜ, ß).
func IsWordleLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || r == 'Ä' || r == 'Ö' || r == 'Ü' || r == 'ß'
}

var umlautCompose = map[rune]rune{'A': 'Ä', 'O': 'Ö', 'U': 'Ü', 'a': 'ä', 'o': 'ö', 'u': 'ü'}

// NormalizeWord trims, composes decomposed umlauts (NFC for the letters that
// matter here) and upper-cases. ß and ẞ both become ß (one letter, never "SS").
// The same rules live in web/static/js/wordle-logic.js.
func NormalizeWord(s string) string {
	rs := []rune(strings.TrimSpace(s))
	var b strings.Builder
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if i+1 < len(rs) && rs[i+1] == '̈' {
			if c, ok := umlautCompose[r]; ok {
				r = c
				i++
			}
		}
		switch r {
		case 'ß', 'ẞ':
			b.WriteRune('ß')
		case 'ä':
			b.WriteRune('Ä')
		case 'ö':
			b.WriteRune('Ö')
		case 'ü':
			b.WriteRune('Ü')
		default:
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// puzzle

// Puzzle modes.
const (
	PuzzleJigsaw  = "jigsaw"
	PuzzleSliding = "sliding"
)

// PuzzleParams are the params of template "puzzle".
type PuzzleParams struct {
	Intro       string `json:"intro,omitempty"`
	Mode        string `json:"mode"`
	Image       string `json:"image"`
	Grid        int    `json:"grid,omitempty"`
	ShowPreview *bool  `json:"show_preview,omitempty"`
	WinText     string `json:"win_text,omitempty"`
}

// ParsePuzzle decodes and validates puzzle params and applies the default grid.
func ParsePuzzle(raw json.RawMessage) (*PuzzleParams, error) {
	p := &PuzzleParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	switch p.Mode {
	case PuzzleJigsaw:
		if p.Grid == 0 {
			p.Grid = 4
		}
	case PuzzleSliding:
		if p.Grid == 0 {
			p.Grid = 3
		}
	case "":
		c.add("mode fehlt (%q oder %q)", PuzzleJigsaw, PuzzleSliding)
	default:
		c.add("mode %q unbekannt – erlaubt: %q oder %q", p.Mode, PuzzleJigsaw, PuzzleSliding)
	}
	if p.Grid != 0 && (p.Grid < 3 || p.Grid > 6) {
		c.add("grid muss zwischen 3 und 6 liegen (ist %d)", p.Grid)
	}
	c.need("image", p.Image)
	c.asset("image", p.Image)
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// Preview reports whether the preview button is shown (default true).
func (p *PuzzleParams) Preview() bool { return p.ShowPreview == nil || *p.ShowPreview }

// AssetRefs returns the referenced image.
func (p *PuzzleParams) AssetRefs() []string { return []string{p.Image} }
