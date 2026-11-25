package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/crossword"
	"github.com/jakobneri/sandyadvent/internal/doors"
	"github.com/jakobneri/sandyadvent/internal/waparse"
)

// gameView is the server-rendered shell of a game door. The game itself is
// built by web/static/js/<kind>.js from Data (JSON in the page).
type gameView struct {
	Kind    string
	Key     string     // localStorage key: day + hash of the params (changes reset progress)
	Intro   [][]string // paragraphs
	Data    any
	Scripts []string // static paths, loaded after js/game.js
	Styles  []string // extra static CSS, loaded after css/games.css
	Error   string   // data file missing/invalid: friendly message instead of the game
	err     error    // the reason, for the log
}

type quizClientQuestion struct {
	Question string   `json:"question"`
	Quote    string   `json:"quote,omitempty"`
	Image    string   `json:"image,omitempty"`
	Options  []string `json:"options"`
	Answer   int      `json:"answer"`
	Explain  string   `json:"explain,omitempty"`
}

type quizClient struct {
	Questions        []quizClientQuestion `json:"questions"`
	ShuffleQuestions bool                 `json:"shuffle_questions"`
	ShuffleOptions   bool                 `json:"shuffle_options"`
	Results          []doors.QuizResult   `json:"results"`
	ClosingText      string               `json:"closing_text,omitempty"`
}

type memoryClient struct {
	Pairs     int      `json:"pairs"`
	Images    []string `json:"images"`
	CardBack  string   `json:"card_back,omitempty"`
	PeekMS    int      `json:"peek_ms"`
	ShowMoves bool     `json:"show_moves"`
	ShowTimer bool     `json:"show_timer"`
	WinText   string   `json:"win_text,omitempty"`
}

type wordleClient struct {
	Words      []doors.WordleWord `json:"words"`
	MaxGuesses int                `json:"max_guesses"`
	WinText    string             `json:"win_text,omitempty"`
}

type puzzleClient struct {
	Mode        string `json:"mode"`
	Image       string `json:"image"`
	Grid        int    `json:"grid"`
	ShowPreview bool   `json:"show_preview"`
	WinText     string `json:"win_text,omitempty"`
}

type crosswordClient struct {
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	Cells       [][]*string       `json:"cells"`
	Entries     []crossword.Entry `json:"entries"`
	AllowCheck  bool              `json:"allow_check"`
	AllowReveal bool              `json:"allow_reveal_letter"`
	ClosingText string            `json:"closing_text,omitempty"`
}

type wrappedClient struct {
	Title     string               `json:"title"`
	OutroText string               `json:"outro_text,omitempty"`
	Slides    []doors.WrappedSlide `json:"slides"`
	Stats     *waparse.Stats       `json:"stats"`
}

// gameKey is stable per day and params, independent of the token, so the
// same progress applies to /d/{token} and the dev preview. extra (e.g. the
// crossword file) is hashed in as well: a new grid starts fresh.
func gameKey(door *doors.Door, extra ...[]byte) string {
	h := sha256.New()
	h.Write([]byte(door.Template + "\x00"))
	h.Write(door.Params)
	for _, e := range extra {
		h.Write([]byte{0})
		h.Write(e)
	}
	return fmt.Sprintf("sa-game-%d-%s", door.Day, hex.EncodeToString(h.Sum(nil))[:10])
}

// gameEnv is what building a game view needs besides the door.
type gameEnv struct {
	asset     func(string) string // asset path -> URL below the door's asset route
	assetsDir string
	now       time.Time
	year      int
}

