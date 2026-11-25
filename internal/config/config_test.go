package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
)

func env(m map[string]string) Env { return func(k string) string { return m[k] } }

func writeCfg(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExampleConfigIsValid(t *testing.T) {
	cfg, err := Load("../../config.example.json", env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Tokens) != 17 || len(cfg.MissingTokenDays()) != 0 {
		t.Fatalf("tokens=%d missing=%v", len(cfg.Tokens), cfg.MissingTokenDays())
	}
	if cfg.Reveal != (advent.Date{Year: 2026, Month: time.December, Day: 24}) {
		t.Fatalf("reveal %v", cfg.Reveal)
	}
	if !strings.HasSuffix(cfg.DoorsConfig, filepath.Join("doors", "config.example.json")) {
		t.Fatalf("doors_config %q", cfg.DoorsConfig)
	}
}

func TestDefaultsAndRelativePaths(t *testing.T) {
	p := writeCfg(t, `{}`)
	cfg, err := Load(p, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != DefaultListen || cfg.Year != 2026 || cfg.Schatzogram.RevealDate != "2026-12-24" {
		t.Fatalf("defaults: %+v", cfg)
	}
	if cfg.AssetsDir != filepath.Join(filepath.Dir(p), "doors", "assets") {
		t.Fatalf("assets_dir %q", cfg.AssetsDir)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEnvOverrides(t *testing.T) {
	p := writeCfg(t, `{"listen":"localhost:1","base_url":"http://a"}`)
	other := writeCfg(t, `{"listen":"localhost:2"}`)
	cfg, err := Load("", env(map[string]string{
		"ADVENT_CONFIG": other, "ADVENT_LISTEN": ":9999", "ADVENT_BASE_URL": "http://b", "ADVENT_DEV": "1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Path != other || cfg.Listen != ":9999" || cfg.BaseURL != "http://b" || !cfg.Dev {
		t.Fatalf("%+v", cfg)
	}
	// an explicit path (-config) wins over ADVENT_CONFIG
	if cfg, _ := Load(p, env(map[string]string{"ADVENT_CONFIG": other})); cfg.Path != p {
		t.Fatalf("Pfad %q", cfg.Path)
	}
	if _, err := Load(p, env(map[string]string{"ADVENT_DEV": "vielleicht"})); err == nil {
		t.Fatal("ADVENT_DEV=vielleicht akzeptiert")
	}
}

func TestAdventNowDevOnly(t *testing.T) {
	p := writeCfg(t, `{}`)
	cfg, err := Load(p, env(map[string]string{"ADVENT_NOW": "2026-12-05T08:00"}))
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Now.IsZero() || !cfg.NowIgnored {
		t.Fatal("ADVENT_NOW darf ohne Dev-Modus nicht wirken")
	}
	if _, ok := cfg.Clock().(interface{ Now() time.Time }); !ok {
		t.Fatal("kein Clock")
	}
	if d := time.Since(cfg.Clock().Now()); d < -time.Minute || d > time.Minute {
		t.Fatal("ohne Dev muss die Systemuhr gelten")
	}

	cfg, err = Load(p, env(map[string]string{"ADVENT_NOW": "2026-12-05T08:00", "ADVENT_DEV": "1"}))
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 12, 5, 8, 0, 0, 0, advent.Berlin)
	if !cfg.Now.Equal(want) {
		t.Fatalf("Now = %v", cfg.Now)
	}
	if d := cfg.Clock().Now().Sub(want); d < 0 || d > time.Minute {
		t.Fatalf("Dev-Uhr weicht ab: %v", d)
	}
	// dev via config file
	p2 := writeCfg(t, `{"dev":true}`)
	cfg, err = Load(p2, env(map[string]string{"ADVENT_NOW": "2026-12-05T08:00"}))
	if err != nil || cfg.Now.IsZero() {
		t.Fatalf("dev aus Datei: %v %v", cfg.Now, err)
	}
	if _, err := Load(p, env(map[string]string{"ADVENT_NOW": "morgen", "ADVENT_DEV": "1"})); err == nil {
		t.Fatal("ungültiges ADVENT_NOW akzeptiert")
	}
}

func TestTokenValidation(t *testing.T) {
	cases := []struct {
		name, json, wantErr string
	}{
		{"ok", `{"tokens":{"abcDEF-_09":1,"Zz0123456_":24}}`, ""},
		{"voucher day", `{"tokens":{"abcdefghij":2}}`, "Gutschein-Tag"},
		{"voucher day 20", `{"tokens":{"abcdefghij":20}}`, "Gutschein-Tag"},
		{"out of range", `{"tokens":{"abcdefghij":25}}`, "kein QR-Tag"},
		{"too short", `{"tokens":{"abc":1}}`, "genau 10 Zeichen"},
		{"too long", `{"tokens":{"abcdefghijk":1}}`, "genau 10 Zeichen"},
		{"bad chars", `{"tokens":{"abcdefghi/":1}}`, "genau 10 Zeichen"},
		{"umlaut", `{"tokens":{"abcdefghiä":1}}`, "genau 10 Zeichen"},
		{"two per day", `{"tokens":{"abcdefghij":1,"bbcdefghij":1}}`, "mehrere Tokens"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg, err := Parse([]byte(c.json))
			if err != nil {
				t.Fatal(err)
			}
			err = cfg.Validate()
			if c.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("err = %v, want %q", err, c.wantErr)
			}
		})
	}
}

func TestDuplicateTokenKeys(t *testing.T) {
	_, err := Parse([]byte(`{"year":2026,"tokens":{"abcdefghij":1,"abcdefghij":3}}`))
	if err == nil || !strings.Contains(err.Error(), "doppelte Tokens") {
		t.Fatalf("err = %v", err)
	}
	if _, err := Parse([]byte(`{"tokens":null}`)); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	if _, err := Parse([]byte(`{"tokenz":{}}`)); err == nil {
		t.Fatal("Tippfehler im Feldnamen akzeptiert")
	}
}

func TestOtherValidation(t *testing.T) {
	cfg, _ := Parse([]byte(`{"base_url":"192.168.0.1","schatzogram":{"reveal_date":"24.12.2026","welcome_url":"ftp://x"}}`))
	err := cfg.Validate()
	if err == nil {
		t.Fatal("erwartet Fehler")
	}
	for _, want := range []string{"base_url", "reveal_date", "welcome_url"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Fehler nennt %s nicht: %v", want, err)
		}
	}
}
