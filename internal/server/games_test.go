package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/clock"
	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/doors"
)

// exampleServer runs the shipped templates (config.example.json + doors/config.example.json).
func exampleServer(t *testing.T, dev bool) (*fixture, *config.Config, *doors.Registry) {
	t.Helper()
	cfg, err := config.Load("../../config.example.json", func(k string) string {
		if k == "ADVENT_DEV" && dev {
			return "1"
		}
		return ""
	})
	must(t, err)
	must(t, cfg.Validate())
	reg, err := doors.Load(cfg.DoorsConfig)
	must(t, err)
	must(t, reg.CheckDates(cfg.Year))
	clk := &clock.Fixed{T: berlin(2026, 12, 24, 12, 0, 0)}
	srv, err := New(Options{Config: cfg, Doors: reg, Clock: clk, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	must(t, err)
	return &fixture{srv: srv, h: srv.Handler(), clock: clk}, cfg, reg
}

var gameDataRE = regexp.MustCompile(`(?s)<script type="application/json" id="game-data">(.*?)</script>`)

// Every door of the example config renders (token route and dev preview) and
// every referenced asset is served.
func TestExampleDoorsRender(t *testing.T) {
	f, cfg, reg := exampleServer(t, true)
	for _, day := range advent.QRDays {
		door := reg.Get(day)
		for _, path := range []string{"/d/" + cfg.TokenForDay(day), "/dev/door/" + itoa(int64(day))} {
			rec := f.get(t, path)
			body := rec.Body.String()
			if rec.Code != 200 || strings.Contains(body, "Interner Fehler") || !strings.Contains(body, "</html>") {
				t.Fatalf("Tag %d %s: %d", day, path, rec.Code)
			}
			if !strings.Contains(body, "door-title") {
				t.Errorf("Tag %d %s: kein Türchen gerendert", day, path)
			}
			if doors.IsGame(door.Template) {
				m := gameDataRE.FindStringSubmatch(body)
				if m == nil {
					t.Fatalf("Tag %d: game-data fehlt", day)
				}
				var data map[string]any
				if err := json.Unmarshal([]byte(m[1]), &data); err != nil {
					t.Fatalf("Tag %d: game-data kein JSON: %v", day, err)
				}
				for _, want := range []string{`data-game="` + door.Template + `"`, "js/game.js", "css/games.css"} {
					if !strings.Contains(body, want) {
						t.Errorf("Tag %d: %q fehlt", day, want)
					}
				}
				if strings.Contains(body, "gebastelt") {
					t.Errorf("Tag %d: Platzhalter statt Spiel", day)
				}
			}
		}
		for _, a := range door.AssetRefs() {
			if rec := f.get(t, "/d/"+cfg.TokenForDay(day)+"/a/"+a); rec.Code != 200 {
				t.Errorf("Tag %d: Asset %s → %d", day, a, rec.Code)
			}
		}
	}
}

func TestExampleGameData(t *testing.T) {
	f, cfg, _ := exampleServer(t, true)
	data := func(day int) map[string]any {
		t.Helper()
		m := gameDataRE.FindStringSubmatch(f.get(t, "/d/"+cfg.TokenForDay(day)).Body.String())
		var d map[string]any
		must(t, json.Unmarshal([]byte(m[1]), &d))
		return d
	}
	tok := cfg.TokenForDay
	// asset URLs are resolved against the door's (date-locked) asset route
	if imgs := data(6)["images"].([]any); len(imgs) != 8 || imgs[0] != "/d/"+tok(6)+"/a/motiv-01.jpg" {
		t.Errorf("Memory 6: %v", imgs)
	}
	if d := data(18); d["pairs"].(float64) != 12 || d["show_timer"] != true || d["peek_ms"].(float64) != 0 {
		t.Errorf("Memory 18: %v", d)
	}
	words := data(9)["words"].([]any)
	if w := words[1].(map[string]any)["word"]; w != "GRÜßE" {
		t.Errorf("Wordle normalisiert nicht: %v", w)
	}
	if d := data(12); d["mode"] != "jigsaw" || d["grid"].(float64) != 4 || d["image"] != "/d/"+tok(12)+"/a/puzzle.jpg" {
		t.Errorf("Puzzle 12: %v", d)
	}
	q := data(15)["questions"].([]any)[0].(map[string]any)
	if q["question"] != doors.DefaultQuoteQuestion || q["quote"] == "" {
		t.Errorf("Quiz 15: %v", q)
	}
}

func TestGameKeyStable(t *testing.T) {
	reg, err := doors.Parse([]byte(`{"doors":{"9":{"template":"wordle","title":"x","params":{"words":[{"word":"KEKSE"}]}}}}`))
	must(t, err)
	reg2, err := doors.Parse([]byte(`{"doors":{"9":{"template":"wordle","title":"x","params":{"words":[{"word":"PLÄTZCHEN"}]}}}}`))
	must(t, err)
	k1, k2 := gameKey(reg.Get(9)), gameKey(reg2.Get(9))
	if !strings.HasPrefix(k1, "sa-game-9-") || k1 == k2 || gameKey(reg.Get(9)) != k1 {
		t.Fatalf("%q %q", k1, k2)
	}
}

// Banner hidden on 23 and 24 – also in the dev preview routes.
func TestBannerHiddenOn23And24(t *testing.T) {
	f, cfg, _ := exampleServer(t, true)
	f.clock.T = berlin(2026, 12, 1, 12, 0, 0)
	for _, day := range advent.QRDays {
		for _, path := range []string{"/dev/door/" + itoa(int64(day)), "/d/" + cfg.TokenForDay(day)} {
			if day > 1 && strings.HasPrefix(path, "/d/") {
				continue // still locked on 1.12.
			}
			has := strings.Contains(f.get(t, path).Body.String(), `class="sg-banner"`)
			if want := day != 23 && day != 24; has != want {
				t.Errorf("Tag %d %s: Banner %v, erwartet %v", day, path, has, want)
			}
		}
	}
	if b := f.get(t, "/dev/door/1").Body.String(); !strings.Contains(b, "Schatzogram – noch 23 Tage") {
		t.Error("Bannertext fehlt")
	}
}

func TestCountdownPlural(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 11, 29, 22, 58, 59))
	// 1 Tag 1 Stunde 1 Minute 1 Sekunde bis 1.12. 00:00
	b := f.get(t, "/d/tokenDay01").Body.String()
	for _, want := range []string{
		`data-d>1</span><span class="cd__l" data-dl>Tag<`, `data-h>01</span><span class="cd__l" data-hl>Stunde<`,
		`data-m>01</span><span class="cd__l" data-ml>Minute<`, `data-s>01</span><span class="cd__l" data-sl>Sekunde<`,
	} {
		if !strings.Contains(b, want) {
			t.Errorf("fehlt: %s", want)
		}
	}
	f.clock.T = berlin(2026, 11, 28, 21, 57, 58)
	b = f.get(t, "/d/tokenDay01").Body.String()
	for _, want := range []string{">Tage<", ">Stunden<", ">Minuten<", ">Sekunden<"} {
		if !strings.Contains(b, want) {
			t.Errorf("fehlt: %s", want)
		}
	}
	f.clock.T = berlin(2026, 11, 30, 23, 0, 0) // 0 Tage, 1 Stunde, 0 Minuten, 0 Sekunden
	b = f.get(t, "/d/tokenDay01").Body.String()
	for _, want := range []string{">Tage<", ">Stunde<", ">Minuten<", ">Sekunden<"} {
		if !strings.Contains(b, want) {
			t.Errorf("fehlt: %s", want)
		}
	}
}
