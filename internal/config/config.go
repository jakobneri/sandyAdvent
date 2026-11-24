// Package config loads config.json, applies ADVENT_* env overrides and
// validates everything that must never be wrong on the Pi (tokens!).
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/clock"
)

// TokenLength is the exact length of every door token.
const TokenLength = 10

// ExampleTokenPrefix marks dummy tokens from config.example.json.
const ExampleTokenPrefix = "EXAMPLE-"

// Defaults.
const (
	DefaultConfigPath  = "config.json"
	DefaultListen      = "localhost:8081"
	DefaultDoorsConfig = "doors/config.json"
	DefaultAssetsDir   = "doors/assets"
	DefaultRevealDate  = "2026-12-24"
	DefaultWelcomeURL  = "https://sg.n3ri.com/welcome"
)

// Schatzogram holds the shared interface values (docs/interface.md).
type Schatzogram struct {
	WelcomeURL string `json:"welcome_url"`
	RevealDate string `json:"reveal_date"`
}

// Admin is reserved for M4 (Basic Auth for /admin).
type Admin struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Config is config.json.
type Config struct {
	Listen         string         `json:"listen"`
	BaseURL        string         `json:"base_url"`
	Year           int            `json:"year"`
	DoorsConfig    string         `json:"doors_config"`
	AssetsDir      string         `json:"assets_dir"`
	Tokens         map[string]int `json:"tokens"`
	Schatzogram    Schatzogram    `json:"schatzogram"`
	Admin          Admin          `json:"admin"`
	OpenLog        string         `json:"open_log"`
	Dev            bool           `json:"dev"`
	CountdownLines []string       `json:"countdown_lines"`

	// Derived, not part of the file.
	Path       string      `json:"-"` // path the config was loaded from
	Reveal     advent.Date `json:"-"`
	Now        time.Time   `json:"-"` // ADVENT_NOW (zero if unset or ignored)
	NowIgnored bool        `json:"-"` // ADVENT_NOW was set but dev mode is off
}

// Env is the subset of the environment the config cares about.
type Env func(string) string

// Load reads the config file and applies env overrides. The path is, in
// order: the path argument (-config flag), $ADVENT_CONFIG, config.json.
// It does not validate; call Validate.
func Load(path string, getenv Env) (*Config, error) {
	if getenv == nil {
		getenv = os.Getenv
	}
	if path == "" {
		path = getenv("ADVENT_CONFIG")
	}
	if path == "" {
		path = DefaultConfigPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config lesen: %w", err)
	}
	cfg, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	cfg.Path = path
	if err := cfg.applyEnv(getenv); err != nil {
		return nil, err
	}
	cfg.resolvePaths(filepath.Dir(path))
	return cfg, nil
}

// Parse decodes config JSON, applies defaults and rejects duplicate tokens.
func Parse(raw []byte) (*Config, error) {
	if dups, err := duplicateTokenKeys(raw); err != nil {
		return nil, err
	} else if len(dups) > 0 {
		return nil, fmt.Errorf("doppelte Tokens in \"tokens\": %s", strings.Join(dups, ", "))
	}
	cfg := &Config{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("ungültiges JSON: %w", err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Listen == "" {
		c.Listen = DefaultListen
	}
	if c.Year == 0 {
		c.Year = 2026
	}
	if c.DoorsConfig == "" {
		c.DoorsConfig = DefaultDoorsConfig
	}
	if c.AssetsDir == "" {
		c.AssetsDir = DefaultAssetsDir
	}
	if c.Schatzogram.RevealDate == "" {
		c.Schatzogram.RevealDate = DefaultRevealDate
	}
	if c.Schatzogram.WelcomeURL == "" {
		c.Schatzogram.WelcomeURL = DefaultWelcomeURL
	}
	if c.Tokens == nil {
		c.Tokens = map[string]int{}
	}
}

func (c *Config) applyEnv(getenv Env) error {
	if v := getenv("ADVENT_LISTEN"); v != "" {
		c.Listen = v
	}
	if v := getenv("ADVENT_BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if v := getenv("ADVENT_DEV"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("ADVENT_DEV=%q: erwartet 1/0/true/false", v)
		}
		c.Dev = b
	}
	if v := getenv("ADVENT_NOW"); v != "" {
		if !c.Dev {
			c.NowIgnored = true
			return nil
		}
		t, err := clock.ParseNow(v, advent.Berlin)
		if err != nil {
			return err
		}
		c.Now = t
	}
	return nil
}

// resolvePaths makes relative file paths relative to the config file's dir,
// so the server works no matter which directory systemd starts it in.
func (c *Config) resolvePaths(dir string) {
	abs := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(dir, p)
	}
	c.DoorsConfig = abs(c.DoorsConfig)
	c.AssetsDir = abs(c.AssetsDir)
	c.OpenLog = abs(c.OpenLog)
}

// Clock returns the clock to use: ADVENT_NOW (dev only) or the system clock.
func (c *Config) Clock() clock.Clock {
	if c.Dev && !c.Now.IsZero() {
		return clock.NewOffset(clock.System{}, c.Now)
	}
	return clock.System{}
}

// ValidToken reports whether s has the token format (10 chars, URL-safe base64 alphabet).
func ValidToken(s string) bool {
	if len(s) != TokenLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'a' && ch <= 'z', ch >= 'A' && ch <= 'Z', ch >= '0' && ch <= '9', ch == '-', ch == '_':
		default:
			return false
		}
	}
	return true
}

