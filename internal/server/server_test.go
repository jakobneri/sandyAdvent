package server

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/clock"
	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/doors"
)

const doorsJSON = `{"doors":{
 "1":{"template":"media","title":"Tür Eins","assets":["ton.m4a","fotos/"],
      "params":{"blocks":[{"type":"text","heading":"Hallo","text":"<b>fett</b>\nZeile 2"},{"type":"audio","src":"ton.m4a"},
      {"type":"slideshow","images":[{"src":"fotos/a.jpg","caption":"A"},{"src":"fotos/b.jpg"}]},
      {"type":"link","url":"https://example.org/x","label":"Link"},{"type":"yearsago","date":"2023-12-01"}]}},
 "3":{"template":"quiz","title":"Quiz","params":{"questions":[{"question":"<i>Wer?</i>","options":["Du","Ich"],"answer":0}]}},
 "16":{"template":"crossword","title":"Rätsel"},
 "9":{"template":"media","title":"Später","assets":["ton.m4a"],"params":{"blocks":[{"type":"audio","src":"ton.m4a"}]}},
 "23":{"template":"media","title":"Teaser","params":{"blocks":[{"type":"teaser"}]}},
 "24":{"template":"reveal","title":"Reveal"}
}}`

type fixture struct {
	srv   *Server
	h     http.Handler
	clock *clock.Fixed
}

func newFixture(t *testing.T, dev bool, now time.Time) *fixture {
	t.Helper()
	dir := t.TempDir()
	assets := filepath.Join(dir, "assets")
	must(t, os.MkdirAll(filepath.Join(assets, "fotos"), 0o755))
	audio := bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes
	must(t, os.WriteFile(filepath.Join(assets, "ton.m4a"), audio, 0o644))
	must(t, os.WriteFile(filepath.Join(assets, "fotos", "a.jpg"), []byte("jpg"), 0o644))
	must(t, os.WriteFile(filepath.Join(assets, "secret.txt"), []byte("geheim"), 0o644))
	must(t, os.WriteFile(filepath.Join(dir, "outside.m4a"), []byte("draussen"), 0o644))
	// symlink escaping the assets dir, listed as allowed asset
	_ = os.Symlink(filepath.Join(dir, "outside.m4a"), filepath.Join(assets, "fotos", "link.jpg"))

	cfg, err := config.Parse([]byte(`{"tokens":{"tokenDay01":1,"tokenDay03":3,"tokenDay04":4,"tokenDay09":9,"tokenDay16":16,"tokenDay23":23,"tokenDay24":24}}`))
	must(t, err)
	cfg.Dev = dev
	cfg.AssetsDir = assets
	must(t, cfg.Validate())
	reg, err := doors.Parse([]byte(doorsJSON))
	must(t, err)
	clk := &clock.Fixed{T: now}
	srv, err := New(Options{Config: cfg, Doors: reg, Clock: clk, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		Pick: func(int) int { return 0 }})
	must(t, err)
	return &fixture{srv: srv, h: srv.Handler(), clock: clk}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func berlin(y int, m time.Month, d, hh, mm, ss int) time.Time {
	return time.Date(y, m, d, hh, mm, ss, 0, advent.Berlin)
}

func (f *fixture) get(t *testing.T, path string, hdr ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	return rec
}

func TestNeutral404EverywhereIdentical(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 10, 12, 0, 0))
	ref := f.get(t, "/d/unknownTok")
	if ref.Code != 404 {
		t.Fatalf("code %d", ref.Code)
	}
	body := ref.Body.String()
	for _, w := range []string{"Advent", "Türchen", "Dezember", "Schatzogram", "Countdown", "app.css"} {
		if strings.Contains(body, w) {
			t.Errorf("404-Seite verrät %q", w)
		}
	}
	paths := []string{
		"/d/short", "/d/tokenDay02", "/d/tokenDay04", // invalid, unknown, token without door config
		"/d/unknownTok/a/ton.m4a", "/d/tokenDay01/a/secret.txt", "/d/tokenDay01/a/nope.m4a",
		"/dev/preview", "/dev/door/1", "/dev/door/1/a/ton.m4a", "/dev/countdown/1",
		"/admin", "/static/nope.css", "/static/", "/static/css", "/favicon.ico", "/d/", "/d/tokenDay01/",
	}
	for _, p := range paths {
		rec := f.get(t, p)
		if rec.Code != 404 || rec.Body.String() != body {
			t.Errorf("%s: code %d, identischer Body: %v", p, rec.Code, rec.Body.String() == body)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/d/tokenDay01", nil)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, req)
	if rec.Code != 404 || rec.Body.String() != body {
		t.Errorf("POST: %d", rec.Code)
	}
}

func TestHomeIsNeutral(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 10, 12, 0, 0))
	rec := f.get(t, "/")
	if rec.Code != 200 || strings.Contains(rec.Body.String(), "Advent") || strings.Contains(rec.Body.String(), "Türchen") {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
}

