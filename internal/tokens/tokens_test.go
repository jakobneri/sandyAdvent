package tokens

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/config"
)

func TestGenerate(t *testing.T) {
	m, err := Generate(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != len(advent.QRDays) {
		t.Fatalf("%d Tokens", len(m))
	}
	if errs := config.ValidateTokens(m); len(errs) > 0 {
		t.Fatal(errs)
	}
	days := map[int]bool{}
	for tok, d := range m {
		if !config.ValidToken(tok) || advent.IsVoucherDay(d) {
			t.Fatalf("%q -> %d", tok, d)
		}
		days[d] = true
	}
	if len(days) != 17 {
		t.Fatal("nicht alle QR-Tage")
	}
	var back map[string]int
	if err := json.Unmarshal(MarshalMap(m, ""), &back); err != nil || len(back) != 17 {
		t.Fatalf("MarshalMap: %v", err)
	}
}

func TestFillKeepsExisting(t *testing.T) {
	existing := map[string]int{"keepMe_123": 1, "EXAMPLE-03": 3}
	m, err := Fill(existing, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if m["keepMe_123"] != 1 {
		t.Fatal("vorhandener Token überschrieben")
	}
	if _, ok := m["EXAMPLE-03"]; ok {
		t.Fatal("EXAMPLE-Token nicht ersetzt")
	}
	if len(m) != 17 {
		t.Fatalf("%d", len(m))
	}
	forced, _ := Fill(existing, nil, true)
	if _, ok := forced["keepMe_123"]; ok {
		t.Fatal("-force muss alles neu erzeugen")
	}
}

func TestWriteConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	orig := `{
  "listen": "localhost:8081",
  "tokens": {"keepMe_123": 1, "EXAMPLE-04": 4},
  "schatzogram": {"reveal_date": "2026-12-24"},
  "dev": false
}`
	if err := os.WriteFile(p, []byte(orig), 0o640); err != nil {
		t.Fatal(err)
	}
	added, err := WriteConfig(p, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 16 {
		t.Fatalf("added %v", added)
	}
	raw, _ := os.ReadFile(p)
	cfg, err := config.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Tokens["keepMe_123"] != 1 || len(cfg.Tokens) != 17 || cfg.Schatzogram.RevealDate != "2026-12-24" {
		t.Fatalf("%+v", cfg)
	}
	// key order preserved
	iL, iT, iD := bytes.Index(raw, []byte(`"listen"`)), bytes.Index(raw, []byte(`"tokens"`)), bytes.Index(raw, []byte(`"dev"`))
	if !(iL < iT && iT < iD) {
		t.Fatalf("Reihenfolge verändert:\n%s", raw)
	}
	if st, _ := os.Stat(p); st.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v", st.Mode())
	}
	// idempotent
	added, err = WriteConfig(p, nil, false)
	if err != nil || len(added) != 0 {
		t.Fatalf("zweiter Lauf: %v %v", added, err)
	}
	raw2, _ := os.ReadFile(p)
	if !bytes.Equal(raw, raw2) {
		t.Fatal("zweiter Lauf hat Datei verändert")
	}
}

func TestWriteConfigRefusesInvalid(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(p, []byte(`{"tokens":{"abcdefghij":2}}`), 0o600)
	if _, err := WriteConfig(p, nil, false); err == nil || !strings.Contains(err.Error(), "Gutschein") {
		t.Fatalf("err = %v", err)
	}
	if _, err := WriteConfig(p, nil, true); err != nil {
		t.Fatal(err)
	}
}
