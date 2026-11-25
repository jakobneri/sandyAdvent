package waparse

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// StatsVersion is the schema version of stats.json.
const StatsVersion = 1

// Stats is stats.json – the input of the "wrapped" door. Schema: README.md.
type Stats struct {
	Version           int           `json:"version"`
	GeneratedAt       string        `json:"generatedAt"`
	Range             DateRange     `json:"range"`
	Since             string        `json:"since"`
	AsOf              string        `json:"asOf"`
	DaysTogether      int           `json:"daysTogether"`
	TotalMessages     int           `json:"totalMessages"`
	ActiveDays        int           `json:"activeDays"`
	Persons           []Person      `json:"persons"`
	TopEmojis         []EmojiCount  `json:"topEmojis"`
	GoodNightFirst    []NameDays    `json:"goodNightFirst"`
	GoodNightNights   int           `json:"goodNightNights"`
	MostActiveHour    HourStat      `json:"mostActiveHour"`
	MostActiveWeekday WeekdayStat   `json:"mostActiveWeekday"`
	LongestStreak     Streak        `json:"longestStreak"`
	TopWords          []WordCount   `json:"topWords"`
	FirstMessage      *FirstMessage `json:"firstMessage,omitempty"`
	BusiestDay        DayCount      `json:"busiestDay"`
}

// DateRange is the first and last day with messages (YYYY-MM-DD).
type DateRange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Person are the per-person numbers.
type Person struct {
	Name               string   `json:"name"`
	Messages           int      `json:"messages"`
	Words              int      `json:"words"`
	Share              float64  `json:"share"`
	Media              int      `json:"media"`
	AvgResponseMinutes *float64 `json:"avgResponseMinutes,omitempty"`
	TopEmoji           string   `json:"topEmoji,omitempty"`
}

// EmojiCount is one entry of topEmojis.
type EmojiCount struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// WordCount is one entry of topWords.
type WordCount struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// NameDays counts days per person.
type NameDays struct {
	Name string `json:"name"`
	Days int    `json:"days"`
}

// HourStat is the busiest hour plus messages per hour 0–23.
type HourStat struct {
	Hour      int   `json:"hour"`
	Count     int   `json:"count"`
	Histogram []int `json:"histogram"`
}

// WeekdayStat is the busiest weekday plus messages per weekday (Mo … So).
type WeekdayStat struct {
	Weekday   string `json:"weekday"`
	Index     int    `json:"index"`
	Count     int    `json:"count"`
	Histogram []int  `json:"histogram"`
}

// Streak is the longest run of consecutive days with messages.
type Streak struct {
	Days int    `json:"days"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}

// DayCount is a day with its message count.
type DayCount struct {
	Date  string `json:"date,omitempty"`
	Count int    `json:"count"`
}

// FirstMessage is the earliest text message of the export.
type FirstMessage struct {
	Date string `json:"date"`
	Time string `json:"time"`
	Name string `json:"name"`
	Text string `json:"text"`
}

// Marshal renders stats.json (indented, no HTML escaping).
func (s *Stats) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// ParseStats decodes and validates stats.json (unknown fields are errors, so
// typos in hand edits show up).
func ParseStats(raw []byte) (*Stats, error) {
	var s Stats
	dec := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("stats.json: %w", err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

// Validate checks what the wrapped door relies on.
func (s *Stats) Validate() error {
	var errs []error
	add := func(format string, a ...any) { errs = append(errs, fmt.Errorf(format, a...)) }
	date := func(name, v string, required bool) {
		if v == "" {
			if required {
				add("%s fehlt", name)
			}
			return
		}
		if _, err := ParseDate(v); err != nil {
			add("%s: %v", name, err)
		}
	}
	if s.Version != StatsVersion {
		add("version muss %d sein (ist %d)", StatsVersion, s.Version)
	}
	if s.TotalMessages <= 0 {
		add("totalMessages muss > 0 sein")
	}
	date("range.from", s.Range.From, true)
	date("range.to", s.Range.To, true)
	date("since", s.Since, true)
	date("asOf", s.AsOf, false)
	date("longestStreak.from", s.LongestStreak.From, false)
	date("longestStreak.to", s.LongestStreak.To, false)
	date("busiestDay.date", s.BusiestDay.Date, false)
	if s.DaysTogether < 0 {
		add("daysTogether darf nicht negativ sein")
	}
	if len(s.Persons) == 0 {
		add("persons ist leer")
	}
	sum := 0
	for i, p := range s.Persons {
		if strings.TrimSpace(p.Name) == "" {
			add("persons[%d].name fehlt", i)
		}
		if p.Messages < 0 || p.Words < 0 || p.Media < 0 || p.Share < 0 || p.Share > 1 {
			add("persons[%d] (%s): ungültige Zahlen", i, p.Name)
		}
		sum += p.Messages
	}
	if len(s.Persons) > 0 && sum != s.TotalMessages {
		add("Summe persons[].messages (%d) ≠ totalMessages (%d)", sum, s.TotalMessages)
	}
	if h := s.MostActiveHour; len(h.Histogram) != 24 || h.Hour < 0 || h.Hour > 23 {
		add("mostActiveHour: hour 0–23 und histogram mit 24 Werten erwartet")
	}
	if w := s.MostActiveWeekday; len(w.Histogram) != 7 || w.Index < 0 || w.Index > 6 {
		add("mostActiveWeekday: index 0–6 (Montag = 0) und histogram mit 7 Werten erwartet")
	}
	for i, e := range s.TopEmojis {
		if e.Emoji == "" || e.Count <= 0 {
			add("topEmojis[%d] ungültig", i)
		}
	}
	for i, w := range s.TopWords {
		if strings.TrimSpace(w.Word) == "" || w.Count <= 0 {
			add("topWords[%d] ungültig", i)
		}
	}
	for i, g := range s.GoodNightFirst {
		if g.Name == "" || g.Days < 0 {
			add("goodNightFirst[%d] ungültig", i)
		}
	}
	if s.LongestStreak.Days < 0 {
		add("longestStreak.days darf nicht negativ sein")
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
