package doors

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const inlineGrid = `{"version":1,"width":5,"height":4,
	"cells":[["K","A","T","Z","E"],["U",null,null,null,null],["S",null,null,null,null],["S",null,null,null,null]],
	"entries":[{"number":1,"direction":"across","row":0,"col":0,"answer":"KATZE","clue":"Miau"},
	           {"number":1,"direction":"down","row":0,"col":0,"answer":"KUSS","clue":"Schmatz"}]}`

func TestCrosswordParams(t *testing.T) {
	reg, err := Parse([]byte(door("crossword", `{"intro":"Los","grid":"kw.json","allow_reveal_letter":true,"closing_text":"Yay"}`, `["kw.json"]`)))
	if err != nil {
		t.Fatal(err)
	}
	p := reg.Get(3).Crossword
	if p == nil || p.GridPath != "kw.json" || !p.Check() || !p.AllowRevealLetter || p.Inline != nil {
		t.Fatalf("%+v", p)
	}
	if refs := reg.Get(3).AssetRefs(); len(refs) != 1 || refs[0] != "kw.json" {
		t.Errorf("AssetRefs %v", refs)
	}
	reg, err = Parse([]byte(door("crossword", `{"grid":`+inlineGrid+`,"allow_check":false}`, "")))
	if err != nil {
		t.Fatal(err)
	}
	p = reg.Get(3).Crossword
	if p.Inline == nil || p.Check() || len(reg.Get(3).AssetRefs()) != 0 {
		t.Fatalf("%+v", p)
	}
	if r, raw, err := p.Raster("/nonexistent"); err != nil || raw != nil || r.Entries[1].Answer != "KUSS" {
		t.Fatalf("%v %v", r, err)
	}
	expectErrs(t, "crossword", map[string]string{
		`{}`:                          "grid fehlt",
		`{"grid":""}`:                 "grid fehlt",
		`{"grid":"../x.json"}`:        "ungültiger Pfad",
		`{"grid":"nicht.json"}`:       "steht aber nicht in assets",
		`{"grid":42}`:                 "Pfad (Text) oder ein Raster-Objekt",
		`{"grid":{"version":1}}`:      "grid (inline)",
		`{"grid":"a.jpg","raster":1}`: "unknown field",
		`{"grid":` + strings.Replace(inlineGrid, `"KUSS"`, `"KUSE"`, 1) + `}`: "passt nicht zum Raster",
	})
}

func TestCrosswordDataFile(t *testing.T) {
	dir := t.TempDir()
	write := func(name, s string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	reg, err := Parse([]byte(door("crossword", `{"grid":"kw.json"}`, `["kw.json"]`)))
	if err != nil {
		t.Fatal(err)
	}
	// missing: no error in CheckData (warning via MissingAssets), error when rendering
	if err := reg.CheckData(dir); err != nil {
		t.Errorf("fehlende Datei: %v", err)
	}
	if m := reg.Get(3).MissingAssets(dir); len(m) != 1 {
		t.Errorf("MissingAssets %v", m)
	}
	if _, _, err := reg.Get(3).Crossword.Raster(dir); err == nil {
		t.Error("fehlende Datei ohne Fehler")
	}
	write("kw.json", strings.Replace(inlineGrid, `"Miau"`, `""`, 1))
	if err := reg.CheckData(dir); err == nil || !strings.Contains(err.Error(), "clue fehlt") || !strings.Contains(err.Error(), "Tag 3") {
		t.Errorf("ungültiges Raster: %v", err)
	}
	write("kw.json", inlineGrid)
	if err := reg.CheckData(dir); err != nil {
		t.Error(err)
	}
	r, raw, err := reg.Get(3).Crossword.Raster(dir)
	if err != nil || r.Width != 5 || len(raw) == 0 {
		t.Fatalf("%v %v", r, err)
	}
}

func TestWrappedParams(t *testing.T) {
	reg, err := Parse([]byte(door("wrapped", `{"stats":"stats.json","title":"Wrapped","outro_text":"Danke",
		"slides":[{"id":"days","title":"So lange schon"},{"id":"emojis"},{"id":"response","text":"Schnell!"}]}`, `["stats.json"]`)))
	if err != nil {
		t.Fatal(err)
	}
	p := reg.Get(3).Wrapped
	list := p.SlideList()
	ids := []string{}
	for _, s := range list {
		ids = append(ids, s.ID)
	}
	if strings.Join(ids, ",") != "intro,days,emojis,response,outro" || list[1].Title != "So lange schon" {
		t.Fatalf("%v", list)
	}
	def := (&WrappedParams{}).SlideList()
	if len(def) != len(WrappedDefaultOrder) || def[0].ID != "intro" || def[len(def)-1].ID != "outro" {
		t.Errorf("%v", def)
	}
	expectErrs(t, "wrapped", map[string]string{
		`{}`:                      "stats fehlt",
		`{"stats":"/etc/x.json"}`: "ungültiger Pfad",
		`{"stats":"x.json"}`:      "steht aber nicht in assets",
		`{"stats":"a.jpg","slides":[{"id":"x"}]}`:                   `id "x" unbekannt`,
		`{"stats":"a.jpg","slides":[{"id":"days"},{"id":"days"}]}`:  "doppelt",
		`{"stats":"a.jpg","slides":[{"id":"days"},{"id":"intro"}]}`: "intro muss die erste",
		`{"stats":"a.jpg","slides":[{"id":"outro"},{"id":"days"}]}`: "outro muss die letzte",
		`{"stats":"a.jpg","year":2026}`:                             "unknown field",
	})
}

func TestWrappedStatsFile(t *testing.T) {
	reg, err := Parse([]byte(door("wrapped", `{"stats":"stats.json"}`, `["stats.json"]`)))
	if err != nil {
		t.Fatal(err)
	}
	st, err := reg.Get(3).Wrapped.LoadStats("../../doors/assets-example")
	if err != nil || st.TotalMessages == 0 || len(st.Persons) != 2 {
		t.Fatalf("%v %+v", err, st)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "stats.json"), []byte(`{"version":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := reg.CheckData(dir); err == nil || !strings.Contains(err.Error(), "totalMessages") {
		t.Errorf("%v", err)
	}
}
