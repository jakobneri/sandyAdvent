// Command crossword turns a word list into Raster-JSON for the crossword
// door (template "crossword", door 16).
//
//	go run ./tools/crossword -in private/woerter.txt -out doors/assets/kreuzwort.json [-seed 42] [-attempts 300] [-max-size 13] [-preview] [-strict]
//
// Input: lines "WORT;Hinweis" (UTF-8, '#' comments, empty lines ok) or JSON
// [{"word": "…", "clue": "…"}]. Words are upper-cased (NFC); Ä, Ö, Ü and ß
// stay single cells. Spaces, digits and hyphens are refused (-strip-spaces
// joins multi-word answers). Unplaced words are listed on stderr; with
// -strict the exit code is then 2.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/jakobneri/sandyadvent/internal/crossword"
)

func main() {
	code, err := run(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		if code == 0 {
			code = 1
		}
	}
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) (int, error) {
	fs := flag.NewFlagSet("crossword", flag.ContinueOnError)
	fs.SetOutput(stderr)
	in := fs.String("in", "", "Wortliste (.txt/.csv mit WORT;Hinweis oder .json)")
	out := fs.String("out", "", "Ausgabe Raster-JSON (leer = stdout)")
	maxSize := fs.Int("max-size", crossword.DefaultMaxSize, "maximale Breite und Höhe (13 passt mit ≥ 24 px pro Feld auf 360 px)")
	seed := fs.Uint64("seed", 0, "Zufalls-Seed (0 = zufällig; der benutzte Seed wird ausgegeben)")
	attempts := fs.Int("attempts", 1000, "Anzahl zufälliger Versuche, der beste gewinnt")
	preview := fs.Bool("preview", false, "ASCII-Vorschau auf stderr")
	strict := fs.Bool("strict", false, "Exit-Code 2, wenn nicht alle Wörter platziert wurden")
	strip := fs.Bool("strip-spaces", false, "Leerzeichen in Wörtern entfernen statt abzulehnen")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	if *in == "" {
		fs.Usage()
		return 2, errors.New("-in fehlt")
	}
	raw, err := os.ReadFile(*in)
	if err != nil {
		return 1, err
	}
	words, err := crossword.ReadWords(*in, raw, crossword.ReadOptions{StripSpaces: *strip})
	if err != nil {
		return 1, fmt.Errorf("%s:\n%w", *in, err)
	}
	s := *seed
	if s == 0 {
		s = uint64(time.Now().UnixNano())
	}
	res, err := crossword.Generate(words, crossword.Options{MaxSize: *maxSize, Seed: s, Attempts: *attempts})
	if err != nil {
		return 1, err
	}
	data, err := crossword.Marshal(res.Raster)
	if err != nil {
		return 1, err
	}
	if *out == "" {
		_, err = stdout.Write(data)
	} else {
		if dir := filepath.Dir(*out); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return 1, err
			}
		}
		err = os.WriteFile(*out, data, 0o644)
	}
	if err != nil {
		return 1, err
	}
	r := res.Raster
	fmt.Fprintf(stderr, "%d von %d Wörtern platziert, Raster %d×%d, Seed %d (Versuch %d)\n",
		len(words)-len(res.Unplaced), len(words), r.Width, r.Height, res.Seed, res.Attempt+1)
	if *preview {
		fmt.Fprint(stderr, "\n"+r.Preview()+"\n")
	}
	if len(res.Unplaced) > 0 {
		fmt.Fprintln(stderr, "Nicht platziert (anderen Seed, mehr -attempts, größeres -max-size oder Wort tauschen):")
		for _, w := range res.Unplaced {
			fmt.Fprintf(stderr, "  %s;%s\n", w.Word, w.Clue)
		}
		if *strict {
			return 2, fmt.Errorf("%d Wörter nicht platziert", len(res.Unplaced))
		}
	}
	return 0, nil
}
