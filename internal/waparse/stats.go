package waparse

import (
	_ "embed"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Options control Compute.
type Options struct {
	Aliases      map[string]string // export name -> display name
	Since        Date              // relationship start ("Tage zusammen"); zero = first message
	Until        Date              // ignore messages after this day; zero = all
	AsOf         Date              // "Tage zusammen" counted up to this day; zero = today
	Loc          *time.Location    // zone of the export (default Europe/Berlin)
	GoodNight    *regexp.Regexp    // default DefaultGoodNight
	DayStartHour int               // messages before this hour belong to the previous "night" (default 5)
	Top          int               // length of top lists (default 10)
	KeepLaughter bool              // keep "haha", "hihi" … in the top words
	Now          func() time.Time  // default time.Now
	MaxGapHours  int               // response times above this are ignored (default 12)
	FirstTextMax int               // firstMessage.text is cut to this many characters (default 140)
}

// DefaultGoodNight matches common German good-night wishes as whole words.
var DefaultGoodNight = regexp.MustCompile(`(?i)(?:^|[^\p{L}\p{N}])(?:gute?\s*nacht|gn8|gn|nachti+|n8i?|gute?s\s*nächtle|nächtle|schlaf\s+(?:gut|schön|süß)|träum\s+(?:was\s+)?(?:süß|schön))(?:[^\p{L}\p{N}]|$)`)

var (
	urlRE   = regexp.MustCompile(`(?i)(?:https?://|www\.)\S+`)
	laughRE = regexp.MustCompile(`^(?:a?(?:h+[aeiou]+){2,}h*|(?:hi){2,}|(?:he){2,}|lo+l|xd+|rofl|lmao)$`)
)

//go:embed stopwords_de.txt
var stopwordsTxt string

// stopwords is the embedded German stopword list (lower case).
var stopwords = func() map[string]bool {
	m := map[string]bool{}
	for _, line := range strings.Split(stopwordsTxt, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		for _, w := range strings.Fields(line) {
			m[w] = true
		}
	}
	return m
}()

var weekdaysDE = [7]string{"Montag", "Dienstag", "Mittwoch", "Donnerstag", "Freitag", "Samstag", "Sonntag"}

// Compute aggregates the messages into Stats.
func Compute(msgs []Message, o Options) *Stats {
	if o.Loc == nil {
		o.Loc = mustBerlin()
	}
	if o.GoodNight == nil {
		o.GoodNight = DefaultGoodNight
	}
	if o.DayStartHour == 0 {
		o.DayStartHour = 5
	}
	if o.Top <= 0 {
		o.Top = 10
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	if o.MaxGapHours <= 0 {
		o.MaxGapHours = 12
	}
	if o.FirstTextMax <= 0 {
		o.FirstTextMax = 140
	}
	now := o.Now().In(o.Loc)
	msgs = prepare(msgs, o)

	st := &Stats{Version: StatsVersion, GeneratedAt: now.Format(time.RFC3339)}
	st.TotalMessages = len(msgs)
	if len(msgs) == 0 {
		return st
	}
	st.Range = DateRange{From: msgs[0].Date.String(), To: msgs[len(msgs)-1].Date.String()}
	since := o.Since
	if since.IsZero() {
		since = msgs[0].Date
	}
	asOf := o.AsOf
	if asOf.IsZero() {
		asOf = DateOf(now)
	}
	st.Since, st.AsOf = since.String(), asOf.String()
	st.DaysTogether = max(0, DaysBetween(since, asOf))

	// names used for the word filter
	nameWords := map[string]bool{}
	addName := func(n string) {
		for _, w := range tokenize(n) {
			nameWords[w.lower] = true
		}
	}
	for k, v := range o.Aliases {
		addName(k)
		addName(v)
	}

	type person struct {
		*Person
		emojis   *counter
		respSum  float64
		respN    int
		firstIdx int
	}
	persons := map[string]*person{}
	get := func(name string, idx int) *person {
		p := persons[name]
		if p == nil {
			p = &person{Person: &Person{Name: name}, emojis: newCounter(), firstIdx: idx}
			persons[name] = p
			addName(name)
		}
		return p
	}
	emojis := newCounter()
	words := newCounter()
	var hours [24]int
	var weekdays [7]int
	perDay := map[Date]int{}
	goodNight := map[Date]string{} // night -> first author
	var prev *Message
	for i := range msgs {
		m := &msgs[i]
		p := get(m.Author, i)
		p.Messages++
		hours[m.Hour]++
		weekdays[(int(m.Date.Weekday())+6)%7]++
		perDay[m.Date]++
		if m.Placeholder {
			p.Media++
		}
		if prev != nil && prev.Author != m.Author {
			gap := m.Time.Sub(prev.Time)
			if gap >= 0 && gap <= time.Duration(o.MaxGapHours)*time.Hour {
				p.respSum += gap.Minutes()
				p.respN++
			}
		}
		prev = m
		if m.Text == "" {
			continue
		}
		if st.FirstMessage == nil {
			st.FirstMessage = &FirstMessage{Date: m.Date.String(), Time: m.Time.Format("15:04"), Name: m.Author, Text: truncate(m.Text, o.FirstTextMax)}
		}
		for _, e := range Emojis(m.Text) {
			k, d := EmojiKey(e), EmojiDisplay(e)
			emojis.add(k, d)
			p.emojis.add(k, d)
		}
		text := urlRE.ReplaceAllString(m.Text, " ")
		for _, f := range strings.Fields(text) {
			if strings.IndexFunc(f, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) >= 0 {
				p.Words++
			}
		}
		for _, w := range tokenize(text) {
			if utf8.RuneCountInString(w.lower) < 3 || stopwords[w.lower] || nameWords[w.lower] {
				continue
			}
			if !o.KeepLaughter && laughRE.MatchString(w.lower) {
				continue
			}
			words.add(w.lower, w.orig)
		}
		if o.GoodNight.MatchString(m.Text) {
			night := m.Date
			if m.Hour < o.DayStartHour {
				night = night.AddDays(-1)
			}
			if _, done := goodNight[night]; !done {
				goodNight[night] = m.Author
			}
		}
	}

	// persons, most messages first
	var list []*person
	for _, p := range persons {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Messages != list[j].Messages {
			return list[i].Messages > list[j].Messages
		}
		return list[i].firstIdx < list[j].firstIdx
	})
	for _, p := range list {
		p.Share = round(float64(p.Messages)/float64(len(msgs)), 3)
		if p.respN > 0 {
			v := round(p.respSum/float64(p.respN), 1)
			p.AvgResponseMinutes = &v
		}
		if top := p.emojis.top(1); len(top) > 0 {
			p.TopEmoji = top[0].display
		}
		st.Persons = append(st.Persons, *p.Person)
	}

	for _, e := range emojis.top(o.Top) {
		st.TopEmojis = append(st.TopEmojis, EmojiCount{Emoji: e.display, Count: e.n})
	}
	for _, w := range words.top(o.Top) {
		st.TopWords = append(st.TopWords, WordCount{Word: w.display, Count: w.n})
	}

	gnCount := map[string]int{}
	for _, a := range goodNight {
		gnCount[a]++
	}
	for _, p := range list {
		st.GoodNightFirst = append(st.GoodNightFirst, NameDays{Name: p.Name, Days: gnCount[p.Name]})
	}
	sort.SliceStable(st.GoodNightFirst, func(i, j int) bool { return st.GoodNightFirst[i].Days > st.GoodNightFirst[j].Days })
	st.GoodNightNights = len(goodNight)

	st.MostActiveHour = HourStat{Histogram: hours[:]}
	for h, n := range hours {
		if n > st.MostActiveHour.Count {
			st.MostActiveHour.Hour, st.MostActiveHour.Count = h, n
		}
	}
	st.MostActiveWeekday = WeekdayStat{Histogram: weekdays[:]}
	for d, n := range weekdays {
		if n > st.MostActiveWeekday.Count {
			st.MostActiveWeekday.Index, st.MostActiveWeekday.Count = d, n
		}
	}
	st.MostActiveWeekday.Weekday = weekdaysDE[st.MostActiveWeekday.Index]

	days := make([]Date, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })
	st.ActiveDays = len(days)
	run, runStart := 0, Date{}
	for i, d := range days {
		if i > 0 && DaysBetween(days[i-1], d) == 1 {
			run++
		} else {
			run, runStart = 1, d
		}
		if run > st.LongestStreak.Days {
			st.LongestStreak = Streak{Days: run, From: runStart.String(), To: d.String()}
		}
		if n := perDay[d]; n > st.BusiestDay.Count {
			st.BusiestDay = DayCount{Date: d.String(), Count: n}
		}
	}
	return st
}

