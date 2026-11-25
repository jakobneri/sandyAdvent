package waparse

import (
	"bytes"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
)

var (
	lrmT  = string(rune(0x200e))
	nbsp  = string(rune(0x00a0))
	nnbsp = string(rune(0x202f))
	zwj   = string(rune(0x200d))
	vs16  = string(rune(0xfe0f))
	bom   = "\xef\xbb\xbf"
)

func fixedNow() time.Time { return time.Date(2026, 11, 25, 10, 0, 0, 0, mustBerlin()) }

func parse(t *testing.T, files ...string) []Message {
	t.Helper()
	p := NewParser(nil)
	for i, f := range files {
		if err := p.Parse("f"+string(rune('0'+i)), strings.NewReader(f)); err != nil {
			t.Fatal(err)
		}
	}
	return p.Messages
}

func compute(msgs []Message, o Options) *Stats {
	if o.Now == nil {
		o.Now = fixedNow
	}
	return Compute(msgs, o)
}

// The committed example exports are exactly what the generator produces.
func TestTestdataUpToDate(t *testing.T) {
	msgs := Synthetic()
	for name, want := range map[string][]byte{"android.txt": RenderAndroid(msgs), "ios.txt": RenderIOS(msgs)} {
		got, err := os.ReadFile("../../tools/waparse/testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s veraltet – go run ./tools/waparse/gen", name)
		}
	}
	if len(msgs) < 800 {
		t.Errorf("nur %d synthetische Nachrichten", len(msgs))
	}
}

