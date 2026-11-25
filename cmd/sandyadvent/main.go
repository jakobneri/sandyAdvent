// Command sandyadvent is the advent calendar server.
//
//	sandyadvent [serve]              start the server (default)
//	sandyadvent check                validate config + doors and report
//	sandyadvent gen-tokens [-write]  generate tokens for the 17 QR days
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jakobneri/sandyadvent/internal/advent"
	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/doors"
	"github.com/jakobneri/sandyadvent/internal/server"
	"github.com/jakobneri/sandyadvent/internal/tokens"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "serve":
		return cmdServe(args, stderr)
	case "check":
		return cmdCheck(args, stdout, stderr)
	case "gen-tokens":
		return cmdGenTokens(args, stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return nil
	default:
		usage(stderr)
		return fmt.Errorf("unbekannter Befehl %q", cmd)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `sandyadvent – digitaler Adventskalender

Befehle:
  serve        Server starten (Standard)
  check        config.json + Türchen-Config prüfen
  gen-tokens   Tokens für die 17 QR-Tage erzeugen (-write schreibt in config.json)

Gemeinsame Flags: -config <pfad> (Standard: $ADVENT_CONFIG oder config.json)
Env: ADVENT_CONFIG, ADVENT_LISTEN, ADVENT_BASE_URL, ADVENT_DEV=1, ADVENT_NOW=2026-12-05T08:00 (nur Dev)
`)
}

func loadAll(path string, log *slog.Logger) (*config.Config, *doors.Registry, error) {
	cfg, err := config.Load(path, os.Getenv)
	if err != nil {
		return nil, nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("%s ungültig:\n%w", cfg.Path, err)
	}
	reg, err := doors.Load(cfg.DoorsConfig)
	if err != nil {
		return nil, nil, err
	}
	if err := reg.CheckDates(cfg.Year); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", cfg.DoorsConfig, err)
	}
	if err := reg.CheckData(cfg.AssetsDir); err != nil {
		return nil, nil, fmt.Errorf("%s: %w", cfg.AssetsDir, err)
	}
	cfg.LogWarnings(log)
	for _, day := range advent.QRDays {
		d := reg.Get(day)
		if d == nil {
			log.Warn("QR-Tag ohne Türchen-Config", "tag", day)
			continue
		}
		if missing := d.MissingAssets(cfg.AssetsDir); len(missing) > 0 {
			log.Warn("Dateien fehlen", "tag", day, "dir", cfg.AssetsDir, "dateien", missing)
		}
	}
	return cfg, reg, nil
}

func cmdServe(args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "", "Pfad zur config.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(stderr, nil))
	slog.SetDefault(log)
	cfg, reg, err := loadAll(*cfgPath, log)
	if err != nil {
		return err
	}
	srv, err := server.New(server.Options{Config: cfg, Doors: reg, Logger: log})
	if err != nil {
		return err
	}
	hs := &http.Server{
		Addr:              cfg.Listen,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- hs.ListenAndServe() }()
	attrs := []any{"listen", cfg.Listen, "dev", cfg.Dev, "config", cfg.Path}
	if cfg.Dev {
		attrs = append(attrs, "preview", "http://"+displayHost(cfg.Listen)+"/dev/preview")
		if !cfg.Now.IsZero() {
			attrs = append(attrs, "advent_now", cfg.Now.Format("2006-01-02 15:04 MST"))
		}
	}
	log.Info("sandyadvent läuft", attrs...)
	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		log.Info("fahre herunter")
		shCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return hs.Shutdown(shCtx)
	}
}

func displayHost(listen string) string {
	if strings.HasPrefix(listen, ":") {
		return "localhost" + listen
	}
	return listen
}

func cmdCheck(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "", "Pfad zur config.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	log := slog.New(slog.NewTextHandler(stderr, nil))
	cfg, reg, err := loadAll(*cfgPath, log)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "OK: %s, %d Tokens, %d Türchen, Assets: %s\n", cfg.Path, len(cfg.Tokens), len(reg.Doors), cfg.AssetsDir)
	return nil
}

func cmdGenTokens(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gen-tokens", flag.ContinueOnError)
	fs.SetOutput(stderr)
	cfgPath := fs.String("config", "", "Pfad zur config.json (für -write)")
	write := fs.Bool("write", false, "Tokens in die config.json schreiben (vorhandene bleiben erhalten, EXAMPLE-Tokens werden ersetzt)")
	force := fs.Bool("force", false, "mit -write: ALLE Tokens neu erzeugen (alte QR-Codes werden ungültig!)")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "sandyadvent gen-tokens [-write [-force]] [-config config.json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*write {
		if *force {
			return errors.New("-force nur zusammen mit -write")
		}
		m, err := tokens.Generate(nil)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "%s\n", tokens.MarshalMap(m, ""))
		fmt.Fprintln(stderr, "Hinweis: mit -write direkt in die config.json übernehmen.")
		return nil
	}
	path := *cfgPath
	if path == "" {
		path = os.Getenv("ADVENT_CONFIG")
	}
	if path == "" {
		path = config.DefaultConfigPath
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("%s nicht gefunden – zuerst 'cp config.example.json %s'", path, path)
	}
	added, err := tokens.WriteConfig(path, nil, *force)
	if err != nil {
		return err
	}
	if len(added) == 0 {
		fmt.Fprintf(stdout, "%s: alle 17 QR-Tage haben bereits Tokens – nichts geändert.\n", path)
		return nil
	}
	fmt.Fprintf(stdout, "%s: neue Tokens für Tag %v geschrieben.\n", path, added)
	return nil
}
