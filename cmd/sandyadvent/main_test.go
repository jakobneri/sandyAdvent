package main

import (
	"bytes"
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
