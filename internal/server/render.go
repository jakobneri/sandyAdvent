package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/doors"
	"github.com/jakobneri/sandyadvent/web"
)

// pageFiles lists, per page, the template files parsed on top of the layout.
var pageFiles = map[string][]string{
	"door":      {"templates/layout.html", "templates/media.html", "templates/games.html", "templates/door.html"},
	"countdown": {"templates/layout.html", "templates/countdown.html"},
	"preview":   {"templates/layout.html", "templates/preview.html"},
	"neutral":   {"templates/neutral.html"},
}

func (s *Server) parseTemplates() error {
	funcs := template.FuncMap{
		"static": s.static.URL,
		"logo":   func() template.HTML { return s.logo },
		"plural": advent.Plural,
	}
	s.pages = map[string]*template.Template{}
	for name, files := range pageFiles {
		t, err := template.New(name).Funcs(funcs).ParseFS(web.FS, files...)
		if err != nil {
			return fmt.Errorf("template %s: %w", name, err)
		}
		s.pages[name] = t
	}
	return nil
}

// --- view models -----------------------------------------------------------

type neutralView struct {
	Title, Heading, Text string
}

// layoutView is embedded by every christmas-layout page.
type layoutView struct {
	Title     string
	BodyClass string
	Day       int  // 0 = no door context
	Intro     bool // play the door-opening animation on first view
	Dev       bool
}

type doorView struct {
	layoutView
	DateLabel   string
	Door        *doors.Door
	Template    string
	Implemented bool
	Banner      string
	Blocks      []blockView
	Game        *gameView
}

type blockView struct {
	doors.Block
	ID         string
	URL        string
	PosterURL  string
	ImageURL   string
	Paragraphs [][]string
	Slides     []slideView
	YearsLabel string
	DateLabel  string
	TeaserText string
}

type slideView struct {
	URL, Caption, Alt string
}

type countdownView struct {
	layoutView
	DateLabel string
	Line      string
	Remaining remaining
	Client    countdownClient
}

// countdownClient is handed to countdown.js as JSON. Now is the server's
// (possibly ADVENT_NOW-shifted) clock so the client can compute its offset.
type countdownClient struct {
	Now    int64    `json:"now"`
	Target int64    `json:"target"`
	Lines  []string `json:"lines"`
	Reload bool     `json:"reload"`
}

type remaining struct{ Days, Hours, Minutes, Seconds int }

func splitRemaining(d time.Duration) remaining {
	if d < 0 {
		d = 0
	}
	secs := int(d / time.Second)
	return remaining{secs / 86400, secs % 86400 / 3600, secs % 3600 / 60, secs % 60}
}

func dateLabel(day int) string { return fmt.Sprintf("%d. Dezember", day) }

// paragraphs splits text into paragraphs (blank line) and lines (newline).
func paragraphs(text string) [][]string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var out [][]string
	for _, p := range strings.Split(text, "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, strings.Split(p, "\n"))
	}
	return out
}

func yearsLabel(then, on advent.Date) string {
	years := advent.YearsAgo(then, on)
	sameDay := then.Month == on.Month && then.Day == on.Day
	prefix := "Vor"
	if sameDay {
		prefix = "Heute vor"
	}
	switch {
	case years <= 0:
		// Dates on/after the door date are rejected by Registry.CheckDates;
		// this only keeps the label sensible if one slips through.
		switch days := advent.DaysBetween(then, on); {
		case days <= 0:
			return "Heute"
		case days == 1:
			return "Gestern"
		default:
			return fmt.Sprintf("Vor %d Tagen", days)
		}
	case years == 1:
		return prefix + " einem Jahr"
	default:
		return fmt.Sprintf("%s %d Jahren", prefix, years)
	}
}

