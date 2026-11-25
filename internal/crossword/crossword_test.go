package crossword

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

func s(v string) *string { return &v }

// small valid grid:
//
//	K A T Z E
//	U · · · ·
//	S · · · ·
//	S · · · ·
func smallRaster() *Raster {
	n := (*string)(nil)
	return &Raster{Version: 1, Width: 5, Height: 4,
		Cells: [][]*string{
			{s("K"), s("A"), s("T"), s("Z"), s("E")},
			{s("U"), n, n, n, n},
			{s("S"), n, n, n, n},
			{s("S"), n, n, n, n},
		},
		Entries: []Entry{
			{Number: 1, Direction: Across, Row: 0, Col: 0, Answer: "KATZE", Clue: "Miau"},
			{Number: 1, Direction: Down, Row: 0, Col: 0, Answer: "KUSS", Clue: "Schmatz"},
		}}
}

func TestValidateOK(t *testing.T) {
	r := smallRaster()
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	data, err := Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	back, err := Parse(data)
	if err != nil {
		t.Fatalf("%v\n%s", err, data)
	}
	if back.Entries[1].Answer != "KUSS" || back.Letter(3, 0) != "S" || back.Letter(1, 1) != "" {
		t.Fatalf("%+v", back)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]func(r *Raster){
		"version muss 1":             func(r *Raster) { r.Version = 2 },
		"width/height":               func(r *Raster) { r.Width = 31 },
		"cells hat 3 Zeilen":         func(r *Raster) { r.Cells = r.Cells[:3] },
		"cells[1] hat 4 Spalten":     func(r *Raster) { r.Cells[1] = r.Cells[1][:4] },
		"genau einen Großbuchstaben": func(r *Raster) { r.Cells[0][1] = s("a") },
		"passt nicht zum Raster":     func(r *Raster) { r.Entries[0].Answer = "KATZA" },
		"ragt über den Rand":         func(r *Raster) { r.Entries[1].Answer = "KUSSE" },
		"außerhalb des Rasters":      func(r *Raster) { r.Entries[0].Row = 9 },
		"Standard-Nummerierung":      func(r *Raster) { r.Entries[1].Number = 2 },
		"clue fehlt":                 func(r *Raster) { r.Entries[0].Clue = " " },
		"direction muss":             func(r *Raster) { r.Entries[0].Direction = "diagonal" },
		"doppelter Eintrag":          func(r *Raster) { r.Entries = append(r.Entries, r.Entries[0]) },
		"nur ein Teil":               func(r *Raster) { r.Entries[0].Answer = "KATZ" },
		"hat aber keinen Eintrag":    func(r *Raster) { r.Entries = r.Entries[:1] },
		// letter touching KATZE from below creates the illegal word "AH"
		"\"AH\"":           func(r *Raster) { r.Cells[1][1] = s("H") },
		"entries ist leer": func(r *Raster) { r.Entries = nil },
	}
	for want, mutate := range cases {
		r := smallRaster()
		mutate(r)
		err := r.Validate()
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", want, err)
		}
	}
	if _, err := Parse([]byte(`{"version":1,"width":2,"height":2,"cells":[],"entries":[],"extra":1}`)); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("unknown field: %v", err)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{
		"grüße":         "GRÜßE",
		"STRAẞE":        "STRAßE",
		"Gru\u0308ße":   "GRÜßE", // NFD
		" mistelzweig ": "MISTELZWEIG",
		"Äpfel":         "ÄPFEL",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Normalize("Gru\u0308ße"); utf8.RuneCountInString(got) != 5 {
		t.Errorf("NFD nicht zusammengesetzt: %q", got)
	}
}

