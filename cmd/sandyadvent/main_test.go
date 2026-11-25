package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exampleConfig = "../../config.example.json"

// Outside dev mode the server must refuse to start with the EXAMPLE- tokens
// of the template (main exits with status 1 on any error from run).
func TestServeRefusesExampleTokens(t *testing.T) {
	t.Setenv("ADVENT_DEV", "0")
	t.Setenv("ADVENT_LISTEN", "127.0.0.1:0")
	var out, errOut bytes.Buffer
	err := run([]string{"serve", "-config", exampleConfig}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "Beispiel-Token") || !strings.Contains(err.Error(), "Dev-Modus") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheck(t *testing.T) {
	var out, errOut bytes.Buffer
	t.Setenv("ADVENT_DEV", "0")
	if err := run([]string{"check", "-config", exampleConfig}, &out, &errOut); err == nil || !strings.Contains(err.Error(), "EXAMPLE-") {
		t.Fatalf("check ohne Dev: err = %v", err)
	}
	t.Setenv("ADVENT_DEV", "1")
	out.Reset()
	if err := run([]string{"check", "-config", exampleConfig}, &out, &errOut); err != nil {
		t.Fatalf("check im Dev-Modus: %v", err)
	}
	if !strings.Contains(out.String(), "OK:") || !strings.Contains(out.String(), "17 Türchen") {
		t.Fatalf("Ausgabe: %q", out.String())
	}
}

// An invalid data file (crossword raster, stats.json) stops start/check with
// the day and the reason; a missing one is only a warning.
func TestCheckDataFiles(t *testing.T) {
	t.Setenv("ADVENT_DEV", "1")
	dir := t.TempDir()
	assets := filepath.Join(dir, "assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	doorsJSON := `{"doors":{"16":{"template":"crossword","title":"K","assets":["kw.json"],"params":{"grid":"kw.json"}},
		"22":{"template":"wrapped","title":"W","assets":["stats.json"],"params":{"stats":"stats.json"}}}}`
	cfg := `{"doors_config":"doors.json","assets_dir":"assets","tokens":{}}`
	for name, body := range map[string]string{"doors.json": doorsJSON, "config.json": cfg} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	if err := run([]string{"check", "-config", filepath.Join(dir, "config.json")}, &out, &errOut); err != nil {
		t.Fatalf("fehlende Dateien sind nur eine Warnung: %v", err)
	}
	if !strings.Contains(errOut.String(), "Dateien fehlen") {
		t.Errorf("Warnung fehlt: %s", errOut.String())
	}
	if err := os.WriteFile(filepath.Join(assets, "stats.json"), []byte(`{"version":1,"totalMessages":0}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"check", "-config", filepath.Join(dir, "config.json")}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "Tag 22") || !strings.Contains(err.Error(), "totalMessages") {
		t.Fatalf("ungültige stats.json: %v", err)
	}
}