func buildGame(door *doors.Door, env gameEnv) *gameView {
	asset := env.asset
	g := &gameView{Kind: door.Template, Key: gameKey(door)}
	switch {
	case door.Quiz != nil:
		p := door.Quiz
		g.Intro = paragraphs(p.Intro)
		c := quizClient{ShuffleQuestions: p.ShuffleQuestions, ShuffleOptions: p.ShuffleOptions,
			Results: p.Results, ClosingText: p.ClosingText}
		if c.Results == nil {
			c.Results = []doors.QuizResult{}
		}
		for _, q := range p.Questions {
			text := q.Question
			if text == "" {
				text = doors.DefaultQuoteQuestion
			}
			c.Questions = append(c.Questions, quizClientQuestion{Question: text, Quote: q.Quote, Image: asset(q.Image),
				Options: q.Options, Answer: *q.Answer, Explain: q.Explain})
		}
		g.Data = c
		g.Scripts = []string{"js/quiz.js"}
	case door.Memory != nil:
		p := door.Memory
		g.Intro = paragraphs(p.Intro)
		c := memoryClient{Pairs: p.Pairs, CardBack: asset(p.CardBack), PeekMS: p.PeekMS,
			ShowMoves: p.Moves(), ShowTimer: p.ShowTimer, WinText: p.WinText}
		for _, img := range p.Images {
			c.Images = append(c.Images, asset(img))
		}
		g.Data = c
		g.Scripts = []string{"js/memory.js"}
	case door.Wordle != nil:
		p := door.Wordle
		g.Intro = paragraphs(p.Intro)
		g.Data = wordleClient{Words: p.Words, MaxGuesses: p.MaxGuesses, WinText: p.WinText}
		g.Scripts = []string{"js/wordle-logic.js", "js/wordle.js"}
	case door.Puzzle != nil:
		p := door.Puzzle
		g.Intro = paragraphs(p.Intro)
		g.Data = puzzleClient{Mode: p.Mode, Image: asset(p.Image), Grid: p.Grid, ShowPreview: p.Preview(), WinText: p.WinText}
		g.Scripts = []string{"js/puzzle-logic.js", "js/puzzle.js"}
	case door.Crossword != nil:
		p := door.Crossword
		g.Intro = paragraphs(p.Intro)
		r, raw, err := p.Raster(env.assetsDir)
		if err != nil {
			g.Error, g.err = "Das Kreuzworträtsel ist noch nicht fertig gedruckt – schau später nochmal vorbei.", err
			return g
		}
		g.Key = gameKey(door, raw)
		entries := append([]crossword.Entry(nil), r.Entries...)
		crossword.SortEntries(entries)
		g.Data = crosswordClient{Width: r.Width, Height: r.Height, Cells: r.Cells, Entries: entries,
			AllowCheck: p.Check(), AllowReveal: p.AllowRevealLetter, ClosingText: p.ClosingText}
		g.Scripts = []string{"js/crossword-logic.js", "js/crossword.js"}
	case door.Wrapped != nil:
		p := door.Wrapped
		g.Intro = paragraphs(p.Intro)
		st, err := p.LoadStats(env.assetsDir)
		if err != nil {
			g.Error, g.err = "Die Statistik wird gerade noch ausgezählt – schau später nochmal vorbei.", err
			return g
		}
		// "Tage zusammen" as of the viewing day (but at least the door's date,
		// so the dev preview before the 22nd shows the real number).
		if since, err := waparse.ParseDate(st.Since); err == nil {
			asOf := advent.DateOf(env.now)
			if doorDate := (advent.Date{Year: env.year, Month: time.December, Day: door.Day}); advent.DaysBetween(asOf, doorDate) > 0 {
				asOf = doorDate
			}
			on := waparse.Date{Year: asOf.Year, Month: asOf.Month, Day: asOf.Day}
			st.AsOf = on.String()
			st.DaysTogether = max(0, waparse.DaysBetween(since, on))
		}
		title := p.Title
		if title == "" {
			title = fmt.Sprintf("Schatzo Wrapped %d", env.year)
		}
		g.Data = wrappedClient{Title: title, OutroText: p.OutroText, Slides: p.SlideList(), Stats: st}
		g.Scripts = []string{"js/wrapped-logic.js", "js/wrapped.js"}
		g.Styles = []string{"css/wrapped.css"}
	default:
		return nil
	}
	return g
}
