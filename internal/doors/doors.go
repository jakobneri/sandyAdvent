// Package doors loads and validates the door registry (doors/config.json).
package doors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
)

// Templates known to the calendar. Templates that are not implemented yet
// render a placeholder until their milestone lands.
var Templates = map[string]bool{
	"media": true, "quiz": true, "memory": true, "wordle": true, "puzzle": true,
	"crossword": true, "wrapped": true, "reveal": true,
}

// Implemented reports whether a template has a real implementation yet.
func Implemented(template string) bool {
	switch template {
	case "media", "quiz", "memory", "wordle", "puzzle", "crossword", "wrapped":
		return true
	}
	return false
}

// IsGame reports whether a template is rendered by the game shell: the
// interactive games (M2), the crossword and the wrapped story (M3).
func IsGame(template string) bool {
	switch template {
	case "quiz", "memory", "wordle", "puzzle", "crossword", "wrapped":
		return true
	}
	return false
}

// Door is one entry of doors/config.json.
type Door struct {
	Day      int             `json:"-"`
	Comment  string          `json:"_comment,omitempty"`
	Template string          `json:"template"`
	Title    string          `json:"title"`
	Params   json.RawMessage `json:"params,omitempty"`
	Assets   []string        `json:"assets,omitempty"`

	// Parsed params; exactly one is set for implemented templates.
	Media  *MediaParams  `json:"-"`
	Quiz   *QuizParams   `json:"-"`
	Memory *MemoryParams `json:"-"`
	Wordle *WordleParams `json:"-"`
	Puzzle *PuzzleParams `json:"-"`

	Crossword *CrosswordParams `json:"-"`
	Wrapped   *WrappedParams   `json:"-"`
}

// File is the on-disk structure.
type File struct {
	Comment string           `json:"_comment,omitempty"`
	Doors   map[string]*Door `json:"doors"`
}

// Registry maps day -> door.
type Registry struct {
	Doors map[int]*Door
}

// Get returns the door for day or nil.
func (r *Registry) Get(day int) *Door {
	if r == nil {
		return nil
	}
	return r.Doors[day]
}

// Load reads and validates the registry.
func Load(file string) (*Registry, error) {
	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Türchen-Config lesen: %w", err)
	}
	reg, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	return reg, nil
}

// Parse decodes and validates a registry.
func Parse(raw []byte) (*Registry, error) {
	var f File
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("ungültiges JSON: %w", err)
	}
	reg := &Registry{Doors: map[int]*Door{}}
	var errs []error
	keys := make([]string, 0, len(f.Doors))
	for k := range f.Doors {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		d := f.Doors[k]
		day, err := strconv.Atoi(k)
		switch {
		case err != nil || day < 1 || day > 24:
			errs = append(errs, fmt.Errorf("Tag %q: muss 1–24 sein", k))
			continue
		case advent.IsVoucherDay(day):
			errs = append(errs, fmt.Errorf("Tag %d ist ein Gutschein-Tag und hat kein Türchen", day))
			continue
		case d == nil:
			errs = append(errs, fmt.Errorf("Tag %d: leerer Eintrag", day))
			continue
		}
		if _, dup := reg.Doors[day]; dup {
			errs = append(errs, fmt.Errorf("Tag %d doppelt", day))
			continue
		}
		d.Day = day
		if err := d.validate(); err != nil {
			errs = append(errs, fmt.Errorf("Tag %d: %w", day, err))
			continue
		}
		reg.Doors[day] = d
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return reg, nil
}

// CheckDates validates the date-dependent blocks against the calendar year:
// a "yearsago" date must lie before the door's own date (December <day>).
func (r *Registry) CheckDates(year int) error {
	if r == nil {
		return nil
	}
	days := make([]int, 0, len(r.Doors))
	for day := range r.Doors {
		days = append(days, day)
	}
	sort.Ints(days)
	var errs []error
	for _, day := range days {
		d := r.Doors[day]
		if d.Media == nil {
			continue
		}
		on := advent.Date{Year: year, Month: time.December, Day: day}
		for i, b := range d.Media.Blocks {
			if b.Type == BlockYearsAgo && advent.DaysBetween(b.YearsAgoDate, on) <= 0 {
				errs = append(errs, fmt.Errorf("Tag %d: Block %d (yearsago): date %s muss vor dem Türchen-Datum %s liegen", day, i+1, b.YearsAgoDate, on))
			}
		}
	}
	return errors.Join(errs...)
}

