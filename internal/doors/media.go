package doors

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jakobneri/sandyadvent/internal/advent"
)

// Block types of the media template.
const (
	BlockText      = "text"
	BlockAudio     = "audio"
	BlockVideo     = "video"
	BlockSlideshow = "slideshow"
	BlockLink      = "link"
	BlockImage     = "image"
	BlockYearsAgo  = "yearsago"
	BlockTeaser    = "teaser"
)

// MediaParams are the params of template "media".
type MediaParams struct {
	Blocks []Block `json:"blocks"`
}

// Slide is one image of a slideshow.
type Slide struct {
	Src     string `json:"src"`
	Caption string `json:"caption,omitempty"`
	Alt     string `json:"alt,omitempty"`
}

// Block is one freely combinable block. Which fields apply depends on Type.
type Block struct {
	Type       string  `json:"type"`
	Heading    string  `json:"heading,omitempty"`
	Text       string  `json:"text,omitempty"`
	Src        string  `json:"src,omitempty"`
	Caption    string  `json:"caption,omitempty"`
	Alt        string  `json:"alt,omitempty"`
	Poster     string  `json:"poster,omitempty"`
	Images     []Slide `json:"images,omitempty"`
	IntervalMS int     `json:"interval_ms,omitempty"`
	URL        string  `json:"url,omitempty"`
	Label      string  `json:"label,omitempty"`
	Note       string  `json:"note,omitempty"`
	Date       string  `json:"date,omitempty"`
	Image      string  `json:"image,omitempty"`

	YearsAgoDate advent.Date `json:"-"`
}

// ParseMedia decodes and validates media params.
func ParseMedia(raw json.RawMessage) (*MediaParams, error) {
	m := &MediaParams{}
	if len(raw) == 0 {
		return nil, errors.New("params.blocks fehlt")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(m); err != nil {
		return nil, fmt.Errorf("params: %w", err)
	}
	if len(m.Blocks) == 0 {
		return nil, errors.New("params.blocks ist leer")
	}
	var errs []error
	for i := range m.Blocks {
		if err := m.Blocks[i].validate(); err != nil {
			errs = append(errs, fmt.Errorf("Block %d (%s): %w", i+1, m.Blocks[i].Type, err))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return m, nil
}

func (b *Block) validate() error {
	var errs []error
	need := func(name, v string) {
		if strings.TrimSpace(v) == "" {
			errs = append(errs, fmt.Errorf("%s fehlt", name))
		}
	}
	asset := func(name, v string) {
		if v == "" {
			return
		}
		if _, err := CleanAssetPath(v); err != nil {
			errs = append(errs, fmt.Errorf("%s %q: %w (relativer Pfad im assets_dir erwartet)", name, v, err))
		}
	}
	switch b.Type {
	case BlockText:
		need("text", b.Text)
	case BlockAudio:
		need("src", b.Src)
		asset("src", b.Src)
	case BlockVideo:
		need("src", b.Src)
		asset("src", b.Src)
		asset("poster", b.Poster)
	case BlockImage:
		need("src", b.Src)
		asset("src", b.Src)
	case BlockSlideshow:
		if len(b.Images) == 0 {
			errs = append(errs, errors.New("images ist leer"))
		}
		for i, s := range b.Images {
			if s.Src == "" {
				errs = append(errs, fmt.Errorf("images[%d].src fehlt", i))
			}
			asset(fmt.Sprintf("images[%d].src", i), s.Src)
		}
		if b.IntervalMS != 0 && b.IntervalMS < 1500 {
			errs = append(errs, errors.New("interval_ms muss 0 (aus) oder ≥ 1500 sein"))
		}
	case BlockLink:
		need("label", b.Label)
		if !IsHTTPURL(b.URL) {
			errs = append(errs, fmt.Errorf("url %q muss mit http:// oder https:// beginnen", b.URL))
		}
	case BlockYearsAgo:
		d, err := advent.ParseDate(b.Date)
		if err != nil {
			errs = append(errs, fmt.Errorf("date: %w", err))
		}
		b.YearsAgoDate = d
		asset("image", b.Image)
	case BlockTeaser:
		// optional text, default "Morgen."
	default:
		errs = append(errs, fmt.Errorf("unbekannter Block-Typ %q", b.Type))
	}
	return errors.Join(errs...)
}

// AssetRefs returns all asset paths referenced by the blocks.
func (m *MediaParams) AssetRefs() []string {
	var refs []string
	add := func(s string) {
		if s != "" {
			refs = append(refs, s)
		}
	}
	for _, b := range m.Blocks {
		add(b.Src)
		add(b.Poster)
		add(b.Image)
		for _, s := range b.Images {
			add(s.Src)
		}
	}
	return refs
}