func TestReadWords(t *testing.T) {
	txt := "\xef\xbb\xbf# Kommentar\r\n\r\nglühwein;Heiß\r\nGRÜßE ; Jeden Morgen\nschatz;Du; und nur du\n"
	ws, err := ReadWords("liste.txt", []byte(txt), ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 3 || ws[0].Word != "GLÜHWEIN" || ws[1].Word != "GRÜßE" || ws[1].Clue != "Jeden Morgen" || ws[2].Clue != "Du; und nur du" {
		t.Fatalf("%+v", ws)
	}
	js := `[{"word":"Kerze","clue":"Brennt"},{"word":"zimtstern","clue":"Gebäck"}]`
	ws, err = ReadWords("liste.json", []byte(js), ReadOptions{})
	if err != nil || len(ws) != 2 || ws[1].Word != "ZIMTSTERN" {
		t.Fatalf("%v %+v", err, ws)
	}
	for in, want := range map[string]string{
		"WEIHNACHTS MARKT;x": "Leerzeichen",
		"ADVENT4;x":          "Ziffern",
		"GLÜH-WEIN;x":        "Bindestrich",
		"CAFÉ;x":             `"É"`,
		"A;x":                "zu kurz",
		"KERZE":              "WORT;Hinweis",
		"KERZE;":             "Hinweis fehlt",
		"KERZE;a\nkerze;b":   "steht schon",
		"# nur Kommentar":    "keine Wörter",
	} {
		_, err := ReadWords("x.txt", []byte(in), ReadOptions{})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want %q", in, err, want)
		}
	}
	ws, err = ReadWords("x.txt", []byte("Weihnachts Markt;Glühwein & Co."), ReadOptions{StripSpaces: true})
	if err != nil || ws[0].Word != "WEIHNACHTSMARKT" {
		t.Fatalf("strip: %v %+v", err, ws)
	}
}

var sample = []Word{
	{"MISTELZWEIG", "Darunter wird geküsst"}, {"SCHATZO", "Du"}, {"GLÜHWEIN", "Heiß"},
	{"PLÄTZCHEN", "Gebacken"}, {"KAKAO", "Mit Sahne"}, {"SCHNEE", "Weiß"}, {"LEBKUCHEN", "Herz"},
	{"GRÜßE", "Jeden Morgen"}, {"RENTIER", "Zieht"}, {"ZIMTSTERN", "Gebäck"}, {"SOCKEN", "Warm"}, {"KERZE", "Brennt"},
}

// checkGrid is an independent validity check: every pair of neighbouring
// letters must belong to the same entry, every entry matches the grid, every
// placed word appears once with its clue.
func checkGrid(t *testing.T, r *Raster, words []Word, unplaced []Word) {
	t.Helper()
	if err := r.Validate(); err != nil {
		t.Fatalf("ungültig: %v\n%s", err, r.Preview())
	}
	type pos struct{ r, c int }
	acrossOf, downOf := map[pos]int{}, map[pos]int{}
	for i, e := range r.Entries {
		for k := range []rune(e.Answer) {
			if e.Direction == Across {
				acrossOf[pos{e.Row, e.Col + k}] = i + 1
			} else {
				downOf[pos{e.Row + k, e.Col}] = i + 1
			}
		}
	}
	for i := 0; i < r.Height; i++ {
		for j := 0; j < r.Width; j++ {
			if r.Letter(i, j) == "" {
				continue
			}
			if acrossOf[pos{i, j}] == 0 && downOf[pos{i, j}] == 0 {
				t.Errorf("Buchstabe (%d,%d) ohne Wort", i, j)
			}
			if r.Letter(i, j+1) != "" && (acrossOf[pos{i, j}] == 0 || acrossOf[pos{i, j}] != acrossOf[pos{i, j + 1}]) {
				t.Errorf("unerlaubt waagerecht benachbart: (%d,%d)\n%s", i, j, r.Preview())
			}
			if r.Letter(i+1, j) != "" && (downOf[pos{i, j}] == 0 || downOf[pos{i, j}] != downOf[pos{i + 1, j}]) {
				t.Errorf("unerlaubt senkrecht benachbart: (%d,%d)\n%s", i, j, r.Preview())
			}
		}
	}
	clue := map[string]string{}
	for _, e := range r.Entries {
		if _, dup := clue[e.Answer]; dup {
			t.Errorf("%s doppelt", e.Answer)
		}
		clue[e.Answer] = e.Clue
	}
	if len(r.Entries)+len(unplaced) != len(words) {
		t.Errorf("%d Einträge + %d unplatziert != %d Wörter", len(r.Entries), len(unplaced), len(words))
	}
	for _, w := range words {
		got, ok := clue[w.Word]
		isUnplaced := false
		for _, u := range unplaced {
			isUnplaced = isUnplaced || u.Word == w.Word
		}
		if ok == isUnplaced || (ok && got != w.Clue) {
			t.Errorf("%s: im Raster %v (Hinweis %q), unplatziert %v", w.Word, ok, got, isUnplaced)
		}
	}
}

