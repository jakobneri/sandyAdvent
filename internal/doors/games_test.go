package doors

import (
	"strings"
	"testing"
)

// door wraps params into a one-door registry for template tpl on day 3.
func door(tpl, params, assets string) string {
	if assets == "" {
		assets = "[]"
	}
	return `{"doors":{"3":{"template":"` + tpl + `","title":"x","assets":` + assets + `,"params":` + params + `}}}`
}

func expectErrs(t *testing.T, tpl string, cases map[string]string) {
	t.Helper()
	for params, want := range cases {
		_, err := Parse([]byte(door(tpl, params, `["a.jpg","b.jpg","c.jpg","fotos/"]`)))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s %s:\n  err = %v\n  want %q", tpl, params, err, want)
		}
	}
}

func TestQuizParams(t *testing.T) {
	reg, err := Parse([]byte(door("quiz", `{"intro":"Hallo","shuffle_options":true,
		"questions":[{"question":"Wo?","options":["A","B","C"],"answer":2,"explain":"weil","image":"a.jpg"},
		             {"quote":"Nur noch eine Folge.","options":["Jakob","Sandy","Beide"],"answer":0}],
		"results":[{"min_score":0,"text":"null"},{"min_score":2,"text":"perfekt"},{"min_score":1,"text":"halb"}],
		"closing_text":"Ende"}`, `["a.jpg"]`)))
	if err != nil {
		t.Fatal(err)
	}
	q := reg.Get(3).Quiz
	if q == nil || len(q.Questions) != 2 || *q.Questions[0].Answer != 2 || !q.ShuffleOptions {
		t.Fatalf("%+v", q)
	}
	for score, want := range map[int]string{0: "null", 1: "halb", 2: "perfekt"} {
		if got := q.ResultText(score); got != want {
			t.Errorf("ResultText(%d) = %q, want %q", score, got, want)
		}
	}
	if refs := reg.Get(3).AssetRefs(); len(refs) != 1 || refs[0] != "a.jpg" {
		t.Errorf("AssetRefs %v", refs)
	}
	// results are optional
	if _, err := Parse([]byte(door("quiz", `{"questions":[{"question":"?","options":["a","b"],"answer":1}]}`, ""))); err != nil {
		t.Error(err)
	}
	expectErrs(t, "quiz", map[string]string{
		`{}`:               "questions ist leer",
		`{"questions":[]}`: "questions ist leer",
		`{"fragen":[]}`:    "unknown field",
		`{"questions":[{"options":["a","b"],"answer":0}]}`:                                                                                  "question (oder quote) fehlt",
		`{"questions":[{"question":"?","options":["a"],"answer":0}]}`:                                                                       "2 bis 6",
		`{"questions":[{"question":"?","options":["1","2","3","4","5","6","7"],"answer":0}]}`:                                               "2 bis 6",
		`{"questions":[{"question":"?","options":["a","b"]}]}`:                                                                              "answer fehlt",
		`{"questions":[{"question":"?","options":["a","b"],"answer":2}]}`:                                                                   "außerhalb von options",
		`{"questions":[{"question":"?","options":["a","b"],"answer":-1}]}`:                                                                  "außerhalb von options",
		`{"questions":[{"question":"?","options":["a",""],"answer":0}]}`:                                                                    "options[1] ist leer",
		`{"questions":[{"question":"?","options":["Ja","ja"],"answer":0}]}`:                                                                 "doppelt",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0,"image":"x.jpg"}]}`:                                                   "nicht in assets",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0,"image":"../x.jpg"}]}`:                                                "ungültiger Pfad",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0}],"results":[{"min_score":1,"text":"x"}]}`:                            "min_score 0",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0}],"results":[{"min_score":0,"text":"x"},{"min_score":2,"text":"y"}]}`: "zwischen 0 und 1",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0}],"results":[{"min_score":0,"text":""}]}`:                             "results[0].text fehlt",
		`{"questions":[{"question":"?","options":["a","b"],"answer":0}],"results":[{"min_score":0,"text":"a"},{"min_score":0,"text":"b"}]}`: "doppelt",
	})
	if _, err := Parse([]byte(`{"doors":{"3":{"template":"quiz","title":"x"}}}`)); err == nil || !strings.Contains(err.Error(), "params fehlen") {
		t.Errorf("ohne params: %v", err)
	}
}

func TestMemoryParams(t *testing.T) {
	reg, err := Parse([]byte(door("memory", `{"pairs":2,"images":["a.jpg","b.jpg","c.jpg"],"card_back":"fotos/back.jpg","peek_ms":1500,"show_timer":true}`,
		`["a.jpg","b.jpg","c.jpg","fotos/"]`)))
	if err != nil {
		t.Fatal(err)
	}
	m := reg.Get(3).Memory
	if m.Pairs != 2 || !m.Moves() || !m.ShowTimer || m.PeekMS != 1500 {
		t.Fatalf("%+v", m)
	}
	if refs := reg.Get(3).AssetRefs(); len(refs) != 4 || refs[3] != "fotos/back.jpg" {
		t.Errorf("AssetRefs %v", refs)
	}
	off := false
	m.ShowMoves = &off
	if m.Moves() {
		t.Error("show_moves:false ignoriert")
	}
	expectErrs(t, "memory", map[string]string{
		`{"pairs":1,"images":["a.jpg"]}`:                              "zwischen 2 und 18",
		`{"pairs":19,"images":["a.jpg"]}`:                             "zwischen 2 und 18",
		`{"pairs":3,"images":["a.jpg","b.jpg"]}`:                      "mindestens so viele Bilder wie pairs (3), hat 2",
		`{"pairs":2,"images":["a.jpg","a.jpg"]}`:                      "doppelt",
		`{"pairs":2,"images":["a.jpg","x.jpg"]}`:                      "nicht in assets",
		`{"pairs":2,"images":["a.jpg","b.jpg"],"peek_ms":-1}`:         "peek_ms",
		`{"pairs":2,"images":["a.jpg","b.jpg"],"peek_ms":99999}`:      "peek_ms",
		`{"pairs":2,"images":["a.jpg","b.jpg"],"card_back":"/etc/x"}`: "ungültiger Pfad",
		`{"pairs":2,"images":["a.jpg","b.jpg"],"timer":true}`:         "unknown field",
	})
}

func TestWordleParams(t *testing.T) {
	reg, err := Parse([]byte(door("wordle", `{"words":[{"word":"kekse","hint":"lecker"},{"word":" Grüße "},{"word":"Gru\u0308\u00dfe"},
		{"word":"STRAẞE"},{"word":"weihnachtsze"}]}`, "")))
	if err != nil {
		t.Fatal(err)
	}
	w := reg.Get(3).Wordle
	got := []string{}
	for _, x := range w.Words {
		got = append(got, x.Word)
	}
	if strings.Join(got, ",") != "KEKSE,GRÜßE,GRÜßE,STRAßE,WEIHNACHTSZE" {
		t.Errorf("Normalisierung: %v", got)
	}
	if w.MaxGuesses != 6 {
		t.Errorf("Standard max_guesses: %d", w.MaxGuesses)
	}
	expectErrs(t, "wordle", map[string]string{
		`{"words":[]}`:                                         "mindestens ein Wort",
		`{"words":[{"word":""}]}`:                              "word fehlt",
		`{"words":[{"word":"AB"}]}`:                            "erlaubt sind 3 bis 12",
		`{"words":[{"word":"WEIHNACHTSMARKT"}]}`:               "hat 15 Buchstaben",
		`{"words":[{"word":"ZWEI WORTE"}]}`:                    "nur Buchstaben",
		`{"words":[{"word":"T-SHIRT"}]}`:                       "nur Buchstaben",
		`{"words":[{"word":"CAFÉ"}]}`:                          "nur Buchstaben",
		`{"words":[{"word":"KEKS1"}]}`:                         "nur Buchstaben",
		`{"words":[{"word":"KEKSE"}],"max_guesses":11}`:        "max_guesses",
		`{"words":[{"word":"KEKSE"}],"max_guesses":-1}`:        "max_guesses",
		`{"words":[{"word":"KEKSE"}],"allow_any_guess":false}`: "kein Wörterbuch",
		`{"words":[{"wort":"KEKSE"}]}`:                         "unknown field",
	})
}

func TestNormalizeWord(t *testing.T) {
	for in, want := range map[string]string{
		"straße": "STRAßE", "STRAẞE": "STRAßE", "äöü": "ÄÖÜ", "A\u0308pfel": "ÄPFEL", " kekse ": "KEKSE", "x\u0308": "X\u0308",
	} {
		if got := NormalizeWord(in); got != want {
			t.Errorf("NormalizeWord(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPuzzleParams(t *testing.T) {
	for params, grid := range map[string]int{
		`{"mode":"jigsaw","image":"a.jpg"}`:                               4,
		`{"mode":"sliding","image":"a.jpg"}`:                              3,
		`{"mode":"sliding","image":"a.jpg","grid":6}`:                     6,
		`{"mode":"jigsaw","image":"a.jpg","grid":3,"show_preview":false}`: 3,
	} {
		reg, err := Parse([]byte(door("puzzle", params, `["a.jpg"]`)))
		if err != nil {
			t.Fatalf("%s: %v", params, err)
		}
		p := reg.Get(3).Puzzle
		if p.Grid != grid {
			t.Errorf("%s: grid %d", params, p.Grid)
		}
		if p.Preview() == strings.Contains(params, "false") {
			t.Errorf("%s: Preview %v", params, p.Preview())
		}
	}
	expectErrs(t, "puzzle", map[string]string{
		`{"image":"a.jpg"}`:                           "mode fehlt",
		`{"mode":"slide","image":"a.jpg"}`:            `mode "slide" unbekannt`,
		`{"mode":"jigsaw"}`:                           "image fehlt",
		`{"mode":"jigsaw","image":"x.jpg"}`:           "nicht in assets",
		`{"mode":"jigsaw","image":"a.jpg","grid":2}`:  "zwischen 3 und 6",
		`{"mode":"sliding","image":"a.jpg","grid":7}`: "zwischen 3 und 6",
	})
}
