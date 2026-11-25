// Package crossword defines the Raster-JSON format of the crossword door
// (template "crossword", door 16), validates it and generates layouts from a
// word list (tools/crossword).
//
// Raster-JSON (version 1):
//
//	{
//	  "version": 1, "width": W, "height": H,
//	  "cells":   [["A", null, …], …],   // H rows of W cells; one upper-case letter (A–Z, Ä, Ö, Ü, ß) or null = block
//	  "entries": [{"number": 1, "direction": "across"|"down", "row": r, "col": c, "answer": "WORT", "clue": "Hinweis"}]
//	}
//
// Rows and columns are 0-based. Numbering is the standard one: cells are
// scanned row by row, every cell that starts an across or down word (two or
// more letters) gets the next number.
package crossword

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// Format limits.
const (
	Version   = 1
	MinSize   = 1
	MaxSize   = 30
	MaxClue   = 200
	Across    = "across"
	Down      = "down"
	MinAnswer = 2
)

// Raster is a crossword grid with its entries.
type Raster struct {
	Version int         `json:"version"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
	Cells   [][]*string `json:"cells"`
	Entries []Entry     `json:"entries"`
}

// Entry is one word of the grid.
type Entry struct {
	Number    int    `json:"number"`
	Direction string `json:"direction"`
	Row       int    `json:"row"`
	Col       int    `json:"col"`
	Answer    string `json:"answer"`
	Clue      string `json:"clue"`
}

// Parse decodes and validates a Raster-JSON document (unknown fields are errors).
func Parse(raw []byte) (*Raster, error) {
	var r Raster
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("Raster-JSON: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// IsLetter reports whether r is a valid cell letter (A–Z, Ä, Ö, Ü, ß).
func IsLetter(r rune) bool {
	return (r >= 'A' && r <= 'Z') || r == 'Ä' || r == 'Ö' || r == 'Ü' || r == 'ß'
}

// Normalize trims, composes (NFC) and upper-cases a word. ß and ẞ both become
// ß (one cell, never "SS"). The same rules live in crossword-logic.js.
func Normalize(s string) string {
	s = norm.NFC.String(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 'ß', 'ẞ':
			b.WriteRune('ß')
		default:
			b.WriteString(strings.ToUpper(string(r)))
		}
	}
	return b.String()
}

// Letter returns the letter at (row, col) or "" for blocks and out-of-range.
func (r *Raster) Letter(row, col int) string {
	if row < 0 || col < 0 || row >= len(r.Cells) || col >= len(r.Cells[row]) || r.Cells[row][col] == nil {
		return ""
	}
	return *r.Cells[row][col]
}

func (r *Raster) open(row, col int) bool { return r.Letter(row, col) != "" }

// Run is a maximal horizontal or vertical sequence of letters (length >= 2).
type Run struct {
	Number    int
	Direction string
	Row, Col  int
	Answer    string
}

// Runs returns all words of the grid in standard numbering order (row-major,
// across before down for the same number). The grid must be rectangular.
func (r *Raster) Runs() []Run {
	var runs []Run
	n := 0
	for row := 0; row < r.Height; row++ {
		for col := 0; col < r.Width; col++ {
			if !r.open(row, col) {
				continue
			}
			startsA := !r.open(row, col-1) && r.open(row, col+1)
			startsD := !r.open(row-1, col) && r.open(row+1, col)
			if !startsA && !startsD {
				continue
			}
			n++
			if startsA {
				var b strings.Builder
				for c := col; r.open(row, c); c++ {
					b.WriteString(r.Letter(row, c))
				}
				runs = append(runs, Run{n, Across, row, col, b.String()})
			}
			if startsD {
				var b strings.Builder
				for rr := row; r.open(rr, col); rr++ {
					b.WriteString(r.Letter(rr, col))
				}
				runs = append(runs, Run{n, Down, row, col, b.String()})
			}
		}
	}
	return runs
}

// Validate checks the whole document: dimensions, cell letters, that every
// entry's answer matches the cells, that entries and grid words correspond
// one to one with standard numbering, and that no letter is orphaned.
func (r *Raster) Validate() error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	if r.Version != Version {
		add("version muss %d sein (ist %d)", Version, r.Version)
	}
	if r.Width < MinSize || r.Width > MaxSize || r.Height < MinSize || r.Height > MaxSize {
		add("width/height müssen zwischen %d und %d liegen (ist %d×%d)", MinSize, MaxSize, r.Width, r.Height)
		return errors.Join(errs...)
	}
	if len(r.Cells) != r.Height {
		add("cells hat %d Zeilen, height ist %d", len(r.Cells), r.Height)
		return errors.Join(errs...)
	}
	letters := 0
	for i, row := range r.Cells {
		if len(row) != r.Width {
			add("cells[%d] hat %d Spalten, width ist %d", i, len(row), r.Width)
			continue
		}
		for j, c := range row {
			if c == nil {
				continue
			}
			letters++
			if utf8.RuneCountInString(*c) != 1 || !IsLetter([]rune(*c)[0]) {
				add("cells[%d][%d] = %q: erwartet genau einen Großbuchstaben (A–Z, Ä, Ö, Ü, ß) oder null", i, j, *c)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	if letters == 0 {
		add("das Raster enthält keine Buchstaben")
	}
	if len(r.Entries) == 0 {
		add("entries ist leer")
	}

	type key struct {
		dir      string
		row, col int
	}
	runs := map[key]Run{}
	for _, run := range r.Runs() {
		runs[key{run.Direction, run.Row, run.Col}] = run
	}
	seen := map[key]bool{}
	numbers := map[[2]int]int{} // (row, col) -> number used by entries
	covered := map[[2]int]bool{}
	for i, e := range r.Entries {
		n := fmt.Sprintf("entries[%d] (%d %s)", i, e.Number, dirDE(e.Direction))
		if e.Direction != Across && e.Direction != Down {
			add("entries[%d]: direction muss %q oder %q sein (ist %q)", i, Across, Down, e.Direction)
			continue
		}
		if e.Row < 0 || e.Row >= r.Height || e.Col < 0 || e.Col >= r.Width {
			add("%s: Start (%d, %d) liegt außerhalb des Rasters", n, e.Row, e.Col)
			continue
		}
		if strings.TrimSpace(e.Clue) == "" {
			add("%s: clue fehlt", n)
		}
		if utf8.RuneCountInString(e.Clue) > MaxClue {
			add("%s: clue ist zu lang (max. %d Zeichen)", n, MaxClue)
		}
		ans := []rune(e.Answer)
		if len(ans) < MinAnswer {
			add("%s: answer %q ist zu kurz", n, e.Answer)
			continue
		}
		dr, dc := 0, 1
		if e.Direction == Down {
			dr, dc = 1, 0
		}
		endR, endC := e.Row+dr*(len(ans)-1), e.Col+dc*(len(ans)-1)
		if endR >= r.Height || endC >= r.Width {
			add("%s: %q ragt über den Rand des Rasters", n, e.Answer)
			continue
		}
		ok := true
		for k, ch := range ans {
			rr, cc := e.Row+dr*k, e.Col+dc*k
			if got := r.Letter(rr, cc); got != string(ch) {
				add("%s: answer %q passt nicht zum Raster (Feld %d, %d ist %q, erwartet %q)", n, e.Answer, rr, cc, orBlock(got), string(ch))
				ok = false
				break
			}
		}
		if !ok {
			continue
		}
		if r.open(e.Row-dr, e.Col-dc) || r.open(endR+dr, endC+dc) {
			add("%s: %q ist nur ein Teil eines längeren Worts im Raster", n, e.Answer)
			continue
		}
		k := key{e.Direction, e.Row, e.Col}
		if seen[k] {
			add("%s: doppelter Eintrag", n)
			continue
		}
		seen[k] = true
		run := runs[k]
		if e.Number != run.Number {
			add("%s: Nummer passt nicht zur Standard-Nummerierung (erwartet %d)", n, run.Number)
		}
		pos := [2]int{e.Row, e.Col}
		if prev, dup := numbers[pos]; dup && prev != e.Number {
			add("%s: gleiches Startfeld wie Nummer %d, aber andere Nummer", n, prev)
		}
		numbers[pos] = e.Number
		for k := range ans {
			covered[[2]int{e.Row + dr*k, e.Col + dc*k}] = true
		}
	}
	// every word in the grid needs an entry (otherwise letters touch illegally)
	missing := make([]Run, 0)
	for k, run := range runs {
		if !seen[k] {
			missing = append(missing, run)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Number != missing[j].Number {
			return missing[i].Number < missing[j].Number
		}
		return missing[i].Direction < missing[j].Direction
	})
	for _, run := range missing {
		add("Wort %q (%d %s) steht im Raster, hat aber keinen Eintrag – berührende Buchstaben?", run.Answer, run.Number, dirDE(run.Direction))
	}
	for i := 0; i < r.Height; i++ {
		for j := 0; j < r.Width; j++ {
			if r.open(i, j) && !covered[[2]int{i, j}] && len(missing) == 0 {
				add("Buchstabe %q in Feld (%d, %d) gehört zu keinem Eintrag", r.Letter(i, j), i, j)
			}
		}
	}
	return errors.Join(errs...)
}

func orBlock(s string) string {
	if s == "" {
		return "■"
	}
	return s
}

func dirDE(d string) string {
	switch d {
	case Across:
		return "waagerecht"
	case Down:
		return "senkrecht"
	}
	return d
}

// SortEntries orders entries by number, across before down.
func SortEntries(es []Entry) {
	sort.SliceStable(es, func(i, j int) bool {
		if es[i].Number != es[j].Number {
			return es[i].Number < es[j].Number
		}
		return es[i].Direction == Across && es[j].Direction == Down
	})
}

// Preview renders the grid as ASCII art (blocks as '·', letters as-is).
func (r *Raster) Preview() string {
	var b strings.Builder
	for _, row := range r.Cells {
		for j, c := range row {
			if j > 0 {
				b.WriteByte(' ')
			}
			if c == nil {
				b.WriteString("·")
			} else {
				b.WriteString(*c)
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
