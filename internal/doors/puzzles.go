package doors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jakobneri/sandyadvent/internal/crossword"
	"github.com/jakobneri/sandyadvent/internal/waparse"
)

// Params of the M3 templates: crossword (door 16) and wrapped (door 22).
// Both reference a data file in assets_dir (Raster-JSON resp. stats.json)
// that is validated at start/check and read again when the door is shown,
// so Jakob can replace it on the Pi without a restart.

// MaxDataFile limits data files read into memory.
const MaxDataFile = 2 << 20

// ReadAssetFile reads a data file below assetsDir (no escaping via ".." or
// symlinks, size limited).
func ReadAssetFile(assetsDir, rel string) ([]byte, error) {
	clean, err := CleanAssetPath(rel)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenInRoot(assetsDir, filepath.FromSlash(clean))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, MaxDataFile+1))
	if err != nil {
		return nil, err
	}
	if len(b) > MaxDataFile {
		return nil, fmt.Errorf("%s ist größer als %d MB", rel, MaxDataFile>>20)
	}
	return b, nil
}

// ---------------------------------------------------------------------------
// crossword

// CrosswordParams are the params of template "crossword".
type CrosswordParams struct {
	Intro             string          `json:"intro,omitempty"`
	Grid              json.RawMessage `json:"grid"`
	ClosingText       string          `json:"closing_text,omitempty"`
	AllowCheck        *bool           `json:"allow_check,omitempty"`
	AllowRevealLetter bool            `json:"allow_reveal_letter,omitempty"`

	GridPath string            `json:"-"` // grid as asset path …
	Inline   *crossword.Raster `json:"-"` // … or inline Raster-JSON
}

// ParseCrossword decodes and validates crossword params. An inline grid is
// validated completely; a grid file is validated by CheckData / Raster.
func ParseCrossword(raw json.RawMessage) (*CrosswordParams, error) {
	p := &CrosswordParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	g := bytes.TrimSpace(p.Grid)
	switch {
	case len(g) == 0 || bytes.Equal(g, []byte("null")):
		c.add("grid fehlt (Pfad zur Raster-JSON-Datei, erzeugt mit tools/crossword, oder das Raster direkt als Objekt)")
	case g[0] == '"':
		if err := json.Unmarshal(g, &p.GridPath); err != nil {
			c.add("grid: %v", err)
		}
		c.need("grid", p.GridPath)
		c.asset("grid", p.GridPath)
	case g[0] == '{':
		r, err := crossword.Parse(g)
		if err != nil {
			c.add("grid (inline): %v", err)
		}
		p.Inline = r
	default:
		c.add("grid muss ein Pfad (Text) oder ein Raster-Objekt sein")
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// Check reports whether the "Prüfen" button is shown (default true).
func (p *CrosswordParams) Check() bool { return p.AllowCheck == nil || *p.AllowCheck }

// AssetRefs returns the grid file, if any.
func (p *CrosswordParams) AssetRefs() []string {
	if p.GridPath != "" {
		return []string{p.GridPath}
	}
	return nil
}

// Raster returns the grid (inline or read and validated from assetsDir) and
// the raw bytes it came from (for the progress key).
func (p *CrosswordParams) Raster(assetsDir string) (*crossword.Raster, []byte, error) {
	if p.Inline != nil {
		return p.Inline, nil, nil
	}
	raw, err := ReadAssetFile(assetsDir, p.GridPath)
	if err != nil {
		return nil, nil, fmt.Errorf("grid %s: %w", p.GridPath, err)
	}
	r, err := crossword.Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("grid %s: %w", p.GridPath, err)
	}
	return r, raw, nil
}

// ---------------------------------------------------------------------------
// wrapped

// Wrapped slide ids in their default order. "response" (average response
// time per person) is optional and only shown when listed in "slides".
var (
	WrappedSlideIDs     = []string{"intro", "days", "messages", "emojis", "goodnight", "hours", "streak", "busiest", "words", "first", "response", "outro"}
	WrappedDefaultOrder = []string{"intro", "days", "messages", "emojis", "goodnight", "hours", "streak", "busiest", "words", "first", "outro"}
)

// WrappedParams are the params of template "wrapped".
type WrappedParams struct {
	Intro     string         `json:"intro,omitempty"`
	Stats     string         `json:"stats"`
	Title     string         `json:"title,omitempty"`
	Slides    []WrappedSlide `json:"slides,omitempty"`
	OutroText string         `json:"outro_text,omitempty"`
}

// WrappedSlide selects/orders a slide and optionally overrides its texts.
type WrappedSlide struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	Text  string `json:"text,omitempty"`
}