func TestDateLockBoundary(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 8, 23, 59, 59))
	rec := f.get(t, "/d/tokenDay09")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "countdown.js") || strings.Contains(rec.Body.String(), "Später") {
		t.Fatalf("23:59:59 muss Countdown zeigen: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "„"+DefaultCountdownLines[0]+"“") {
		t.Error("frecher Spruch fehlt")
	}
	// server time + target are embedded for the client
	if !strings.Contains(rec.Body.String(), `"target":`+itoa(advent.UnlockTime(2026, 9).UnixMilli())) ||
		!strings.Contains(rec.Body.String(), `"now":`+itoa(f.clock.T.UnixMilli())) {
		t.Errorf("Countdown-Daten fehlen:\n%s", rec.Body)
	}
	if got := f.get(t, "/d/tokenDay09/a/ton.m4a"); got.Code != 404 {
		t.Fatalf("Asset vor dem Öffnen: %d", got.Code)
	}

	f.clock.T = berlin(2026, 12, 9, 0, 0, 0)
	rec = f.get(t, "/d/tokenDay09")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "Später") || strings.Contains(rec.Body.String(), "countdown.js") {
		t.Fatalf("00:00 muss Tür zeigen")
	}
	if got := f.get(t, "/d/tokenDay09/a/ton.m4a"); got.Code != 200 {
		t.Fatalf("Asset nach dem Öffnen: %d", got.Code)
	}
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }

func TestDoorRendering(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 22, 9, 0, 0))
	rec := f.get(t, "/d/tokenDay01")
	body := rec.Body.String()
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	for _, want := range []string{
		"Tür Eins", "&lt;b&gt;fett&lt;/b&gt;<br>Zeile 2", `src="/d/tokenDay01/a/ton.m4a"`, "playsinline",
		`src="/d/tokenDay01/a/fotos/a.jpg"`, `rel="noopener noreferrer"`, "Schatzogram – noch 2 Tage",
		"Heute vor 3 Jahren", "1. Dezember 2023",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("fehlt: %q", want)
		}
	}
	if !strings.Contains(f.get(t, "/d/tokenDay23").Body.String(), "countdown.js") {
		t.Error("Tag 23 muss am 22.12. noch gesperrt sein")
	}
	if strings.Contains(body, "<b>fett") {
		t.Error("Text nicht escaped")
	}
	if h := rec.Header(); h.Get("Cache-Control") != "no-store" || h.Get("Referrer-Policy") != "no-referrer" ||
		!strings.Contains(h.Get("Content-Security-Policy"), "default-src 'self'") {
		t.Errorf("Header: %v", h)
	}
	// placeholder template (not implemented yet)
	if b := f.get(t, "/d/tokenDay16").Body.String(); !strings.Contains(b, "Dieses Türchen wird noch gebastelt") {
		t.Error("Platzhalter fehlt")
	}
	// game template: shell + JSON data (escaped) + scripts
	b3 := f.get(t, "/d/tokenDay03").Body.String()
	for _, want := range []string{`data-game="quiz"`, `id="game-data"`, "js/game.js", "js/quiz.js", "css/games.css", `\u003ci\u003eWer?`} {
		if !strings.Contains(b3, want) {
			t.Errorf("Quiz-Tür: %q fehlt", want)
		}
	}
	if strings.Contains(b3, "<i>Wer?") || strings.Contains(b3, "gebastelt") {
		t.Error("Quiz-Tür: unescaped oder Platzhalter")
	}
	// teaser: blurred logo inline + "Morgen."
	f.clock.T = berlin(2026, 12, 23, 0, 0, 0)
	b := f.get(t, "/d/tokenDay23").Body.String()
	if !strings.Contains(b, "sg-teaser__logo") || !strings.Contains(b, "<svg") || !strings.Contains(b, "Morgen.") {
		t.Error("Teaser unvollständig")
	}
	// no banner on 23 (teaser) and 24 (reveal)
	if strings.Contains(b, `class="sg-banner"`) || strings.Contains(b, "Schatzogram – noch") {
		t.Error("Tag 23 darf keinen Banner zeigen")
	}
	if b := f.get(t, "/d/tokenDay01").Body.String(); !strings.Contains(b, "Schatzogram – noch 1 Tag<") {
		t.Error("Banner am 23.12. muss 'noch 1 Tag' sagen")
	}
	f.clock.T = berlin(2026, 12, 24, 0, 0, 1)
	if b := f.get(t, "/d/tokenDay24").Body.String(); strings.Contains(b, `class="sg-banner"`) {
		t.Error("Tag 24 darf keinen Banner zeigen")
	}
	if b := f.get(t, "/d/tokenDay01").Body.String(); !strings.Contains(b, "Schatzogram – heute") {
		t.Error("Banner 'heute' fehlt")
	}
}

func TestYearsLabel(t *testing.T) {
	on := advent.Date{Year: 2026, Month: 12, Day: 13}
	cases := map[advent.Date]string{
		{Year: 2023, Month: 12, Day: 13}: "Heute vor 3 Jahren",
		{Year: 2025, Month: 12, Day: 13}: "Heute vor einem Jahr",
		{Year: 2024, Month: 7, Day: 1}:   "Vor 2 Jahren",
		{Year: 2026, Month: 12, Day: 12}: "Gestern",
		{Year: 2026, Month: 12, Day: 3}:  "Vor 10 Tagen",
		{Year: 2026, Month: 12, Day: 13}: "Heute",
		{Year: 2027, Month: 1, Day: 1}:   "Heute",
	}
	for then, want := range cases {
		if got := yearsLabel(then, on); got != want {
			t.Errorf("%v: %q, want %q", then, got, want)
		}
	}
}

