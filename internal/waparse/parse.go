// Package waparse reads WhatsApp chat exports (text, without media, German
// locale; Android and iOS format) and computes the statistics shown on the
// "Schatzo Wrapped" door (template "wrapped", door 22).
//
// Raw exports never leave the machine they are parsed on; only the
// aggregated stats.json is copied to the calendar's assets.
package waparse

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Message is one chat message.
type Message struct {
	Time        time.Time // wall clock of the export in the chosen zone, truncated to the minute
	Date        Date      // calendar day as written in the export
	Hour        int       // hour as written in the export (0–23)
	Author      string    // name as in the export (before aliases)
	Text        string    // cleaned text ("" for placeholders)
	Placeholder bool      // media / deleted / call placeholder
	Source      string    // file:line
}

// Date is a calendar day.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// ParseDate parses YYYY-MM-DD.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("Datum %q: erwartet JJJJ-MM-TT", s)
	}
	return DateOf(t), nil
}

// DateOf returns the calendar day of t in t's location.
func DateOf(t time.Time) Date { y, m, d := t.Date(); return Date{y, m, d} }

func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day) }

// IsZero reports whether d is unset.
func (d Date) IsZero() bool { return d.Year == 0 }

func (d Date) num() int64 {
	return time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC).Unix() / 86400
}

// DaysBetween returns b - a in calendar days (DST-safe).
func DaysBetween(a, b Date) int { return int(b.num() - a.num()) }

// AddDays returns d + n days.
func (d Date) AddDays(n int) Date {
	return DateOf(time.Date(d.Year, d.Month, d.Day+n, 12, 0, 0, 0, time.UTC))
}

// Before reports whether d is before o.
func (d Date) Before(o Date) bool { return d.num() < o.num() }

// Weekday of the date.
func (d Date) Weekday() time.Weekday {
	return time.Date(d.Year, d.Month, d.Day, 12, 0, 0, 0, time.UTC).Weekday()
}

