package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakobneri/sandyadvent/internal/crossword"
)

func TestRunExampleList(t *testing.T) {
	out := filepath.Join(t.TempDir(), "kw.json")
	var stderr bytes.Buffer
	code, err := run([]string{"-in", "beispiel-woerter.txt", "-out", out, "-seed", "1", "-attempts", "60", "-preview"}, &bytes.Buffer{}, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("%d %v\n%s", code, err, stderr.String())
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := crossword.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "Seed 1") || !strings.Contains(stderr.String(), "·") {
		t.Errorf("stderr: %s", stderr.String())
	}
}

// The example raster used by doors/config.example.json is valid and
// contains all words of the example list.
func TestExampleRasterMatchesList(t *testing.T) {
	raw, err := os.ReadFile("../../doors/assets-example/kreuzwort.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := crossword.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	list, _ := os.ReadFile("beispiel-woerter.txt")
	words, err := crossword.ReadWords("beispiel-woerter.txt", list, crossword.ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Entries) != len(words) {
		t.Errorf("%d Einträge, %d Wörter – make crossword neu ausführen", len(r.Entries), len(words))
	}
}

func TestRunStrict(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "w.txt")
	if err := os.WriteFile(in, []byte("KERZE;a\nZIMTSTERN;b\nBOBBY;c\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	code, err := run([]string{"-in", in, "-out", filepath.Join(dir, "o.json"), "-seed", "2", "-attempts", "5", "-strict"}, &bytes.Buffer{}, &stderr)
	if code != 2 || err == nil || !strings.Contains(stderr.String(), "BOBBY;c") {
		t.Fatalf("code %d err %v\n%s", code, err, stderr.String())
	}
	code, err = run([]string{"-in", in, "-seed", "2", "-attempts", "5"}, &bytes.Buffer{}, &bytes.Buffer{})
	if code != 0 || err != nil {
		t.Fatalf("ohne -strict: %d %v", code, err)
	}
	if err := os.WriteFile(in, []byte("GLÜH WEIN;a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run([]string{"-in", in}, &bytes.Buffer{}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "Leerzeichen") {
		t.Fatalf("%v", err)
	}
}
