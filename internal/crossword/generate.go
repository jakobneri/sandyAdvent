package crossword

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Word is one input word of the generator.
type Word struct {
	Word string `json:"word"`
	Clue string `json:"clue"`
}

// ReadOptions control how a word list is read.
type ReadOptions struct {
	StripSpaces bool // remove spaces inside words instead of refusing them
}

// ReadWords parses a word list. JSON ([{word, clue}]) is detected by the
// ".json" extension or a leading '['; everything else is read as lines
// "WORT;Hinweis" (UTF-8, '#' comments and empty lines allowed). Words are
// normalized; invalid words are reported together in German.
func ReadWords(name string, raw []byte, o ReadOptions) ([]Word, error) {
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	var words []Word
	var errs []error
	isJSON := strings.EqualFold(filepath.Ext(name), ".json") || bytes.HasPrefix(bytes.TrimSpace(raw), []byte("["))
	if isJSON {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&words); err != nil {
			return nil, fmt.Errorf("%s: JSON erwartet [{\"word\": …, \"clue\": …}]: %w", name, err)
		}
	} else {
		sc := bufio.NewScanner(bytes.NewReader(raw))
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		line := 0
		for sc.Scan() {
			line++
			s := strings.TrimSpace(strings.TrimSuffix(sc.Text(), "\r"))
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			w, c, ok := strings.Cut(s, ";")
			if !ok {
				errs = append(errs, fmt.Errorf("Zeile %d: %q – Format ist WORT;Hinweis", line, s))
				continue
			}
			words = append(words, Word{Word: w, Clue: strings.TrimSpace(c)})
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	seen := map[string]int{}
	for i := range words {
		w, err := NormalizeInput(words[i].Word, o.StripSpaces)
		label := fmt.Sprintf("Wort %d (%q)", i+1, words[i].Word)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", label, err))
			continue
		}
		words[i].Word = w
		words[i].Clue = strings.TrimSpace(words[i].Clue)
		if words[i].Clue == "" {
			errs = append(errs, fmt.Errorf("%s: Hinweis fehlt", label))
		}
		if utf8.RuneCountInString(words[i].Clue) > MaxClue {
			errs = append(errs, fmt.Errorf("%s: Hinweis ist zu lang (max. %d Zeichen)", label, MaxClue))
		}
		if j, dup := seen[w]; dup {
			errs = append(errs, fmt.Errorf("%s: %s steht schon als Wort %d in der Liste", label, w, j))
		}
		seen[w] = i + 1
	}
	if len(words) == 0 && len(errs) == 0 {
		errs = append(errs, errors.New("keine Wörter gefunden"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return words, nil
}

// NormalizeInput normalizes a word for the grid and explains in German why a
// word cannot be used.
func NormalizeInput(s string, stripSpaces bool) (string, error) {
	w := Normalize(s)
	if stripSpaces {
		w = strings.Join(strings.Fields(w), "")
	}
	for _, r := range w {
		switch {
		case IsLetter(r):
		case unicode.IsSpace(r):
			return "", errors.New("enthält Leerzeichen – zusammenschreiben (oder -strip-spaces benutzen)")
		case unicode.IsDigit(r):
			return "", errors.New("enthält Ziffern – im Kreuzworträtsel nur Buchstaben (Zahl ausschreiben)")
		case r == '-' || r == '‐' || r == '–':
			return "", errors.New("enthält einen Bindestrich – bitte zusammenschreiben")
		default:
			return "", fmt.Errorf("enthält %q – erlaubt sind nur A–Z, Ä, Ö, Ü und ß", string(r))
		}
	}
	if n := utf8.RuneCountInString(w); n < MinAnswer {
		return "", fmt.Errorf("zu kurz (mindestens %d Buchstaben)", MinAnswer)
	}
	return w, nil
}

// Options control the generator.
type Options struct {
	MaxSize  int    // maximum width and height (default 13: fits 360 px with 24 px cells)
	Seed     uint64 // random seed; same seed + same words = same grid
	Attempts int    // randomized attempts, the best one wins (default 200)
}

// Result is the best layout found.
type Result struct {
	Raster   *Raster
	Unplaced []Word
	Seed     uint64
	Attempt  int
}

// DefaultMaxSize keeps the board at >= 24 px per cell on a 360 px phone.
const DefaultMaxSize = 13

// Generate lays out the words as a connected crossword. Longest words go
// first; every further word must cross existing letters without touching
// parallel words or extending other words. Each attempt shuffles the order
// of equally long words and the choice among good placements; attempts are
// scored by placed words, then compactness (area, then width – the phone is
// portrait), then crossings.
func Generate(words []Word, o Options) (*Result, error) {
	if len(words) == 0 {
		return nil, errors.New("keine Wörter")
	}
	if o.MaxSize == 0 {
		o.MaxSize = DefaultMaxSize
	}
	if o.MaxSize < 3 || o.MaxSize > MaxSize {
		return nil, fmt.Errorf("max-size muss zwischen 3 und %d liegen", MaxSize)
	}
	if o.Attempts <= 0 {
		o.Attempts = 200
	}
	var best *layout
	bestAttempt := 0
	for a := 0; a < o.Attempts; a++ {
		rng := rand.New(rand.NewPCG(o.Seed, uint64(a)))
		l := attempt(words, o.MaxSize, rng)
		if best == nil || l.better(best) {
			best, bestAttempt = l, a
		}
		if len(best.placed) == len(words) && a >= o.Attempts/4 && best.area() <= minArea(words) {
			break
		}
	}
	if len(best.placed) == 0 {
		longest, name := 0, ""
		for _, w := range words {
			if n := utf8.RuneCountInString(w.Word); n > longest {
				longest, name = n, w.Word
			}
		}
		if longest > o.MaxSize {
			return nil, fmt.Errorf("kein Wort passt in max-size %d – das längste Wort (%s) hat %d Buchstaben, -max-size erhöhen (höchstens %d) oder kürzere Wörter wählen", o.MaxSize, name, longest, MaxSize)
		}
		return nil, fmt.Errorf("kein Wort ließ sich in max-size %d platzieren – -max-size erhöhen oder andere Wörter wählen", o.MaxSize)
	}
	r := best.raster()
	var unplaced []Word
	for i, w := range words {
		if !best.isPlaced(i) {
			unplaced = append(unplaced, w)
		}
	}
	if err := r.Validate(); err != nil {
		return nil, fmt.Errorf("interner Fehler, erzeugtes Raster ungültig: %w", err)
	}
	return &Result{Raster: r, Unplaced: unplaced, Seed: o.Seed, Attempt: bestAttempt}, nil
}

// minArea is a lower bound for an early exit (longest word × 1 is not a grid).
func minArea(words []Word) int {
	longest := 0
	for _, w := range words {
		longest = max(longest, utf8.RuneCountInString(w.Word))
	}
	return longest * longest / 2
}

type cell struct {
	ch     rune
	across bool
	down   bool
}

type placement struct {
	idx      int // index into words
	row, col int
	down     bool
}

type layout struct {
	words  []Word
	runes  [][]rune
	grid   map[[2]int]*cell
	placed []placement
	minR   int
	maxR   int
	minC   int
	maxC   int
	cross  int
	limit  int
}

func (l *layout) area() int {
	if len(l.placed) == 0 {
		return 0
	}
	return (l.maxR - l.minR + 1) * (l.maxC - l.minC + 1)
}

func (l *layout) width() int { return l.maxC - l.minC + 1 }

func (l *layout) better(o *layout) bool {
	if len(l.placed) != len(o.placed) {
		return len(l.placed) > len(o.placed)
	}
	if l.area() != o.area() {
		return l.area() < o.area()
	}
	if l.width() != o.width() {
		return l.width() < o.width()
	}
	return l.cross > o.cross
}

func (l *layout) isPlaced(i int) bool {
	for _, p := range l.placed {
		if p.idx == i {
			return true
		}
	}
	return false
}

func (l *layout) at(r, c int) *cell { return l.grid[[2]int{r, c}] }

func attempt(words []Word, limit int, rng *rand.Rand) *layout {
	l := &layout{words: words, grid: map[[2]int]*cell{}, limit: limit}
	for _, w := range words {
		l.runes = append(l.runes, []rune(w.Word))
	}
	order := make([]int, len(words))
	for i := range order {
		order[i] = i
	}
	rng.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	// longest first, with some jitter so medium words get a chance to anchor
	jitter := make([]float64, len(words))
	for i := range jitter {
		jitter[i] = float64(len(l.runes[i])) + rng.Float64()*2.5
	}
	sort.SliceStable(order, func(i, j int) bool { return jitter[order[i]] > jitter[order[j]] })
	longest := 0
	for k, idx := range order {
		if len(l.runes[idx]) > len(l.runes[order[longest]]) {
			longest = k
		}
	}
	order[0], order[longest] = order[longest], order[0]
	st := strategy{compactFirst: rng.IntN(2) == 0, constrained: rng.IntN(2) == 0, diverse: rng.IntN(3) == 0, topK: 1 + rng.IntN(4)}

	first := order[0]
	if len(l.runes[first]) > limit {
		// cannot place anything sensible; try the next word that fits
		for k, idx := range order {
			if len(l.runes[idx]) <= limit {
				order[0], order[k] = order[k], order[0]
				first = idx
				break
			}
		}
	}
	if len(l.runes[first]) <= limit {
		l.put(placement{idx: first, row: 0, col: 0, down: rng.IntN(4) == 0})
	}
	if len(l.placed) == 0 {
		return l
	}
	return beamSearch(l, order[1:], st, rng)
}

// beamWidth is the number of partial layouts kept per step.
const beamWidth = 6

type beamState struct {
	l       *layout
	pending []int
}

// beamSearch repeatedly places one of the remaining words – depending on
// the strategy the first placeable one (jittered longest-first order) or the
// most constrained one (fewest possible positions) – and keeps the best few
// partial layouts per step. Words without any position are skipped.
func beamSearch(start *layout, pending []int, st strategy, rng *rand.Rand) *layout {
	best := start
	states := []beamState{{start, pending}}
	for len(states) > 0 {
		var next []beamState
		for _, s := range states {
			bestK, bestN := -1, 0
			var bestC []candidate
			for k, idx := range s.pending {
				cands := s.l.candidates(idx)
				if len(cands) == 0 {
					continue
				}
				if bestK < 0 || (st.constrained && len(cands) < bestN) {
					bestK, bestN, bestC = k, len(cands), cands
				}
			}
			if bestK < 0 {
				continue
			}
			rest := make([]int, 0, len(s.pending)-1)
			rest = append(rest, s.pending[:bestK]...)
			rest = append(rest, s.pending[bestK+1:]...)
			rankCandidates(bestC, st)
			rng.Shuffle(min(len(bestC), st.topK), func(i, j int) { bestC[i], bestC[j] = bestC[j], bestC[i] })
			for _, c := range bestC[:min(len(bestC), beamWidth)] {
				child := s.l.clone()
				child.put(c.p)
				next = append(next, beamState{child, rest})
			}
		}
		if len(next) == 0 {
			break
		}
		rng.Shuffle(len(next), func(i, j int) { next[i], next[j] = next[j], next[i] })
		if !st.diverse {
			sort.SliceStable(next, func(i, j int) bool { return next[i].l.better(next[j].l) })
		}
		if len(next) > beamWidth {
			next = next[:beamWidth]
		}
		states = next
		for _, n := range next {
			if n.l.better(best) {
				best = n.l
			}
		}
	}
	return best
}

func (l *layout) clone() *layout {
	c := *l
	c.grid = make(map[[2]int]*cell, len(l.grid))
	for k, v := range l.grid {
		cv := *v
		c.grid[k] = &cv
	}
	c.placed = append([]placement(nil), l.placed...)
	return &c
}

type candidate struct {
	p     placement
	cross int
	area  int
	width int
}

// strategy varies per attempt: which candidate property wins and how much
// randomness is allowed among the best candidates.
type strategy struct {
	compactFirst bool
	constrained  bool
	diverse      bool // keep random partial layouts instead of the most compact ones
	topK         int
}

// rankCandidates sorts candidates best first: most crossings or smallest
// resulting area first, depending on the strategy.
func rankCandidates(cs []candidate, st strategy) {
	sort.SliceStable(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		if st.compactFirst && a.area != b.area {
			return a.area < b.area
		}
		if a.cross != b.cross {
			return a.cross > b.cross
		}
		if a.area != b.area {
			return a.area < b.area
		}
		return a.width < b.width
	})
}

func (l *layout) candidates(idx int) []candidate {
	w := l.runes[idx]
	var out []candidate
	seen := map[placement]bool{}
	for pos, c := range l.grid {
		for k, ch := range w {
			if ch != c.ch {
				continue
			}
			// cross perpendicular to the existing word at this cell
			for _, down := range []bool{false, true} {
				if (down && c.down) || (!down && c.across) {
					continue
				}
				p := placement{idx: idx, down: down}
				if down {
					p.row, p.col = pos[0]-k, pos[1]
				} else {
					p.row, p.col = pos[0], pos[1]-k
				}
				if seen[p] {
					continue
				}
				seen[p] = true
				if cross, ok := l.fits(p); ok {
					minR, maxR, minC, maxC := l.boundsWith(p)
					out = append(out, candidate{p: p, cross: cross,
						area: (maxR - minR + 1) * (maxC - minC + 1), width: maxC - minC + 1})
				}
			}
		}
	}
	// deterministic order before random pick (map iteration is random)
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].p, out[j].p
		if a.row != b.row {
			return a.row < b.row
		}
		if a.col != b.col {
			return a.col < b.col
		}
		return !a.down && b.down
	})
	return out
}