func TestAssetServing(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 2, 0, 0, 0))
	rec := f.get(t, "/d/tokenDay01/a/ton.m4a")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "audio/mp4" || rec.Body.Len() != 1000 {
		t.Fatalf("%d %q %d", rec.Code, rec.Header().Get("Content-Type"), rec.Body.Len())
	}
	if rec.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("Accept-Ranges fehlt")
	}
	// Range request (iOS Safari: bytes=0-1)
	rec = f.get(t, "/d/tokenDay01/a/ton.m4a", "Range", "bytes=0-1")
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "01" || rec.Header().Get("Content-Range") != "bytes 0-1/1000" {
		t.Fatalf("Range: %d %q %q", rec.Code, rec.Body, rec.Header().Get("Content-Range"))
	}
	if rec := f.get(t, "/d/tokenDay01/a/fotos/a.jpg"); rec.Code != 200 || rec.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("jpg: %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	// traversal & friends
	for _, p := range []string{
		"/d/tokenDay01/a/../config.json", "/d/tokenDay01/a/%2e%2e/outside.m4a", "/d/tokenDay01/a/fotos/..%2f..%2foutside.m4a",
		"/d/tokenDay01/a/fotos%2f..%2f..%2foutside.m4a", "/d/tokenDay01/a/fotos/link.jpg", "/d/tokenDay01/a/fotos/",
		"/d/tokenDay01/a/fotos", "/d/tokenDay01/a/secret.txt", "/d/tokenDay01/a//etc/passwd", "/d/tokenDay01/a/fotos/.x",
		"/d/tokenDay03/a/ton.m4a", // other door's asset
	} {
		rec := f.get(t, p)
		if rec.Code == 200 {
			t.Errorf("%s: 200 (%q)", p, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "draussen") || strings.Contains(rec.Body.String(), "geheim") {
			t.Errorf("%s: Inhalt geleakt", p)
		}
	}
}

func TestDevRoutes(t *testing.T) {
	f := newFixture(t, true, berlin(2026, 11, 24, 12, 0, 0))
	rec := f.get(t, "/dev/preview")
	if rec.Code != 200 {
		t.Fatal(rec.Code)
	}
	b := rec.Body.String()
	for _, want := range []string{"Gutschein", "/dev/door/1", "/dev/countdown/9", "/d/tokenDay01", "keine Türchen-Config"} {
		if !strings.Contains(b, want) {
			t.Errorf("Vorschau: %q fehlt", want)
		}
	}
	if rec := f.get(t, "/dev/door/9"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "Später") {
		t.Fatalf("dev door ignoriert Sperre nicht: %d", rec.Code)
	}
	if rec := f.get(t, "/dev/door/9/a/ton.m4a"); rec.Code != 200 {
		t.Fatalf("dev asset %d", rec.Code)
	}
	if rec := f.get(t, "/dev/countdown/1"); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"reload":false`) {
		t.Fatalf("dev countdown %d", rec.Code)
	}
	for _, p := range []string{"/dev/door/2", "/dev/door/4", "/dev/door/25", "/dev/door/x", "/dev/countdown/2", "/dev/door/1/a/secret.txt"} {
		if rec := f.get(t, p); rec.Code != 404 {
			t.Errorf("%s: %d", p, rec.Code)
		}
	}
	// real route still locked in dev mode
	if b := f.get(t, "/d/tokenDay01").Body.String(); !strings.Contains(b, "countdown.js") {
		t.Error("Token-Route muss auch im Dev-Modus sperren")
	}
}

func TestStaticFiles(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 12, 1, 0, 0, 0))
	url := f.srv.static.URL("css/app.css")
	if !strings.Contains(url, "?v=") {
		t.Fatal(url)
	}
	rec := f.get(t, url)
	if rec.Code != 200 || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css") ||
		!strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Fatalf("%d %v", rec.Code, rec.Header())
	}
	if rec := f.get(t, "/static/css/app.css"); !strings.Contains(rec.Header().Get("Cache-Control"), "max-age=3600") {
		t.Error("ohne v= kurz cachen")
	}
	if rec := f.get(t, "/static/fonts/nunito-latin-400-normal.woff2"); rec.Code != 200 || rec.Header().Get("Content-Type") != "font/woff2" {
		t.Errorf("font %d %q", rec.Code, rec.Header().Get("Content-Type"))
	}
	etag := rec.Header().Get("ETag")
	if rec := f.get(t, url, "If-None-Match", etag); rec.Code != http.StatusNotModified {
		t.Errorf("ETag: %d", rec.Code)
	}
}

func TestCustomCountdownLines(t *testing.T) {
	f := newFixture(t, false, berlin(2026, 11, 30, 12, 0, 0))
	f.srv.lines = []string{"Eigener Spruch"}
	if b := f.get(t, "/d/tokenDay01").Body.String(); !strings.Contains(b, "„Eigener Spruch“") {
		t.Error("eigener Spruch fehlt")
	}
}

func TestMaskPath(t *testing.T) {
	if got := maskPath("/d/abcdefghij/a/x.jpg"); got != "/d/***/a/x.jpg" {
		t.Fatal(got)
	}
	if got := maskPath("/d/abcdefghij"); got != "/d/***" {
		t.Fatal(got)
	}
}
