package doors

import (
	"strings"
	"testing"

	"github.com/jakobneri/sandyadvent/internal/advent"
)

// The table from CLAUDE.md.
var wantTemplates = map[int][]string{
	1: {"media"}, 3: {"quiz"}, 4: {"media"}, 6: {"memory"}, 7: {"media"}, 9: {"wordle"},
	10: {"media"}, 12: {"puzzle"}, 13: {"media"}, 15: {"quiz"}, 16: {"crossword"},
	18: {"memory", "puzzle"}, 19: {"media"}, 21: {"quiz"}, 22: {"wrapped"}, 23: {"media"}, 24: {"reveal"},
}

func TestExampleRegistry(t *testing.T) {
	reg, err := Load("../../doors/config.example.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Doors) != 17 {
		t.Fatalf("%d Türchen", len(reg.Doors))
	}
	for _, day := range advent.QRDays {
		d := reg.Get(day)
		if d == nil {
			t.Fatalf("Tag %d fehlt", day)
		}
		ok := false
		for _, tpl := range wantTemplates[day] {
			ok = ok || d.Template == tpl
		}
		if !ok {
			t.Errorf("Tag %d: Template %q, erwartet %v", day, d.Template, wantTemplates[day])
		}
		if d.Template == "media" && d.Media == nil {
			t.Errorf("Tag %d: media-Params nicht geparst", day)
		}
		if IsGame(d.Template) && d.Quiz == nil && d.Memory == nil && d.Wordle == nil && d.Puzzle == nil {
			t.Errorf("Tag %d: Spiel-Params nicht geparst", day)
		}
		if missing := d.MissingAssets("../../doors/assets-example"); len(missing) > 0 {
			t.Errorf("Tag %d: Dummy-Assets fehlen: %v", day, missing)
		}
	}
	if err := reg.CheckDates(2026); err != nil {
		t.Error(err)
	}
	if reg.Get(6).Memory.Pairs != 8 || reg.Get(18).Memory.Pairs != 12 || !reg.Get(18).Memory.ShowTimer {
		t.Error("Memory 6/18: Paare/Timer")
	}
	if reg.Get(12).Puzzle.Mode != PuzzleJigsaw || reg.Get(12).Puzzle.Grid != 4 {
		t.Error("Tag 12 muss jigsaw 4×4 sein")
	}
	// Door 23 is the teaser.
	if b := reg.Get(23).Media.Blocks; len(b) == 0 || b[0].Type != BlockTeaser {
		t.Error("Tag 23 braucht einen teaser-Block")
	}
}

func TestVoucherDayRefused(t *testing.T) {
	_, err := Parse([]byte(`{"doors":{"5":{"template":"crossword","title":"x"}}}`))
	if err == nil || !strings.Contains(err.Error(), "Gutschein-Tag") {
		t.Fatalf("err = %v", err)
	}
}

func TestDoorValidation(t *testing.T) {
	cases := map[string]string{
		`{"doors":{"0":{"template":"crossword","title":"x"}}}`:                                                                           "1–24",
		`{"doors":{"1":{"template":"disco","title":"x"}}}`:                                                                               "unbekanntes Template",
		`{"doors":{"1":{"template":"crossword","title":" "}}}`:                                                                           "title fehlt",
		`{"doors":{"1":{"template":"crossword","title":"x","params":[1]}}}`:                                                              "params muss ein Objekt",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[]}}}}`:                                                        "leer",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"gif"}]}}}}`:                                          "unbekannter Block-Typ",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"audio","src":"a.m4a"}]}}}}`:                          "nicht in assets",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"audio","src":"../a.m4a"}]}}}}`:                       "ungültiger Pfad",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"link","url":"javascript:alert(1)","label":"x"}]}}}}`: "http",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"yearsago","date":"13.12.2023"}]}}}}`:                 "date",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"slideshow","images":[],"interval_ms":100}]}}}}`:      "interval_ms",
		`{"doors":{"1":{"template":"media","title":"x","params":{"blocks":[{"type":"text","txt":"Tippfehler"}]}}}}`:                      "unknown field",
		`{"doors":{"1":{"template":"crossword","title":"x","assets":["/etc/passwd"]}}}`:                                                  "ungültiger Pfad",
	}
	for in, want := range cases {
		_, err := Parse([]byte(in))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s:\n  err = %v, want %q", in, err, want)
		}
	}
}

func TestMediaOK(t *testing.T) {
	reg, err := Parse([]byte(`{"doors":{"1":{"template":"media","title":"x","assets":["fotos/","a.m4a"],
		"params":{"blocks":[{"type":"audio","src":"a.m4a"},{"type":"slideshow","images":[{"src":"fotos/1.jpg"}]},
		{"type":"teaser"},{"type":"yearsago","date":"2023-12-13"}]}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	d := reg.Get(1)
	if !d.AllowsAsset("fotos/1.jpg") || !d.AllowsAsset("a.m4a") || d.AllowsAsset("b.m4a") || d.AllowsAsset("fotos") {
		t.Fatal("AllowsAsset falsch")
	}
}

func TestCleanAssetPath(t *testing.T) {
	good := []string{"a.jpg", "01/sprachnachricht.m4a", "fotos/2024/x y.jpg"}
	bad := []string{"", "/a", "../a", "a/../b", "a//b", "./a", "a/", ".hidden", "a/.git/config", `a\b`, "a\x00b", ".."}
	for _, p := range good {
		if c, err := CleanAssetPath(p); err != nil || c != p {
			t.Errorf("%q: %q %v", p, c, err)
		}
	}
	for _, p := range bad {
		if _, err := CleanAssetPath(p); err == nil {
			t.Errorf("%q akzeptiert", p)
		}
	}
}

func TestCheckDates(t *testing.T) {
	mk := func(date string) *Registry {
		t.Helper()
		reg, err := Parse([]byte(`{"doors":{"13":{"template":"media","title":"x","params":{"blocks":[{"type":"yearsago","date":"` + date + `"}]}}}}`))
		if err != nil {
			t.Fatal(err)
		}
		return reg
	}
	for _, ok := range []string{"2023-12-13", "2026-12-12", "2025-12-31"} {
		if err := mk(ok).CheckDates(2026); err != nil {
			t.Errorf("%s abgelehnt: %v", ok, err)
		}
	}
	for _, bad := range []string{"2026-12-13", "2026-12-14", "2027-01-01"} {
		if err := mk(bad).CheckDates(2026); err == nil || !strings.Contains(err.Error(), "Tag 13") {
			t.Errorf("%s: err = %v", bad, err)
		}
	}
}
