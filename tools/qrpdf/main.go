// Command qrpdf renders the 17 door QR codes (error correction H) onto A4
// pages: <base-url>/d/<token>, day number printed below each code.
//
//	go run ./tools/qrpdf -base-url http://<Pi-IP> [-config config.json] [-out out/qr-codes.pdf] [-size-mm 40] [-test]
//
// It refuses localhost base URLs and the EXAMPLE- placeholder tokens unless
// -test is given (test prints are marked TESTDRUCK in the header).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jakobneri/sandyadvent/internal/config"
	"github.com/jakobneri/sandyadvent/internal/qrpdf"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run() error {
	cfgPath := flag.String("config", "", "Pfad zur config.json (Standard: $ADVENT_CONFIG oder config.json)")
	baseURL := flag.String("base-url", "", "Basis-URL, z. B. http://<Pi-IP> (Standard: base_url aus der Config)")
	out := flag.String("out", "out/qr-codes.pdf", "Ausgabedatei")
	size := flag.Float64("size-mm", 40, fmt.Sprintf("Kantenlänge des QR-Codes ohne Ruhezone in mm (mind. %d)", qrpdf.MinSizeMM))
	cut := flag.Bool("cut-marks", true, "Schnittmarken zeichnen")
	testPrint := flag.Bool("test", false, "Testdruck: localhost-URL und Beispiel-Tokens erlauben (nicht zum Aufkleben!)")
	flag.Parse()

	if *size < qrpdf.MinSizeMM {
		return fmt.Errorf("-size-mm %g zu klein, mindestens %d mm", *size, qrpdf.MinSizeMM)
	}
	cfg, err := config.Load(*cfgPath, os.Getenv)
	if err != nil {
		return err
	}
	url := *baseURL
	if url == "" {
		url = cfg.BaseURL
	}
	codes, err := qrpdf.Codes(cfg.Tokens, url)
	if err != nil {
		return err
	}
	title := fmt.Sprintf("sandyAdvent - %d QR-Codes - %s - Fehlerkorrektur H - %g mm", len(codes), strings.TrimRight(url, "/"), *size)
	if err := qrpdf.CheckPrintable(cfg.Tokens, url); err != nil {
		if !*testPrint {
			return fmt.Errorf("%w\n(nur für einen Testdruck: -test bzw. make qr QR_TEST=1)", err)
		}
		fmt.Fprintf(os.Stderr, "WARNUNG – nur Testdruck, nicht aufkleben:\n%v\n", err)
		title = "TESTDRUCK - " + title
	}
	var buf bytes.Buffer
	if err := qrpdf.WritePDF(&buf, codes, qrpdf.Options{SizeMM: *size, CutMarks: *cut, Title: title}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o600); err != nil {
		return err
	}
	fmt.Printf("%s geschrieben (%d Codes, %s/d/…)\n", *out, len(codes), strings.TrimRight(url, "/"))
	return nil
}