func (l *layout) boundsWith(p placement) (minR, maxR, minC, maxC int) {
	n := len(l.runes[p.idx])
	endR, endC := p.row, p.col+n-1
	if p.down {
		endR, endC = p.row+n-1, p.col
	}
	if len(l.placed) == 0 {
		return p.row, endR, p.col, endC
	}
	return min(l.minR, p.row), max(l.maxR, endR), min(l.minC, p.col), max(l.maxC, endC)
}

// fits checks the crossword rules for a placement and returns the number of
// crossings. Rules: letters must match where cells are used; the cells right
// before and after the word are empty; new letters have no neighbours across
// the word direction (no parallel touching); a used cell must not already
// carry a word in the same direction; at least one crossing; bounds.
func (l *layout) fits(p placement) (int, bool) {
	w := l.runes[p.idx]
	dr, dc := 0, 1
	if p.down {
		dr, dc = 1, 0
	}
	if l.at(p.row-dr, p.col-dc) != nil || l.at(p.row+dr*len(w), p.col+dc*len(w)) != nil {
		return 0, false
	}
	cross := 0
	for k, ch := range w {
		r, c := p.row+dr*k, p.col+dc*k
		if cur := l.at(r, c); cur != nil {
			if cur.ch != ch || (p.down && cur.down) || (!p.down && cur.across) {
				return 0, false
			}
			cross++
			continue
		}
		// perpendicular neighbours of a new letter must be empty
		if l.at(r+dc, c+dr) != nil || l.at(r-dc, c-dr) != nil {
			return 0, false
		}
	}
	if cross == 0 || cross == len(w) {
		return 0, false
	}
	minR, maxR, minC, maxC := l.boundsWith(p)
	if maxR-minR+1 > l.limit || maxC-minC+1 > l.limit {
		return 0, false
	}
	return cross, true
}