func (d *Door) validate() error {
	var errs []error
	if !Templates[d.Template] {
		errs = append(errs, fmt.Errorf("unbekanntes Template %q", d.Template))
	}
	if strings.TrimSpace(d.Title) == "" {
		errs = append(errs, errors.New("title fehlt"))
	}
	for _, a := range d.Assets {
		if _, err := CleanAssetPath(strings.TrimSuffix(a, "/")); err != nil {
			errs = append(errs, fmt.Errorf("assets %q: %w", a, err))
		}
	}
	if len(d.Params) > 0 && !bytes.Equal(bytes.TrimSpace(d.Params), []byte("null")) {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(d.Params, &obj); err != nil {
			errs = append(errs, fmt.Errorf("params muss ein Objekt sein: %w", err))
		}
	}
	if len(errs) == 0 && Implemented(d.Template) {
		if err := d.parseParams(); err != nil {
			errs = append(errs, err)
		} else {
			for _, src := range d.AssetRefs() {
				if !d.AllowsAsset(src) {
					errs = append(errs, fmt.Errorf("Datei %q wird benutzt, steht aber nicht in assets", src))
				}
			}
		}
	}
	return errors.Join(errs...)
}

// parseParams parses the params of an implemented template.
func (d *Door) parseParams() error {
	var err error
	switch d.Template {
	case "media":
		d.Media, err = ParseMedia(d.Params)
	case "quiz":
		d.Quiz, err = ParseQuiz(d.Params)
	case "memory":
		d.Memory, err = ParseMemory(d.Params)
	case "wordle":
		d.Wordle, err = ParseWordle(d.Params)
	case "puzzle":
		d.Puzzle, err = ParsePuzzle(d.Params)
	case "crossword":
		d.Crossword, err = ParseCrossword(d.Params)
	case "wrapped":
		d.Wrapped, err = ParseWrapped(d.Params)
	}
	return err
}

// AssetRefs returns all asset paths referenced by the parsed params.
func (d *Door) AssetRefs() []string {
	switch {
	case d.Media != nil:
		return d.Media.AssetRefs()
	case d.Quiz != nil:
		return d.Quiz.AssetRefs()
	case d.Memory != nil:
		return d.Memory.AssetRefs()
	case d.Puzzle != nil:
		return d.Puzzle.AssetRefs()
	case d.Crossword != nil:
		return d.Crossword.AssetRefs()
	case d.Wrapped != nil:
		return d.Wrapped.AssetRefs()
	}
	return nil
}

// CleanAssetPath validates a relative asset path (slash separated) and returns
// its clean form. Absolute paths, "..", backslashes and hidden segments are refused.
func CleanAssetPath(p string) (string, error) {
	if p == "" {
		return "", errors.New("leerer Pfad")
	}
	if strings.ContainsAny(p, "\\\x00") || strings.HasPrefix(p, "/") {
		return "", errors.New("ungültiger Pfad")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == "" || seg == "." || seg == ".." || strings.HasPrefix(seg, ".") {
			return "", errors.New("ungültiger Pfad")
		}
	}
	c := path.Clean(p)
	if c != p || !filepath.IsLocal(filepath.FromSlash(c)) {
		return "", errors.New("ungültiger Pfad")
	}
	return c, nil
}

// AllowsAsset reports whether p (already clean) belongs to this door: either
// listed exactly or below a listed directory ("fotos/").
func (d *Door) AllowsAsset(p string) bool {
	for _, a := range d.Assets {
		if a == p {
			return true
		}
		if strings.HasSuffix(a, "/") && strings.HasPrefix(p, a) {
			return true
		}
	}
	return false
}

// MissingAssets returns the door's files that do not exist below dir.
// Directory entries are checked for existence only.
func (d *Door) MissingAssets(dir string) []string {
	var missing []string
	check := map[string]bool{}
	for _, a := range d.Assets {
		check[a] = true
	}
	for _, a := range d.AssetRefs() {
		check[a] = true
	}
	keys := make([]string, 0, len(check))
	for k := range check {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, a := range keys {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(a, "/")))); err != nil {
			missing = append(missing, a)
		}
	}
	return missing
}

// IsHTTPURL reports whether s is an absolute http(s) URL.
func IsHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