// ParseWrapped decodes and validates wrapped params.
func ParseWrapped(raw json.RawMessage) (*WrappedParams, error) {
	p := &WrappedParams{}
	if err := decodeParams(raw, p); err != nil {
		return nil, err
	}
	var c checker
	c.need("stats", p.Stats)
	c.asset("stats", p.Stats)
	c.maxLen("title", p.Title, 60)
	c.maxLen("outro_text", p.OutroText, 600)
	known := map[string]bool{}
	for _, id := range WrappedSlideIDs {
		known[id] = true
	}
	seen := map[string]bool{}
	for i, s := range p.Slides {
		switch {
		case !known[s.ID]:
			c.add("slides[%d]: id %q unbekannt – erlaubt: %s", i, s.ID, strings.Join(WrappedSlideIDs, ", "))
		case seen[s.ID]:
			c.add("slides[%d]: id %q doppelt", i, s.ID)
		case s.ID == "intro" && i != 0:
			c.add("slides[%d]: intro muss die erste Slide sein", i)
		case s.ID == "outro" && i != len(p.Slides)-1:
			c.add("slides[%d]: outro muss die letzte Slide sein", i)
		}
		seen[s.ID] = true
		c.maxLen(fmt.Sprintf("slides[%d].title", i), s.Title, 80)
		c.maxLen(fmt.Sprintf("slides[%d].text", i), s.Text, 300)
	}
	if err := c.err(); err != nil {
		return nil, err
	}
	return p, nil
}

// SlideList returns the slides to show: the listed ones (intro and outro are
// always added) or the default order.
func (p *WrappedParams) SlideList() []WrappedSlide {
	if len(p.Slides) == 0 {
		out := make([]WrappedSlide, len(WrappedDefaultOrder))
		for i, id := range WrappedDefaultOrder {
			out[i] = WrappedSlide{ID: id}
		}
		return out
	}
	out := append([]WrappedSlide(nil), p.Slides...)
	if out[0].ID != "intro" {
		out = append([]WrappedSlide{{ID: "intro"}}, out...)
	}
	if out[len(out)-1].ID != "outro" {
		out = append(out, WrappedSlide{ID: "outro"})
	}
	return out
}

// AssetRefs returns the stats file.
func (p *WrappedParams) AssetRefs() []string { return []string{p.Stats} }

// LoadStats reads and validates the stats file.
func (p *WrappedParams) LoadStats(assetsDir string) (*waparse.Stats, error) {
	raw, err := ReadAssetFile(assetsDir, p.Stats)
	if err != nil {
		return nil, fmt.Errorf("stats %s: %w", p.Stats, err)
	}
	st, err := waparse.ParseStats(raw)
	if err != nil {
		return nil, fmt.Errorf("stats %s: %w", p.Stats, err)
	}
	return st, nil
}

// CheckData validates the data files of crossword and wrapped doors that
// exist below assetsDir. Missing files are only reported by MissingAssets
// (a warning) – the door then shows a friendly error until the file is there.
func (r *Registry) CheckData(assetsDir string) error {
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
		var err error
		switch {
		case d.Crossword != nil && d.Crossword.GridPath != "":
			if exists(assetsDir, d.Crossword.GridPath) {
				_, _, err = d.Crossword.Raster(assetsDir)
			}
		case d.Wrapped != nil:
			if exists(assetsDir, d.Wrapped.Stats) {
				_, err = d.Wrapped.LoadStats(assetsDir)
			}
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("Tag %d: %w", day, err))
		}
	}
	return errors.Join(errs...)
}

func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
	return err == nil
}