var (
	// 24.12.23, 18:05 - Name: Text   (also 24.12.2023; seconds tolerated)
	androidHeader = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.(\d{2}|\d{4}),? (\d{1,2}):(\d{2})(?::(\d{2}))? - (.*)$`)
	// [24.12.23, 18:05:12] Name: Text
	iosHeader = regexp.MustCompile(`^\[(\d{1,2})\.(\d{1,2})\.(\d{2}|\d{4}),? (\d{1,2}):(\d{2})(?::(\d{2}))?\] (.*)$`)
	// invisible direction/format marks WhatsApp sprinkles into exports
	marks = strings.NewReplacer("\u200e", "", "\u200f", "", "\u202a", "", "\u202b", "", "\u202c", "",
		"\u202d", "", "\u202e", "", "\u2066", "", "\u2067", "", "\u2068", "", "\u2069", "", "\ufeff", "")
	spaces = strings.NewReplacer("\u202f", " ", "\u00a0", " ", "\u2007", " ")
)

// maxAuthorLen guards against treating a system sentence with ": " as author.
const maxAuthorLen = 60

// Parser collects messages from one or more exports.
type Parser struct {
	Loc      *time.Location
	Messages []Message
	Skipped  int // system lines dropped

	// Authors, if non-empty, lists the export names of the chat partners.
	// A header-like line whose name is not in it (e.g. a pasted
	// "25.12.23, 18:05 - Treffpunkt: Bahnhof" inside a multi-line message)
	// is then treated as continuation text of the previous message.
	Authors map[string]bool
	// Folded counts such lines per name (for the CLI's report).
	Folded map[string]int
}

// NewParser returns a parser for the given zone (nil = Europe/Berlin).
func NewParser(loc *time.Location) *Parser {
	if loc == nil {
		loc = mustBerlin()
	}
	return &Parser{Loc: loc}
}

// Parse reads one export. Lines that do not start a message continue the
// previous one (multi-line messages). CRLF, lone CR and a BOM are handled.
func (p *Parser) Parse(name string, r io.Reader) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	raw = bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	raw = bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
	raw = bytes.ReplaceAll(raw, []byte("\r"), []byte("\n"))
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var cur *Message
	var curLines []string
	var curMarked bool
	flush := func() {
		if cur == nil {
			return
		}
		p.finish(cur, strings.Join(curLines, "\n"), curMarked)
		cur, curLines = nil, nil
	}
	line := 0
	found := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		head := spaces.Replace(marks.Replace(text))
		m := androidHeader.FindStringSubmatch(head)
		if m == nil {
			m = iosHeader.FindStringSubmatch(head)
		}
		var t time.Time
		var d Date
		var hour int
		ok := m != nil
		if ok {
			t, d, hour, ok = p.stamp(m)
		}
		if !ok {
			if cur != nil {
				curLines = append(curLines, text)
			}
			continue
		}
		found++
		rest := m[7]
		author, body, hasAuthor := splitAuthor(rest)
		if hasAuthor && len(p.Authors) > 0 && !p.Authors[author] {
			if p.Folded == nil {
				p.Folded = map[string]int{}
			}
			p.Folded[author]++
			if cur != nil {
				curLines = append(curLines, text)
			}
			continue
		}
		flush()
		if !hasAuthor || isSystem(rest, false) {
			p.Skipped++ // Android system line ("Nachrichten und Anrufe sind …")
			continue
		}
		cur = &Message{Time: t, Date: d, Hour: hour, Author: author, Source: fmt.Sprintf("%s:%d", name, line)}
		curLines = []string{body}
		curMarked = bodyMarked(text)
	}
	flush()
	if err := sc.Err(); err != nil {
		return err
	}
	if found == 0 {
		return fmt.Errorf("%s: keine WhatsApp-Nachrichten erkannt – erwartet z. B. „24.12.23, 18:05 - Name: Text“ (Android) oder „[24.12.23, 18:05:12] Name: Text“ (iOS), deutsches Datumsformat", name)
	}
	return nil
}

// bodyMarked reports whether the raw message body (after "Name: ") starts
// with U+200E – iOS marks its notices and placeholders that way.
func bodyMarked(raw string) bool {
	i := strings.Index(spaces.Replace(raw), ": ")
	if i < 0 {
		return false
	}
	return strings.HasPrefix(spaces.Replace(raw)[i+2:], "\u200e")
}

// finish cleans the text and appends the message (or drops system notices).
func (p *Parser) finish(m *Message, text string, marked bool) {
	clean := strings.TrimSpace(editedSuffix.ReplaceAllString(marks.Replace(text), ""))
	if isSystem(clean, marked) {
		p.Skipped++
		return
	}
	if isPlaceholder(clean) {
		m.Placeholder = true
		clean = ""
	}
	m.Text = clean
	p.Messages = append(p.Messages, *m)
}

// stamp converts the header fields. Date and hour are kept as written (the
// calendar day the couple saw); the instant is only used for durations.
func (p *Parser) stamp(m []string) (time.Time, Date, int, bool) {
	day, _ := strconv.Atoi(m[1])
	mon, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])
	hour, _ := strconv.Atoi(m[4])
	min, _ := strconv.Atoi(m[5])
	if len(m[3]) == 2 {
		year += 2000
	}
	if mon < 1 || mon > 12 || day < 1 || day > 31 || hour > 23 || min > 59 {
		return time.Time{}, Date{}, 0, false
	}
	// reject impossible dates (31.02.) – they are continuation text then
	if c := time.Date(year, time.Month(mon), day, 12, 0, 0, 0, time.UTC); c.Day() != day {
		return time.Time{}, Date{}, 0, false
	}
	return time.Date(year, time.Month(mon), day, hour, min, 0, 0, p.Loc), Date{year, time.Month(mon), day}, hour, true
}

// splitAuthor splits "Name: Text". Lines without an author are system lines.
func splitAuthor(rest string) (author, text string, ok bool) {
	i := strings.Index(rest, ": ")
	if i < 0 {
		if strings.HasSuffix(rest, ":") && len([]rune(rest)) <= maxAuthorLen+1 {
			return strings.TrimSpace(strings.TrimSuffix(rest, ":")), "", true
		}
		return "", "", false
	}
	author = strings.TrimSpace(rest[:i])
	if author == "" || len([]rune(author)) > maxAuthorLen {
		return "", "", false
	}
	return author, rest[i+2:], true
}

func mustBerlin() *time.Location {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		panic(err)
	}
	return loc
}