func TestGenerateDeterministic(t *testing.T) {
	o := Options{Seed: 2026, Attempts: 40}
	a, err := Generate(sample, o)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(sample, o)
	if err != nil {
		t.Fatal(err)
	}
	ja, _ := Marshal(a.Raster)
	jb, _ := Marshal(b.Raster)
	if !bytes.Equal(ja, jb) {
		t.Fatalf("gleicher Seed, anderes Raster:\n%s\n%s", a.Raster.Preview(), b.Raster.Preview())
	}
	checkGrid(t, a.Raster, sample, a.Unplaced)
	if a.Raster.Width > DefaultMaxSize || a.Raster.Height > DefaultMaxSize {
		t.Errorf("zu groß: %dx%d", a.Raster.Width, a.Raster.Height)
	}
	c, err := Generate(sample, Options{Seed: 7, Attempts: 40})
	if err != nil {
		t.Fatal(err)
	}
	jc, _ := Marshal(c.Raster)
	if bytes.Equal(ja, jc) {
		t.Log("Seed 7 ergibt zufällig dasselbe Raster")
	}
}

func TestGenerateManySeedsValid(t *testing.T) {
	for seed := uint64(1); seed <= 25; seed++ {
		res, err := Generate(sample, Options{Seed: seed, Attempts: 8, MaxSize: 11 + int(seed%5)})
		if err != nil {
			t.Fatal(err)
		}
		checkGrid(t, res.Raster, sample, res.Unplaced)
	}
}

func TestGenerateUmlauts(t *testing.T) {
	ws, err := ReadWords("x.txt", []byte("glühwein;a\ngrüße;b\nstraße;c\näpfel;d\nmöhre;e\nsüß;f"), ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := Generate(ws, Options{Seed: 3, Attempts: 150})
	if err != nil {
		t.Fatal(err)
	}
	checkGrid(t, res.Raster, ws, res.Unplaced)
	if len(res.Unplaced) > 0 {
		t.Fatalf("unplatziert: %v\n%s", res.Unplaced, res.Raster.Preview())
	}
	found := map[string]bool{}
	for _, row := range res.Raster.Cells {
		for _, c := range row {
			if c != nil {
				found[*c] = true
				if utf8.RuneCountInString(*c) != 1 {
					t.Errorf("Feld mit %q", *c)
				}
			}
		}
	}
	for _, want := range []string{"Ü", "ß", "Ö", "Ä"} {
		if !found[want] {
			t.Errorf("%s fehlt im Raster", want)
		}
	}
}

func TestGenerateUnplaceable(t *testing.T) {
	ws := []Word{{"KERZE", "a"}, {"ZIMTSTERN", "b"}, {"BOBBY", "d"}}
	res, err := Generate(ws, Options{Seed: 1, Attempts: 10})
	if err != nil {
		t.Fatal(err)
	}
	checkGrid(t, res.Raster, ws, res.Unplaced)
	if len(res.Unplaced) != 1 || res.Unplaced[0].Word != "BOBBY" {
		t.Fatalf("unplatziert: %+v\n%s", res.Unplaced, res.Raster.Preview())
	}
	// too long for max-size
	res, err = Generate([]Word{{"KERZE", "a"}, {"WEIHNACHTSMARKT", "b"}}, Options{Seed: 1, Attempts: 3, MaxSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unplaced) != 1 || res.Unplaced[0].Word != "WEIHNACHTSMARKT" {
		t.Fatalf("%+v", res.Unplaced)
	}
	if _, err := Generate(ws, Options{MaxSize: 99}); err == nil {
		t.Error("max-size 99 akzeptiert")
	}
	// no word fits at all: German user error, not "interner Fehler"
	_, err = Generate([]Word{{"WEIHNACHTEN", "x"}, {"SCHNEEMANN", "y"}}, Options{Seed: 1, Attempts: 5, MaxSize: 8})
	if err == nil {
		t.Fatal("kein Fehler, obwohl kein Wort passt")
	}
	if msg := err.Error(); strings.Contains(msg, "interner Fehler") || !strings.Contains(msg, "max-size 8") || !strings.Contains(msg, "11 Buchstaben") {
		t.Errorf("Fehlermeldung: %q", msg)
	}
}

func TestPreview(t *testing.T) {
	got := smallRaster().Preview()
	if !strings.HasPrefix(got, "K A T Z E\nU · · · ·\n") {
		t.Fatalf("%q", got)
	}
}
