package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jakobneri/sandyadvent/internal/waparse"
)

func now() time.Time { return time.Date(2026, 11, 25, 10, 0, 0, 0, time.UTC) }

func TestRunBothFormats(t *testing.T) {
	dir := t.TempDir()
	var outs [][]byte
	for _, f := range []string{"testdata/android.txt", "testdata/ios.txt"} {
		out := filepath.Join(dir, filepath.Base(f)+".json")
		var stderr bytes.Buffer
		err := run([]string{"-in", f, "-out", out, "-alias", "Max Mustermann=Jakob", "-alias", "Erika 🌸=Sandy",
			"-since", "2024-02-14", "-asof", "2026-12-22"}, &bytes.Buffer{}, &stderr, now)
		if err != nil {
			t.Fatalf("%s: %v\n%s", f, err, stderr.String())
		}
		if !strings.Contains(stderr.String(), "→ Sandy") {
			t.Errorf("Namensliste fehlt: %s", stderr.String())
		}
		raw, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := waparse.ParseStats(raw); err != nil {
			t.Fatal(err)
		}
		outs = append(outs, raw)
	}
	if !bytes.Equal(outs[0], outs[1]) {
		t.Error("Android- und iOS-Export ergeben unterschiedliche stats.json")
	}
}

func TestRunErrors(t *testing.T) {
	cases := map[string][]string{
		"mindestens eine":   {},
		"Format":            {"-alias", "ohneGleich", "testdata/android.txt"},
		"-since":            {"-since", "14.02.2024", "testdata/android.txt"},
		"goodnight-pattern": {"-goodnight-pattern", "(", "testdata/android.txt"},
		"no such file":      {"-in", "gibtsnicht.txt"},
	}
	for want, args := range cases {
		err := run(args, &bytes.Buffer{}, &bytes.Buffer{}, now)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%v: %v (erwartet %q)", args, err, want)
		}
	}
	var stderr bytes.Buffer
	err := run([]string{"-alias", "Niemand=X", "testdata/android.txt"}, &bytes.Buffer{}, &stderr, now)
	if err == nil || !strings.Contains(err.Error(), "fehlt ein -alias") {
		t.Errorf("nur ein unpassendes -alias: %v", err)
	}
	if !strings.Contains(stderr.String(), "passt zu keinem Namen") {
		t.Errorf("Alias-Warnung fehlt: %s", stderr.String())
	}
}

// A pasted "Datum - Name: Text" line inside a message must not become a
// third person: with -alias it is continuation text, without it the run fails.
func TestRunPastedHeaderLine(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "export.txt")
	src := "24.12.23, 18:00 - Max Mustermann: Wo treffen wir uns?\n" +
		"24.12.23, 18:04 - Erika 🌸: Hab's dir weitergeleitet:\n" +
		"25.12.23, 18:05 - Treffpunkt: Bahnhof\n" +
		"24.12.23, 18:06 - Max Mustermann: Gute Nacht ❤️\n" +
		"24.12.23, 18:07 - Erika 🌸: Gute Nacht\n"
	if err := os.WriteFile(in, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	err := run([]string{"-alias", "Max Mustermann=Jakob", "-alias", "Erika 🌸=Sandy", in}, &out, &stderr, now)
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var st waparse.Stats
	if err := json.Unmarshal(out.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st.TotalMessages != 4 || len(st.Persons) != 2 {
		t.Errorf("total %d, persons %+v", st.TotalMessages, st.Persons)
	}
	if !strings.Contains(stderr.String(), "Treffpunkt") {
		t.Errorf("Hinweis auf Fortsetzungszeile fehlt: %s", stderr.String())
	}
	err = run([]string{in}, &bytes.Buffer{}, &bytes.Buffer{}, now)
	if err == nil || !strings.Contains(err.Error(), "3 Personen") || !strings.Contains(err.Error(), "Treffpunkt") {
		t.Errorf("ohne -alias: %v", err)
	}
}