// prepare applies aliases and the -until filter, sorts by time (stable, so
// equal minutes keep file order) and removes duplicates from overlapping
// exports.
func prepare(in []Message, o Options) []Message {
	out := make([]Message, 0, len(in))
	type key struct {
		t      int64
		author string
		text   string
		ph     bool
	}
	seen := map[key]int{}
	for _, m := range in {
		if !o.Until.IsZero() && o.Until.Before(m.Date) {
			continue
		}
		if a, ok := o.Aliases[m.Author]; ok {
			m.Author = a
		}
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	// Identical messages at the same minute can legitimately repeat ("😘" "😘"),
	// so per source file the count is kept: a duplicate is only dropped when
	// another file already contributed that many copies.
	res := out[:0]
	perFile := map[string]map[key]int{}
	for _, m := range out {
		k := key{m.Time.Unix(), m.Author, m.Text, m.Placeholder}
		file := m.Source
		if i := strings.LastIndexByte(file, ':'); i >= 0 {
			file = file[:i]
		}
		if perFile[file] == nil {
			perFile[file] = map[key]int{}
		}
		perFile[file][k]++
		if perFile[file][k] <= seen[k] {
			continue
		}
		seen[k] = perFile[file][k]
		res = append(res, m)
	}
	return res
}

type word struct{ lower, orig string }

// tokenize splits into letter runs (umlauts and ß included).
func tokenize(s string) []word {
	var out []word
	for _, f := range strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) }) {
		out = append(out, word{strings.ToLower(f), f})
	}
	return out
}

func truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func round(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}

// counter counts keys and remembers the most frequent display form.
type counter struct {
	n     map[string]int
	forms map[string]map[string]int
	order map[string]int
}

type counted struct {
	key, display string
	n            int
	first        int
}

// preferForm breaks ties between equally frequent display forms.
func preferForm(a, b string) bool {
	av, bv := strings.ContainsRune(a, 0xFE0F), strings.ContainsRune(b, 0xFE0F)
	if av != bv {
		return av
	}
	return a < b
}

func newCounter() *counter {
	return &counter{n: map[string]int{}, forms: map[string]map[string]int{}, order: map[string]int{}}
}

func (c *counter) add(key, form string) {
	if _, ok := c.order[key]; !ok {
		c.order[key] = len(c.order)
		c.forms[key] = map[string]int{}
	}
	c.n[key]++
	c.forms[key][form]++
}

// top returns the n most frequent keys (ties: first seen first). The display
// form is the most frequent spelling; on a tie a form with U+FE0F (emoji
// presentation) wins over the bare text form.
func (c *counter) top(n int) []counted {
	out := make([]counted, 0, len(c.n))
	for k, v := range c.n {
		best, bn := "", -1
		for f, fn := range c.forms[k] {
			if fn > bn || (fn == bn && preferForm(f, best)) {
				best, bn = f, fn
			}
		}
		out = append(out, counted{key: k, display: best, n: v, first: c.order[k]})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].n != out[j].n {
			return out[i].n > out[j].n
		}
		return out[i].first < out[j].first
	})
	if len(out) > n {
		out = out[:n]
	}
	return out
}