// Both formats of the same conversation give identical stats.
func TestAndroidAndIOSIdentical(t *testing.T) {
	read := func(name string) []Message {
		raw, err := os.ReadFile("../../tools/waparse/testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return parse(t, string(raw))
	}
	a, i := read("android.txt"), read("ios.txt")
	if len(a) != len(Synthetic()) || len(a) != len(i) {
		t.Fatalf("Android %d, iOS %d, erwartet %d", len(a), len(i), len(Synthetic()))
	}
	for k := range a {
		x, y := a[k], i[k]
		if x.Author != y.Author || x.Text != y.Text || x.Placeholder != y.Placeholder || !x.Time.Equal(y.Time) || x.Date != y.Date || x.Hour != y.Hour {
			t.Fatalf("Nachricht %d unterschiedlich:\n%+v\n%+v", k, x, y)
		}
	}
	o := Options{Aliases: map[string]string{SynthMe: "Jakob", SynthPartner: "Sandy"}, Since: Date{2024, 2, 14}, AsOf: Date{2026, 12, 22}}
	sa, si := compute(a, o), compute(i, o)
	ja, _ := sa.Marshal()
	ji, _ := si.Marshal()
	if !bytes.Equal(ja, ji) {
		t.Fatalf("Stats unterschiedlich:\n%s\n---\n%s", ja, ji)
	}
	if err := sa.Validate(); err != nil {
		t.Fatal(err)
	}
	// sanity of the synthetic conversation
	if sa.TotalMessages != len(a) || len(sa.Persons) != 2 || sa.Persons[0].Name != "Sandy" {
		t.Errorf("%+v", sa.Persons)
	}
	if sa.DaysTogether != DaysBetween(Date{2024, 2, 14}, Date{2026, 12, 22}) {
		t.Errorf("daysTogether %d", sa.DaysTogether)
	}
	if sa.Range.From != "2026-05-01" || sa.Range.To != "2026-11-01" && sa.Range.To != "2026-10-31" {
		t.Errorf("range %+v", sa.Range)
	}
	if sa.FirstMessage == nil || sa.FirstMessage.Name != "Jakob" || !strings.HasPrefix(sa.FirstMessage.Text, "Hallo Erika") {
		t.Errorf("firstMessage %+v", sa.FirstMessage)
	}
	if sa.Persons[0].Media == 0 || sa.GoodNightNights < 100 || sa.LongestStreak.Days < 10 {
		t.Errorf("media %d, goodnight %d, streak %+v", sa.Persons[0].Media, sa.GoodNightNights, sa.LongestStreak)
	}
	var zwjSeen bool
	for _, e := range sa.TopEmojis {
		zwjSeen = zwjSeen || strings.Contains(e.Emoji, zwj)
	}
	for _, w := range sa.TopWords {
		lw := strings.ToLower(w.Word)
		if lw == "erika" || lw == "max" || lw == "sandy" || lw == "jakob" || lw == "haha" || lw == "hahaha" || lw == "hihi" || stopwords[lw] {
			t.Errorf("Top-Wort %q hätte gefiltert werden müssen", w.Word)
		}
	}
	for _, m := range a {
		if strings.Contains(m.Text, "bearbeitet") || strings.Contains(m.Text, "verschlüsselt") || strings.Contains(m.Text, lrmT) {
			t.Fatalf("nicht bereinigt: %q", m.Text)
		}
	}
}

func TestYearsAndHeaders(t *testing.T) {
	msgs := parse(t, "24.12.23, 18:05 - Max: zwei\n24.12.2023, 18:06 - Max: vier\n"+
		lrmT+"[24.12.23,"+nbsp+"18:07:09] Max: "+lrmT+"Bild weggelassen\n"+
		"[24.12.2023, 18:08:59] Max: ios vier\n[24.12.23,"+nnbsp+"18:09:00] Max: schmal\n"+
		"31.02.23, 10:00 - Max: kein Datum\n")
	if len(msgs) != 5 {
		t.Fatalf("%d Nachrichten: %+v", len(msgs), msgs)
	}
	for k, m := range msgs {
		if m.Date != (Date{2023, 12, 24}) || m.Hour != 18 || m.Author != "Max" {
			t.Errorf("%d: %+v", k, m)
		}
	}
	if !msgs[2].Placeholder || msgs[2].Text != "" {
		t.Errorf("iOS-Medienzeile: %+v", msgs[2])
	}
	if msgs[4].Text != "schmal\n31.02.23, 10:00 - Max: kein Datum" {
		t.Errorf("ungültiges Datum muss Fortsetzung sein: %q", msgs[4].Text)
	}
	if msgs[3].Time.Second() != 0 {
		t.Error("Sekunden müssen abgeschnitten werden (gleiche Stats für Android und iOS)")
	}
}

func TestMultilineCRLFBOMAndSystem(t *testing.T) {
	android := bom + "01.03.26, 09:00 - Nachrichten und Anrufe sind Ende-zu-Ende-verschlüsselt. Tippe, um mehr zu erfahren.\r\n" +
		"01.03.26, 09:01 - Max: Einkaufsliste:\r\nMilch\r\n\r\nEier\r\n" +
		"01.03.26, 09:02 - Erika: Okay <Diese Nachricht wurde bearbeitet>\r\n" +
		"01.03.26, 09:03 - Erika: Diese Nachricht wurde gelöscht\r\n" +
		"01.03.26, 09:04 - Max: Du hast diese Nachricht gelöscht\r\n" +
		"01.03.26, 09:05 - Max: <Medien ausgelassen>\r\n" +
		"01.03.26, 09:06 - Erika hat die Sicherheitsnummer geändert\r\n" +
		"01.03.26, 09:07 - Erika: Was ist eine Sicherheitsnummer?\r\n"
	ios := "[01.03.26, 09:00:00] Erika: " + lrmT + "Nachrichten und Anrufe sind Ende-zu-Ende-verschlüsselt. Niemand außerhalb dieses Chats kann sie lesen.\r" +
		"[01.03.26, 09:01:10] Max: Einkaufsliste:\rMilch\r\rEier\r" +
		"[01.03.26, 09:02:20] Erika: Okay " + lrmT + "<Diese Nachricht wurde bearbeitet.>\r" +
		lrmT + "[01.03.26, 09:03:30] Erika: " + lrmT + "Diese Nachricht wurde gelöscht.\r" +
		lrmT + "[01.03.26, 09:04:40] Max: " + lrmT + "Du hast diese Nachricht gelöscht.\r" +
		lrmT + "[01.03.26, 09:05:50] Max: " + lrmT + "<Anhang: 00000012-PHOTO-2026-03-01-09-05-50.jpg>\r" +
		"[01.03.26, 09:07:00] Erika: Was ist eine Sicherheitsnummer?"
	for name, src := range map[string]string{"android": android, "ios": ios} {
		msgs := parse(t, src)
		if len(msgs) != 6 {
			t.Fatalf("%s: %d Nachrichten %+v", name, len(msgs), msgs)
		}
		if msgs[0].Text != "Einkaufsliste:\nMilch\n\nEier" {
			t.Errorf("%s mehrzeilig: %q", name, msgs[0].Text)
		}
		if msgs[1].Text != "Okay" {
			t.Errorf("%s bearbeitet: %q", name, msgs[1].Text)
		}
		for k := 2; k <= 4; k++ {
			if !msgs[k].Placeholder {
				t.Errorf("%s: %d kein Platzhalter: %+v", name, k, msgs[k])
			}
		}
		if msgs[5].Text != "Was ist eine Sicherheitsnummer?" {
			t.Errorf("%s: normale Nachricht verschluckt: %+v", name, msgs[5])
		}
		st := compute(msgs, Options{})
		if st.TotalMessages != 6 || st.Persons[0].Media+st.Persons[1].Media != 3 {
			t.Errorf("%s: %+v", name, st.Persons)
		}
	}
}

// Name-led notice patterns only apply to iOS bodies marked with U+200E.
func TestNoticesVsOrdinaryMessages(t *testing.T) {
	ios := "[25.12.23, 00:10:00] Erika: " + lrmT + "Erika hat zu einer neuen Nummer gewechselt.\n" +
		"[25.12.23, 00:11:00] Erika: " + lrmT + "Erika hat selbstlöschende Nachrichten aktiviert.\n" +
		"[25.12.23, 00:13:00] Jakob N.: Mein Bruder hat zu Vodafone gewechselt\n" +
		"[25.12.23, 00:14:00] Erika: Lisa verwendet jetzt eine andere Nummer, sagt sie\n" +
		"[25.12.23, 00:15:00] Erika: Der Laden ist jetzt ein Business-Konto auf Insta\n"
	android := "25.12.23, 00:10 - Erika hat zu einer neuen Nummer gewechselt. Tippe, um eine Nachricht zu schreiben.\n" +
		"25.12.23, 00:13 - Jakob N.: Mein Bruder hat zu Vodafone gewechselt\n" +
		"25.12.23, 00:14 - Erika: Lisa hat zu einer neuen Nummer gewechselt. Tippe, um sie zu speichern.\n"
	if got := len(parse(t, ios)); got != 3 {
		t.Errorf("ios: %d Nachrichten, erwartet 3", got)
	}
	msgs := parse(t, android)
	if len(msgs) != 1 || msgs[0].Text != "Mein Bruder hat zu Vodafone gewechselt" {
		t.Errorf("android: %+v", msgs)
	}
}

// With known authors, a pasted "Datum - Name: Text" line stays in the
// previous message instead of creating a third person.
func TestKnownAuthorsFoldPastedHeaders(t *testing.T) {
	src := "25.12.23, 18:00 - Max: Treffpunkt?\n" +
		"25.12.23, 18:04 - Erika: Siehe hier:\n" +
		"25.12.23, 18:05 - Treffpunkt: Bahnhof\n" +
		"25.12.23, 18:06 - Max: Danke\n"
	if n := len(parse(t, src)); n != 4 {
		t.Fatalf("ohne Authors: %d", n)
	}
	p := NewParser(nil)
	p.Authors = map[string]bool{"Max": true, "Erika": true}
	if err := p.Parse("x", strings.NewReader(src)); err != nil {
		t.Fatal(err)
	}
	if len(p.Messages) != 3 || p.Messages[1].Text != "Siehe hier:\n25.12.23, 18:05 - Treffpunkt: Bahnhof" || p.Folded["Treffpunkt"] != 1 {
		t.Errorf("%+v %v", p.Messages, p.Folded)
	}
}

func TestMidnightGoodNightAndStreak(t *testing.T) {
	src := "10.06.26, 21:00 - Max: Hallo\n" +
		"10.06.26, 23:59 - Erika: Gute Nacht 😘\n" + // night of 10.06., Erika first
		"11.06.26, 00:00 - Max: gn8\n" +
		"11.06.26, 00:30 - Max: Schlaf gut\n" +
		"11.06.26, 08:00 - Max: Guten Morgen\n" +
		"12.06.26, 01:30 - Max: Nachti 🌙\n" + // night of 11.06., Max first
		"12.06.26, 04:59 - Erika: gute nacht\n" +
		"12.06.26, 05:00 - Erika: gn\n" + // night of 12.06.
		"14.06.26, 22:00 - Erika: Gnocchi mit Gnade, keine Gnu-Nacht\n" + // no match
		"14.06.26, 22:01 - Erika: Guten Abend\n"
	msgs := parse(t, src)
	st := compute(msgs, Options{})
	want := []NameDays{{"Erika", 2}, {"Max", 1}}
	if !reflect.DeepEqual(st.GoodNightFirst, want) || st.GoodNightNights != 3 {
		t.Errorf("goodNightFirst %+v (%d Nächte)", st.GoodNightFirst, st.GoodNightNights)
	}
	if st.LongestStreak != (Streak{3, "2026-06-10", "2026-06-12"}) {
		t.Errorf("streak %+v", st.LongestStreak)
	}
	if h := st.MostActiveHour.Histogram; h[0] != 2 || h[23] != 1 || h[5] != 1 || h[4] != 1 {
		t.Errorf("hours %v", h)
	}
	// custom pattern
	st = compute(msgs, Options{GoodNight: regexp.MustCompile(`(?i)gnocchi`)})
	if st.GoodNightNights != 1 || st.GoodNightFirst[0].Name != "Erika" {
		t.Errorf("eigenes Muster: %+v", st.GoodNightFirst)
	}
	for _, s := range []string{"Gute Nacht!", "gn", "GN8", "nachti", "Nachtiii", "schlaf gut", "Schlaf schön 💤", "n8", "Guts Nächtle"} {
		if !DefaultGoodNight.MatchString(s) {
			t.Errorf("%q nicht erkannt", s)
		}
	}
	for _, s := range []string{"Gnade", "gnocchi", "Guten Abend", "Mitternacht", "nachtisch", "Signal"} {
		if DefaultGoodNight.MatchString(s) {
			t.Errorf("%q fälschlich erkannt", s)
		}
	}
}

func TestDST(t *testing.T) {
	// spring forward 29.03.2026 02:00 → 03:00, fall back 25.10.2026 03:00 → 02:00
	src := "28.03.26, 23:00 - A: x\n29.03.26, 02:30 - A: gibt es nicht\n29.03.26, 03:10 - B: y\n30.03.26, 09:00 - A: z\n" +
		"24.10.26, 23:50 - A: a\n25.10.26, 02:15 - B: erstes Mal\n25.10.26, 02:15 - B: zweites Mal\n25.10.26, 02:45 - A: b\n26.10.26, 10:00 - A: c\n"
	msgs := parse(t, src)
	if len(msgs) != 9 {
		t.Fatalf("%d", len(msgs))
	}
	if msgs[1].Hour != 2 || msgs[1].Date != (Date{2026, 3, 29}) {
		t.Errorf("Uhrzeit in der Zeitumstellungslücke: %+v", msgs[1])
	}
	st := compute(msgs, Options{})
	if st.LongestStreak.Days != 3 || st.ActiveDays != 6 {
		t.Errorf("streak %+v, aktive Tage %d", st.LongestStreak, st.ActiveDays)
	}
	if st.MostActiveHour.Hour != 2 || st.MostActiveHour.Count != 4 {
		t.Errorf("mostActiveHour %+v", st.MostActiveHour)
	}
	// both 02:15 messages kept (same minute, same text would be fine too)
	if st.TotalMessages != 9 {
		t.Errorf("total %d", st.TotalMessages)
	}
}

func TestAliasesMergeAndOverlap(t *testing.T) {
	a := "01.01.26, 10:00 - Erika Musterfrau: Hallo\n01.01.26, 10:01 - Max: Hi 😘\n01.01.26, 10:01 - Max: 😘\n"
	b := "01.01.26, 10:01 - Max: Hi 😘\n01.01.26, 10:01 - Max: 😘\n02.01.26, 11:00 - Erika 🌸: Neue Nummer\n"
	msgs := parse(t, a, b)
	st := compute(msgs, Options{Aliases: map[string]string{"Erika Musterfrau": "Sandy", "Erika 🌸": "Sandy", "Max": "Jakob"}})
	if st.TotalMessages != 4 {
		t.Fatalf("Duplikate aus überlappenden Exporten: %d", st.TotalMessages)
	}
	// equal counts: first author first
	if len(st.Persons) != 2 || st.Persons[0].Name != "Sandy" || st.Persons[0].Messages != 2 || st.Persons[1].Name != "Jakob" || st.Persons[1].Messages != 2 {
		t.Errorf("%+v", st.Persons)
	}
	if st.Persons[1].Share != 0.5 || st.Persons[1].TopEmoji != "😘" {
		t.Errorf("%+v", st.Persons[1])
	}
	// response time: Max answers Erika after 1 minute
	if r := st.Persons[1].AvgResponseMinutes; r == nil || *r != 1 {
		t.Errorf("avgResponseMinutes %v", r)
	}
	for _, w := range st.TopWords {
		if strings.EqualFold(w.Word, "Erika") || strings.EqualFold(w.Word, "Musterfrau") || strings.EqualFold(w.Word, "Sandy") {
			t.Errorf("Name in Top-Wörtern: %q", w.Word)
		}
	}
	// until
	st = compute(msgs, Options{Until: Date{2026, 1, 1}})
	if st.TotalMessages != 3 {
		t.Errorf("until: %d", st.TotalMessages)
	}
}

func TestEmojis(t *testing.T) {
	couple := "👩" + zwj + "❤" + vs16 + zwj + "👨"
	s := "Hey " + couple + " 👍🏽👍 🇩🇪🇮🇹 1" + vs16 + "⃣ ❤" + vs16 + " ❤ ✨ ©" + vs16 + " © 123 #tag " + "🧑" + zwj + "🎄 🫪"
	got := Emojis(s)
	want := []string{couple, "👍🏽", "👍", "🇩🇪", "🇮🇹", "1" + vs16 + string(rune(0x20e3)), "❤" + vs16, "❤", "✨", "©" + vs16, "🧑" + zwj + "🎄", "🫪"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Emojis:\n got %q\nwant %q", got, want)
	}
	if EmojiKey("❤"+vs16) != EmojiKey("❤") {
		t.Error("❤️ und ❤ müssen gleich zählen")
	}
	msgs := parse(t, "01.01.26, 10:00 - A: ❤️ ❤️ ❤ "+couple+"\n01.01.26, 10:01 - B: 👍🏽 👍🏽 👍\n")
	st := compute(msgs, Options{Top: 3})
	wantTop := []EmojiCount{{"❤️", 3}, {"👍🏽", 2}, {couple, 1}}
	if !reflect.DeepEqual(st.TopEmojis, wantTop) {
		t.Errorf("topEmojis %+v", st.TopEmojis)
	}
	// bare text heart ties with or beats ❤️: display stays the emoji form
	for _, src := range []string{"01.01.26, 10:00 - A: gn8 ❤️❤\n", "01.01.26, 10:00 - A: ❤ ❤ ❤️ ☺\n"} {
		st = compute(parse(t, src), Options{Top: 3})
		if st.TopEmojis[0].Emoji != "❤"+vs16 || st.Persons[0].TopEmoji != "❤"+vs16 {
			t.Errorf("%q: %q / %q", src, st.TopEmojis[0].Emoji, st.Persons[0].TopEmoji)
		}
		for _, e := range st.TopEmojis {
			if strings.HasPrefix(e.Emoji, "☺") && e.Emoji != "☺"+vs16 {
				t.Errorf("☺ ohne FE0F: %q", e.Emoji)
			}
		}
	}
	if EmojiDisplay("✨") != "✨" || EmojiDisplay("😘") != "😘" || EmojiDisplay("❤") != "❤"+vs16 {
		t.Error("EmojiDisplay")
	}
}

func TestTopWords(t *testing.T) {
	src := "01.01.26, 10:00 - Max: Pizza heute? https://example.com/pizza-rezept www.pizza.de 2026 haha hahaha hihi\n" +
		"01.01.26, 10:01 - Erika: Pizza! Pizza mit Max und Sandy 42 Kaffee kaffee lol\n" +
		"01.01.26, 10:02 - Erika: Kuscheldecke, ich und du\n"
	msgs := parse(t, src)
	st := compute(msgs, Options{Aliases: map[string]string{"Erika": "Sandy"}})
	want := []WordCount{{"Pizza", 3}, {"Kaffee", 2}, {"Kuscheldecke", 1}}
	if !reflect.DeepEqual(st.TopWords, want) {
		t.Errorf("topWords %+v", st.TopWords)
	}
	st = compute(msgs, Options{KeepLaughter: true})
	found := map[string]bool{}
	for _, w := range st.TopWords {
		found[w.Word] = true
	}
	if !found["haha"] || !found["hihi"] {
		t.Errorf("KeepLaughter: %+v", st.TopWords)
	}
	if st.Persons[0].Words == 0 {
		t.Error("words zählt nicht")
	}
}

func TestParseErrorsAndValidate(t *testing.T) {
	p := NewParser(nil)
	if err := p.Parse("leer.txt", strings.NewReader("Hallo\nWelt\n")); err == nil || !strings.Contains(err.Error(), "keine WhatsApp-Nachrichten") {
		t.Errorf("%v", err)
	}
	msgs := parse(t, "01.01.26, 10:00 - A: x\n")
	st := compute(msgs, Options{})
	raw, _ := st.Marshal()
	if _, err := ParseStats(raw); err != nil {
		t.Fatal(err)
	}
	bad := map[string]string{
		`"version": 1`:       `"version": 2`,
		`"totalMessages": 1`: `"totalMessages": 0`,
		`"since": "`:         `"since": "x`,
		`"version": 1,`:      `"version": 1, "extra": true,`,
	}
	for from, to := range bad {
		if _, err := ParseStats([]byte(strings.Replace(string(raw), from, to, 1))); err == nil {
			t.Errorf("%s → %s akzeptiert", from, to)
		}
	}
}
