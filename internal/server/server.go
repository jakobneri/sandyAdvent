// Package server is the HTTP side of the calendar: token routing, date lock,
// countdown, door rendering, door assets, static files and dev routes.
package server

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/clock"
	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/doors"
	"github.com/jakobneri/sandyadvent/web"
)

// Server serves the calendar.
type Server struct {
	cfg    *config.Config
	doors  *doors.Registry
	clock  clock.Clock
	log    *slog.Logger
	lines  []string
	pick   func(n int) int
	static *staticFiles
	pages  map[string]*template.Template
	logo   template.HTML

	notFoundBody []byte
	homeBody     []byte
}

// Options configure New.
type Options struct {
	Config *config.Config
	Doors  *doors.Registry
	Clock  clock.Clock     // default: cfg.Clock()
	Logger *slog.Logger    // default: slog.Default()
	Pick   func(n int) int // random index for countdown lines (tests)
}

// New builds a server. The config must already be validated.
func New(o Options) (*Server, error) {
	if o.Config == nil || o.Doors == nil {
		return nil, errors.New("server: Config und Doors erforderlich")
	}
	s := &Server{cfg: o.Config, doors: o.Doors, clock: o.Clock, log: o.Logger, pick: o.Pick}
	if s.clock == nil {
		s.clock = o.Config.Clock()
	}
	if s.log == nil {
		s.log = slog.Default()
	}
	if s.pick == nil {
		s.pick = rand.IntN
	}
	s.lines = DefaultCountdownLines
	if len(o.Config.CountdownLines) > 0 {
		s.lines = o.Config.CountdownLines
	}
	staticFS, err := fs.Sub(web.FS, "static")
	if err != nil {
		return nil, err
	}
	if s.static, err = newStaticFiles(staticFS); err != nil {
		return nil, err
	}
	logo, err := fs.ReadFile(web.FS, "brand/logo-mark.svg")
	if err != nil {
		return nil, err
	}
	s.logo = template.HTML(strings.Replace(string(logo), "<svg ", `<svg aria-hidden="true" focusable="false" `, 1)) // #nosec: embedded, trusted
	if err := s.parseTemplates(); err != nil {
		return nil, err
	}
	if s.notFoundBody, err = s.renderBytes("neutral", neutralView{Title: "Seite nicht gefunden", Heading: "404", Text: "Diese Seite gibt es nicht."}); err != nil {
		return nil, err
	}
	if s.homeBody, err = s.renderBytes("neutral", neutralView{Title: "Hallo", Heading: "Hallo!", Text: "Hier gibt es gerade nichts zu sehen."}); err != nil {
		return nil, err
	}
	return s, nil
}

// Handler returns the root handler with all routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleHome)
	mux.HandleFunc("GET /d/{token}", s.handleDoor)
	mux.HandleFunc("GET /d/{token}/a/{path...}", s.handleDoorAsset)
	mux.Handle("GET /static/{path...}", s.static.handler(s.notFound))
	if s.cfg.Dev {
		mux.HandleFunc("GET /dev/{$}", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/dev/preview", http.StatusFound)
		})
		mux.HandleFunc("GET /dev/preview", s.handlePreview)
		mux.HandleFunc("GET /dev/door/{day}", s.handleDevDoor)
		mux.HandleFunc("GET /dev/door/{day}/a/{path...}", s.handleDevAsset)
		mux.HandleFunc("GET /dev/countdown/{day}", s.handleDevCountdown)
	}
	mux.HandleFunc("/", s.notFound)
	return s.withHeaders(mux)
}

const csp = "default-src 'self'; img-src 'self' data:; media-src 'self'; style-src 'self'; " +
	"script-src 'self'; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; " +
	"form-action 'self'; frame-ancestors 'none'"

func (s *Server) withHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("Referrer-Policy", "no-referrer") // tokens must never leak via Referer
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Robots-Tag", "noindex, nofollow")
		h.Set("X-Frame-Options", "DENY")
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			s.log.Info("request", "method", r.Method, "path", maskPath(r.URL.Path), "status", rec.status,
				"dur", time.Since(start).Round(time.Millisecond))
		}
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// maskPath hides tokens in logs: /d/AbC.../a/x -> /d/***/a/x.
func maskPath(p string) string {
	if !strings.HasPrefix(p, "/d/") {
		return p
	}
	rest := p[3:]
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		return "/d/***" + rest[i:]
	}
	return "/d/***"
}

// notFound is the single neutral 404 for every failure.
func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNotFound)
	if r.Method != http.MethodHead {
		_, _ = w.Write(s.notFoundBody)
	}
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	_, _ = w.Write(s.homeBody)
}

// lookup resolves a token to its day and door. Every failure is "not found".
func (s *Server) lookup(token string) (int, *doors.Door, bool) {
	if !config.ValidToken(token) {
		return 0, nil, false
	}
	day, ok := s.cfg.Tokens[token]
	if !ok || !advent.IsQRDay(day) {
		return 0, nil, false
	}
	door := s.doors.Get(day)
	if door == nil {
		return 0, nil, false
	}
	return day, door, true
}

func (s *Server) unlocked(day int) bool {
	return advent.IsUnlocked(s.clock.Now(), s.cfg.Year, day)
}

func (s *Server) handleDoor(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	day, door, ok := s.lookup(token)
	if !ok {
		s.notFound(w, r)
		return
	}
	if !s.unlocked(day) {
		s.renderCountdown(w, r, day, true)
		return
	}
	s.renderDoor(w, r, door, "/d/"+token+"/a/")
}

func (s *Server) handleDoorAsset(w http.ResponseWriter, r *http.Request) {
	day, door, ok := s.lookup(r.PathValue("token"))
	if !ok || !s.unlocked(day) {
		s.notFound(w, r)
		return
	}
	s.serveAsset(w, r, door, r.PathValue("path"))
}

// devDay parses {day} for dev routes: any QR day with a door config.
func (s *Server) devDay(r *http.Request) (int, *doors.Door, bool) {
	day, err := strconv.Atoi(r.PathValue("day"))
	if err != nil || !advent.IsQRDay(day) {
		return 0, nil, false
	}
	door := s.doors.Get(day)
	return day, door, door != nil
}

func (s *Server) handleDevDoor(w http.ResponseWriter, r *http.Request) {
	day, door, ok := s.devDay(r)
	if !ok {
		s.notFound(w, r)
		return
	}
	s.renderDoor(w, r, door, fmt.Sprintf("/dev/door/%d/a/", day))
}

func (s *Server) handleDevAsset(w http.ResponseWriter, r *http.Request) {
	_, door, ok := s.devDay(r)
	if !ok {
		s.notFound(w, r)
		return
	}
	s.serveAsset(w, r, door, r.PathValue("path"))
}

func (s *Server) handleDevCountdown(w http.ResponseWriter, r *http.Request) {
	day, err := strconv.Atoi(r.PathValue("day"))
	if err != nil || !advent.IsQRDay(day) {
		s.notFound(w, r)
		return
	}
	// Preview only: never auto-reload (the day may already be open).
	s.renderCountdown(w, r, day, false)
}

func (s *Server) render(w http.ResponseWriter, status int, page string, data any) {
	body, err := s.renderBytes(page, data)
	if err != nil {
		s.log.Error("template", "page", page, "err", err)
		http.Error(w, "Interner Fehler", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (s *Server) renderBytes(page string, data any) ([]byte, error) {
	t, ok := s.pages[page]
	if !ok {
		return nil, fmt.Errorf("unbekannte Seite %q", page)
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "page", data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