// Validate checks the loaded config. All problems are reported at once.
func (c *Config) Validate() error {
	var errs []error
	if c.Year < 2000 || c.Year > 2100 {
		errs = append(errs, fmt.Errorf("year %d unplausibel", c.Year))
	}
	reveal, err := advent.ParseDate(c.Schatzogram.RevealDate)
	if err != nil {
		errs = append(errs, fmt.Errorf("schatzogram.reveal_date: %w", err))
	}
	c.Reveal = reveal
	if c.BaseURL != "" && !strings.HasPrefix(c.BaseURL, "http://") && !strings.HasPrefix(c.BaseURL, "https://") {
		errs = append(errs, fmt.Errorf("base_url %q muss mit http:// oder https:// beginnen", c.BaseURL))
	}
	if !strings.HasPrefix(c.Schatzogram.WelcomeURL, "http://") && !strings.HasPrefix(c.Schatzogram.WelcomeURL, "https://") {
		errs = append(errs, fmt.Errorf("schatzogram.welcome_url %q muss mit http:// oder https:// beginnen", c.Schatzogram.WelcomeURL))
	}
	errs = append(errs, ValidateTokens(c.Tokens)...)
	if ex := c.ExampleTokens(); len(ex) > 0 && !c.Dev {
		errs = append(errs, fmt.Errorf("%d Beispiel-Tokens (EXAMPLE-…) aus config.example.json sind nur im Dev-Modus erlaubt – "+
			"echte Tokens mit 'make tokens' bzw. 'sandyadvent gen-tokens -write' erzeugen (zum Testen: ADVENT_DEV=1)", len(ex)))
	}
	return errors.Join(errs...)
}

// ExampleTokens returns the placeholder tokens (EXAMPLE-…) in the config, sorted.
func (c *Config) ExampleTokens() []string {
	var ex []string
	for _, tok := range sortedKeys(c.Tokens) {
		if strings.HasPrefix(tok, ExampleTokenPrefix) {
			ex = append(ex, tok)
		}
	}
	return ex
}

// ValidateTokens checks the token map: format, only QR days, one token per day.
func ValidateTokens(tokens map[string]int) []error {
	var errs []error
	byDay := map[int][]string{}
	for _, tok := range sortedKeys(tokens) {
		day := tokens[tok]
		switch {
		case !ValidToken(tok):
			errs = append(errs, fmt.Errorf("token %q: muss genau %d Zeichen aus [A-Za-z0-9_-] haben", tok, TokenLength))
		case advent.IsVoucherDay(day):
			errs = append(errs, fmt.Errorf("token %q: Tag %d ist ein Gutschein-Tag und darf keinen Token haben", tok, day))
		case !advent.IsQRDay(day):
			errs = append(errs, fmt.Errorf("token %q: Tag %d ist kein QR-Tag", tok, day))
		}
		byDay[day] = append(byDay[day], tok)
	}
	days := make([]int, 0, len(byDay))
	for d := range byDay {
		days = append(days, d)
	}
	sort.Ints(days)
	for _, d := range days {
		if len(byDay[d]) > 1 {
			errs = append(errs, fmt.Errorf("Tag %d hat mehrere Tokens (%s) – genau einer erlaubt", d, strings.Join(byDay[d], ", ")))
		}
	}
	return errs
}

// MissingTokenDays returns QR days without a token.
func (c *Config) MissingTokenDays() []int {
	have := map[int]bool{}
	for _, d := range c.Tokens {
		have[d] = true
	}
	var missing []int
	for _, d := range advent.QRDays {
		if !have[d] {
			missing = append(missing, d)
		}
	}
	return missing
}

// TokenForDay returns the token of a day ("" if none).
func (c *Config) TokenForDay(day int) string {
	for _, tok := range sortedKeys(c.Tokens) {
		if c.Tokens[tok] == day {
			return tok
		}
	}
	return ""
}

// LogWarnings logs non-fatal problems.
func (c *Config) LogWarnings(log *slog.Logger) {
	if c.NowIgnored {
		log.Warn("ADVENT_NOW ist gesetzt, wird aber ignoriert (nur im Dev-Modus wirksam)")
	}
	if missing := c.MissingTokenDays(); len(missing) > 0 {
		log.Warn("QR-Tage ohne Token", "tage", missing)
	}
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// duplicateTokenKeys scans the raw JSON for keys that occur more than once in
// the top-level "tokens" object (encoding/json would silently keep the last).
func duplicateTokenKeys(raw []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("ungültiges JSON: %w", err)
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("ungültiges JSON: Objekt erwartet")
	}
	var dups []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("ungültiges JSON: %w", err)
		}
		key, _ := k.(string)
		if key != "tokens" {
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil, fmt.Errorf("ungültiges JSON: %w", err)
			}
			continue
		}
		var obj json.RawMessage
		if err := dec.Decode(&obj); err != nil {
			return nil, fmt.Errorf("ungültiges JSON: %w", err)
		}
		keys, err := ObjectKeys(obj)
		if err != nil {
			return nil, fmt.Errorf("tokens: %w", err)
		}
		seen := map[string]bool{}
		for _, k := range keys {
			if seen[k] {
				dups = append(dups, k)
			}
			seen[k] = true
		}
	}
	return dups, nil
}

// ObjectKeys returns the keys of a JSON object in file order (with duplicates).
// "null" yields no keys.
func ObjectKeys(raw []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok == nil {
		return nil, nil
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, errors.New("Objekt erwartet")
	}
	var keys []string
	for dec.More() {
		k, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := k.(string)
		keys = append(keys, key)
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	if _, err := dec.Token(); err != nil && err != io.EOF {
		return nil, err
	}
	return keys, nil
}