func (s *Server) buildDoorView(door *doors.Door, assetBase string) doorView {
	now := s.clock.Now()
	v := doorView{
		layoutView: layoutView{
			Title:     door.Title,
			BodyClass: "page-door tpl-" + door.Template,
			Day:       door.Day,
			Intro:     true,
			Dev:       s.cfg.Dev,
		},
		DateLabel:   dateLabel(door.Day),
		Door:        door,
		Template:    door.Template,
		Implemented: doors.Implemented(door.Template),
	}
	// Not on 23 (teaser) and 24 (reveal), see CLAUDE.md.
	if advent.ShowsBanner(door.Day) {
		v.Banner = advent.BannerText(now, s.cfg.Reveal)
	}
	asset := func(p string) string {
		if p == "" {
			return ""
		}
		return assetBase + p
	}
	if doors.IsGame(door.Template) {
		v.Game = buildGame(door, gameEnv{asset: asset, assetsDir: s.cfg.AssetsDir, now: now, year: s.cfg.Year})
		if v.Game != nil && v.Game.err != nil {
			s.log.Error("Türchen-Daten", "tag", door.Day, "err", v.Game.err)
		}
	}
	if door.Media == nil {
		return v
	}
	doorDate := advent.Date{Year: s.cfg.Year, Month: time.December, Day: door.Day}
	for i, b := range door.Media.Blocks {
		bv := blockView{Block: b, ID: fmt.Sprintf("b%d", i+1)}
		bv.URL = asset(b.Src)
		bv.PosterURL = asset(b.Poster)
		bv.ImageURL = asset(b.Image)
		switch b.Type {
		case doors.BlockText:
			bv.Paragraphs = paragraphs(b.Text)
		case doors.BlockYearsAgo:
			bv.YearsLabel = yearsLabel(b.YearsAgoDate, doorDate)
			bv.DateLabel = b.YearsAgoDate.FormatDE()
			bv.Paragraphs = paragraphs(b.Text)
		case doors.BlockSlideshow:
			for j, sl := range b.Images {
				alt := sl.Alt
				if alt == "" {
					alt = sl.Caption
				}
				if alt == "" {
					alt = fmt.Sprintf("Foto %d", j+1)
				}
				bv.Slides = append(bv.Slides, slideView{URL: asset(sl.Src), Caption: sl.Caption, Alt: alt})
			}
		case doors.BlockTeaser:
			bv.TeaserText = b.Text
			if bv.TeaserText == "" {
				bv.TeaserText = "Morgen."
			}
		}
		v.Blocks = append(v.Blocks, bv)
	}
	return v
}

func (s *Server) renderDoor(w http.ResponseWriter, _ *http.Request, door *doors.Door, assetBase string) {
	s.render(w, http.StatusOK, "door", s.buildDoorView(door, assetBase))
}

func (s *Server) renderCountdown(w http.ResponseWriter, _ *http.Request, day int, reload bool) {
	now := s.clock.Now()
	target := advent.UnlockTime(s.cfg.Year, day)
	v := countdownView{
		layoutView: layoutView{Title: fmt.Sprintf("Türchen %d", day), BodyClass: "page-countdown", Day: day, Dev: s.cfg.Dev},
		DateLabel:  dateLabel(day),
		Line:       s.lines[s.pick(len(s.lines))],
		Remaining:  splitRemaining(target.Sub(now)),
		Client: countdownClient{
			Now:    now.UnixMilli(),
			Target: target.UnixMilli(),
			Lines:  s.lines,
			Reload: reload,
		},
	}
	s.render(w, http.StatusOK, "countdown", v)
}

// --- dev preview -----------------------------------------------------------

type previewDay struct {
	Day         int
	Voucher     bool
	QR          bool
	Unlocked    bool
	UnlockAt    string
	Door        *doors.Door
	Token       string
	Missing     []string
	Implemented bool
}

type previewView struct {
	layoutView
	Now        string
	Overridden bool
	Banner     string
	ConfigPath string
	AssetsDir  string
	BaseURL    string
	MissingTok []int
	Days       []previewDay
}

func (s *Server) handlePreview(w http.ResponseWriter, _ *http.Request) {
	now := s.clock.Now()
	v := previewView{
		layoutView: layoutView{Title: "Vorschau", BodyClass: "page-preview", Dev: true},
		Now:        now.In(advent.Berlin).Format("02.01.2006 15:04:05 MST"),
		Overridden: !s.cfg.Now.IsZero(),
		Banner:     advent.BannerText(now, s.cfg.Reveal),
		ConfigPath: s.cfg.Path,
		AssetsDir:  s.cfg.AssetsDir,
		BaseURL:    s.cfg.BaseURL,
		MissingTok: s.cfg.MissingTokenDays(),
	}
	for day := 1; day <= 24; day++ {
		pd := previewDay{
			Day:      day,
			Voucher:  advent.IsVoucherDay(day),
			QR:       advent.IsQRDay(day),
			Unlocked: advent.IsUnlocked(now, s.cfg.Year, day),
			UnlockAt: advent.UnlockTime(s.cfg.Year, day).Format("02.01. 15:04"),
		}
		if pd.QR {
			pd.Door = s.doors.Get(day)
			pd.Token = s.cfg.TokenForDay(day)
			if pd.Door != nil {
				pd.Missing = pd.Door.MissingAssets(s.cfg.AssetsDir)
				pd.Implemented = doors.Implemented(pd.Door.Template)
			}
		}
		v.Days = append(v.Days, pd)
	}
	s.render(w, http.StatusOK, "preview", v)
}