func (l *layout) put(p placement) {
	w := l.runes[p.idx]
	dr, dc := 0, 1
	if p.down {
		dr, dc = 1, 0
	}
	l.minR, l.maxR, l.minC, l.maxC = l.boundsWith(p)
	for k, ch := range w {
		key := [2]int{p.row + dr*k, p.col + dc*k}
		c := l.grid[key]
		if c == nil {
			c = &cell{ch: ch}
			l.grid[key] = c
		} else {
			l.cross++
		}
		if p.down {
			c.down = true
		} else {
			c.across = true
		}
	}
	l.placed = append(l.placed, p)
}

// raster converts the layout to Raster-JSON with standard numbering.
func (l *layout) raster() *Raster {
	r := &Raster{Version: Version}
	if len(l.placed) == 0 {
		return r
	}
	r.Height, r.Width = l.maxR-l.minR+1, l.maxC-l.minC+1
	r.Cells = make([][]*string, r.Height)
	for i := range r.Cells {
		r.Cells[i] = make([]*string, r.Width)
	}
	for pos, c := range l.grid {
		s := string(c.ch)
		r.Cells[pos[0]-l.minR][pos[1]-l.minC] = &s
	}
	clues := map[string]string{}
	for _, p := range l.placed {
		dir := Across
		if p.down {
			dir = Down
		}
		clues[fmt.Sprintf("%s/%d/%d", dir, p.row-l.minR, p.col-l.minC)] = l.words[p.idx].Clue
	}
	for _, run := range r.Runs() {
		r.Entries = append(r.Entries, Entry{Number: run.Number, Direction: run.Direction, Row: run.Row, Col: run.Col,
			Answer: run.Answer, Clue: clues[fmt.Sprintf("%s/%d/%d", run.Direction, run.Row, run.Col)]})
	}
	return r
}

// Marshal renders a raster as indented JSON with one grid row per line.
func Marshal(r *Raster) ([]byte, error) {
	var b bytes.Buffer
	fmt.Fprintf(&b, "{\n  \"version\": %d,\n  \"width\": %d,\n  \"height\": %d,\n  \"cells\": [\n", r.Version, r.Width, r.Height)
	for i, row := range r.Cells {
		line, err := marshalLine(row)
		if err != nil {
			return nil, err
		}
		b.WriteString("    ")
		b.Write(line)
		if i < len(r.Cells)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ],\n  \"entries\": [\n")
	for i, e := range r.Entries {
		line, err := marshalLine(e)
		if err != nil {
			return nil, err
		}
		b.WriteString("    ")
		b.Write(line)
		if i < len(r.Entries)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("  ]\n}\n")
	return b.Bytes(), nil
}

// marshalLine encodes v on one line without HTML escaping ("&" stays "&").
func marshalLine(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}
