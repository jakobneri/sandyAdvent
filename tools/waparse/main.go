// Command waparse turns WhatsApp chat exports (.txt without media, German
// locale, Android or iOS) into stats.json for the "Schatzo Wrapped" door.
//
//	go run ./tools/waparse -in private/chat.txt [-in private/chat2.txt] -out doors/assets/stats.json \
//	    -alias 'Max Mustermann=Jakob' -alias 'Erika 🌸=Sandy' -since 2023-02-14
//
// The raw export stays on the machine it is parsed on and must never be
// committed (see .gitignore); only the aggregated stats.json is copied.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/jakobneri/sandyadvent/internal/waparse"
)

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer, now func() time.Time) error {
	fs := flag.NewFlagSet("waparse", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var ins, aliases multi
	fs.Var(&ins, "in", "WhatsApp-Export (.txt), mehrfach möglich; weitere Dateien auch ohne -in")
	out := fs.String("out", "", "Ausgabe stats.json (leer = stdout)")
	fs.Var(&aliases, "alias", "'Name im Export=Anzeigename', mehrfach möglich (z. B. 'Erika 🌸=Sandy')")
	since := fs.String("since", "", "Beginn der Beziehung JJJJ-MM-TT für „Tage zusammen“ (Standard: erste Nachricht)")
	until := fs.String("until", "", "Nachrichten nach diesem Tag ignorieren (JJJJ-MM-TT)")
	asOf := fs.String("asof", "", "„Tage zusammen“ bis zu diesem Tag zählen (Standard: heute; das Türchen rechnet beim Anzeigen ohnehin neu)")
	tz := fs.String("tz", "Europe/Berlin", "Zeitzone des Exports")
	gn := fs.String("goodnight-pattern", "", "eigener regulärer Ausdruck für Gute-Nacht-Nachrichten (Go-Syntax, z. B. '(?i)\\bgute nacht\\b')")
	top := fs.Int("top", 10, "Länge der Top-Listen (Emojis, Wörter)")
	keepLaugh := fs.Bool("keep-laughter", false, "„haha“, „hihi“ … in den Top-Wörtern behalten")
	dayStart := fs.Int("day-start-hour", 5, "Gute-Nacht-Nachrichten vor dieser Uhrzeit zählen zur Nacht davor")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "waparse -in <export.txt> [-in …] -out stats.json -alias 'Export-Name=Anzeigename' [-since JJJJ-MM-TT]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	ins = append(ins, fs.Args()...)
	if len(ins) == 0 {
		fs.Usage()
		return errors.New("mindestens eine Export-Datei mit -in angeben")
	}
	loc, err := time.LoadLocation(*tz)
	if err != nil {
		return fmt.Errorf("-tz: %w", err)
	}
	o := waparse.Options{Aliases: map[string]string{}, Loc: loc, Top: *top, KeepLaughter: *keepLaugh, DayStartHour: *dayStart, Now: now}
	if *dayStart < 0 || *dayStart > 12 {
		return errors.New("-day-start-hour muss zwischen 0 und 12 liegen")
	}
	for _, a := range aliases {
		k, v, ok := strings.Cut(a, "=")
		if !ok || strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			return fmt.Errorf("-alias %q: Format 'Name im Export=Anzeigename'", a)
		}
		o.Aliases[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	for flagName, p := range map[string]struct {
		v   string
		dst *waparse.Date
	}{"since": {*since, &o.Since}, "until": {*until, &o.Until}, "asof": {*asOf, &o.AsOf}} {
		if p.v == "" {
			continue
		}
		d, err := waparse.ParseDate(p.v)
		if err != nil {
			return fmt.Errorf("-%s: %w", flagName, err)
		}
		*p.dst = d
	}
	if *gn != "" {
		re, err := regexp.Compile(*gn)
		if err != nil {
			return fmt.Errorf("-goodnight-pattern: %w", err)
		}
		o.GoodNight = re
	}

	p := waparse.NewParser(loc)
	if len(o.Aliases) > 0 {
		// With aliases, only these names start a message; header-like lines
		// of other names (pasted chat snippets) stay continuation text.
		p.Authors = map[string]bool{}
		for k := range o.Aliases {
			p.Authors[k] = true
		}
	}
	for _, in := range ins {
		f, err := os.Open(in)
		if err != nil {
			return err
		}
		err = p.Parse(filepath.Base(in), f)
		f.Close()
		if err != nil {
			return err
		}
	}
	authors := map[string]int{}
	for _, m := range p.Messages {
		authors[m.Author]++
	}
	names := make([]string, 0, len(authors))
	for n := range authors {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return authors[names[i]] > authors[names[j]] })
	fmt.Fprintf(stderr, "%d Nachrichten gelesen (%d Systemzeilen übersprungen). Namen im Export:\n", len(p.Messages), p.Skipped)
	for _, n := range names {
		alias := o.Aliases[n]
		if alias == "" {
			alias = "(kein -alias)"
		} else {
			alias = "→ " + alias
		}
		fmt.Fprintf(stderr, "  %-30q %6d %s\n", n, authors[n], alias)
	}
	for k := range o.Aliases {
		if authors[k] == 0 {
			fmt.Fprintf(stderr, "Warnung: -alias %q passt zu keinem Namen im Export (Groß-/Kleinschreibung, Emojis im Namen?)\n", k)
		}
	}
	if len(p.Folded) > 0 {
		folded := make([]string, 0, len(p.Folded))
		for n := range p.Folded {
			folded = append(folded, n)
		}
		sort.Slice(folded, func(i, j int) bool {
			return p.Folded[folded[i]] > p.Folded[folded[j]] || (p.Folded[folded[i]] == p.Folded[folded[j]] && folded[i] < folded[j])
		})
		fmt.Fprintln(stderr, "Zeilen mit Datum und Namen ohne -alias, als Fortsetzung der vorigen Nachricht gelesen:")
		for _, n := range folded {
			fmt.Fprintf(stderr, "  %-30q %6d\n", n, p.Folded[n])
		}
	}
	persons := map[string]bool{}
	for _, n := range names {
		if a := o.Aliases[n]; a != "" {
			persons[a] = true
		} else {
			persons[n] = true
		}
	}
	switch {
	case len(persons) > 2:
		return fmt.Errorf("%d Personen im Export (%s) – Türchen 22 ist für zwei Personen gedacht. "+
			"Für jeden echten Export-Namen ein -alias angeben (mehrere Namen dürfen auf denselben Anzeigenamen zeigen); "+
			"Zeilen anderer Namen werden dann als Fortsetzungstext gelesen", len(persons), quoteList(names))
	case len(persons) < 2 && len(o.Aliases) > 0:
		return fmt.Errorf("nur %d Person(en) erkannt – fehlt ein -alias? Jeder Export-Name braucht ein -alias, auch wenn er gleich bleibt (z. B. 'Jakob=Jakob'); oben stehen die Namen, die als Fortsetzung gelesen wurden", len(persons))
	}
	st := waparse.Compute(p.Messages, o)
	if st.TotalMessages == 0 {
		return errors.New("keine Nachrichten im gewählten Zeitraum")
	}
	if err := st.Validate(); err != nil {
		return fmt.Errorf("interner Fehler, stats.json ungültig: %w", err)
	}
	data, err := st.Marshal()
	if err != nil {
		return err
	}
	if *out == "" {
		_, err = stdout.Write(data)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "%s geschrieben: %d Nachrichten, %s bis %s, %d Tage zusammen (Stand %s)\n",
		*out, st.TotalMessages, st.Range.From, st.Range.To, st.DaysTogether, st.AsOf)
	return nil
}

func quoteList(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = fmt.Sprintf("%q", n)
	}
	return strings.Join(q, ", ")
}
