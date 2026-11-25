package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/jakobneri/sandyadvent/internal/doors"
)

// gameView is the server-rendered shell of a game door. The game itself is
// built by web/static/js/<kind>.js from Data (JSON in the page).
type gameView struct {
	Kind    string
	Key     string     // localStorage key: day + hash of the params (changes reset progress)
	Intro   [][]string // paragraphs
	Data    any
	Scripts []string // static paths, loaded after js/game.js
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

// gameKey is stable per day and params, independent of the token, so the
// same progress applies to /d/{token} and the dev preview.
func gameKey(door *doors.Door) string {
	sum := sha256.Sum256(append([]byte(door.Template+"\x00"), door.Params...))
	return fmt.Sprintf("sa-game-%d-%s", door.Day, hex.EncodeToString(sum[:])[:10])
}

func buildGame(door *doors.Door, asset func(string) string) *gameView {
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
	default:
		return nil
	}
	return g
}
