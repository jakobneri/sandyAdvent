package waparse

import (
	"bytes"
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
)

// Synthetic test data: one made-up conversation between two fake people,
// rendered as an Android and an iOS export (German locale). Used by the tests
// and by `go run ./tools/waparse/gen` (tools/waparse/testdata/*.txt).
// Everything here is invented – no real chat content.

// Names of the synthetic chat partners as they appear in the export.
const (
	SynthMe      = "Max Mustermann" // the person who exported the chat
	SynthPartner = "Erika 🌸"
)

// Synthetic message kinds.
const (
	SynthText = iota
	SynthMedia
	SynthDeleted
	SynthEdited
)

// SynthMsg is one message of the synthetic conversation.
type SynthMsg struct {
	T      time.Time // wall clock with seconds (Europe/Berlin)
	Author string
	Kind   int
	Text   string
	Media  string // iOS placeholder ("Bild weggelassen" …)
}

var (
	lrm       = string(rune(0x200e))
	synthFrom = Date{2026, time.May, 1}
	synthTo   = Date{2026, time.October, 31}
)

var (
	synthMorning = []string{"Guten Morgen ☀️", "Morgen Schatzo 😘", "Gut geschlafen? 🥰", "Kaffee ist durch ☕", "Guten Morgen, Sonnenschein 🌻"}
	synthDay     = []string{
		"Wie läuft's bei dir?", "Denk an dich ❤️", "Pizza heute Abend? 🍕", "Hab grad an unseren Urlaub gedacht 🇮🇹",
		"Haha 😂😂", "hahaha", "Schau mal: https://example.com/rezept-plaetzchen", "Ich bring Plätzchen mit 🍪",
		"Kino am Freitag? 🎬", "Du bist die Beste 👩‍❤️‍👨", "Bin gleich da 🚗", "Sooo müde 😴", "Daumen hoch 👍🏽",
		"Winke winke 👋🏻", "Fertig mit der Schicht 💪", "Wichtel-Idee 🧑‍🎄", "Top 3: 1️⃣ Pizza 2️⃣ Kino 3️⃣ Du",
		"Kuscheldecke und Tee heute?", "Wir brauchen noch Milch 🥛", "Noch 17 Minuten", "Schatzo ❤️❤️",
		"Wann bist du da?", "Pizza oder Pasta? 🍕🍝", "Kuscheldecke, Tee und Serie 🥰", "Ich vermisse dich 🥺",
		"Plätzchen sind fertig 🍪🍪", "Lieblingsmensch 💛", "Daheim! 🏠", "Kino war super 🎬😍", "Deutschland spielt 🇩🇪⚽",
	}
	synthReply = []string{"Haha ja 😂", "Gerne ❤️", "Oh ja 😍", "Klingt gut 👍", "Bin dabei!", "Schatzo 🥰", "Hihi", "Okay 😘", "Pizza! 🍕", "Na klar 💛"}
	synthNight = []string{"Gute Nacht Schatzo 😘", "Nachti 🌙", "gn8 ❤️", "Schlaf gut 💤", "Gute Nacht ❤️ Träum was Süßes"}
	synthMulti = []string{"Einkaufsliste:\nMilch\nEier\nSchokolade 🍫", "Erstens: du bist toll.\nZweitens: ich auch 😎\nDrittens: Pizza 🍕", "Weißt du was?\n\nIch hab dich lieb ❤️"}
	synthMedia = []string{"Bild weggelassen", "Sticker weggelassen", "Audio weggelassen", "Video weggelassen", "GIF weggelassen"}
)

// Synthetic returns the deterministic synthetic conversation.
func Synthetic() []SynthMsg {
	loc := mustBerlin()
	rng := rand.New(rand.NewPCG(2026, 1224))
	var out []SynthMsg
	at := func(d Date, h, m int) time.Time {
		return time.Date(d.Year, d.Month, d.Day, h, m, rng.IntN(60), 0, loc)
	}
	other := func(a string) string {
		if a == SynthMe {
			return SynthPartner
		}
		return SynthMe
	}
	pick := func(list []string) string { return list[rng.IntN(len(list))] }
	who := func() string {
		if rng.IntN(100) < 55 {
			return SynthPartner
		}
		return SynthMe
	}
	gaps := map[Date]bool{{2026, time.May, 19}: true, {2026, time.June, 3}: true, {2026, time.July, 14}: true,
		{2026, time.July, 15}: true, {2026, time.August, 27}: true, {2026, time.September, 21}: true}
	days := DaysBetween(synthFrom, synthTo)
	for i := 0; i <= days; i++ {
		d := synthFrom.AddDays(i)
		if gaps[d] {
			continue
		}
		var day []SynthMsg
		add := func(t time.Time, a string, kind int, text, media string) {
			day = append(day, SynthMsg{T: t, Author: a, Kind: kind, Text: text, Media: media})
		}
		if i == 0 {
			// first exchange of the export
			add(at(d, 9, 12), SynthMe, SynthText, "Hallo Erika! Schön, dass wir jetzt hier schreiben 😊", "")
			add(at(d, 9, 15), SynthPartner, SynthText, "Hallo Max 😊 Freu mich!", "")
		}
		a := who()
		if i > 0 {
			add(at(d, 7+rng.IntN(2), rng.IntN(60)), a, SynthText, pick(synthMorning), "")
		}
		if rng.IntN(2) > 0 {
			add(at(d, 9, rng.IntN(60)), other(a), SynthText, pick(synthReply), "")
		}
		for h := 10 + rng.IntN(4); h <= 21; h += 3 + rng.IntN(5) {
			a := who()
			switch r := rng.IntN(20); {
			case r == 0:
				add(at(d, h, rng.IntN(60)), a, SynthMedia, "", pick(synthMedia))
			case r == 1:
				add(at(d, h, rng.IntN(60)), a, SynthDeleted, "", "")
			case r == 2:
				add(at(d, h, rng.IntN(60)), a, SynthText, pick(synthMulti), "")
			case r == 3:
				add(at(d, h, rng.IntN(60)), a, SynthEdited, pick(synthDay), "")
			default:
				add(at(d, h, rng.IntN(60)), a, SynthText, pick(synthDay), "")
			}
			if rng.IntN(2) == 0 {
				add(at(d, h, 30+rng.IntN(30)), other(a), SynthText, pick(synthReply), "")
			}
		}
		// good night: sometimes after midnight (belongs to the same night)
		first := who()
		if rng.IntN(100) < 10 {
			first = SynthPartner
		}
		switch {
		case d == (Date{2026, time.October, 24}):
			// DST ends on 25.10.2026 at 03:00 – 02:xx happens twice
			n := d.AddDays(1)
			add(at(n, 2, 15), first, SynthText, "Gute Nacht, die Uhr wird zurückgestellt ⏰", "")
			add(at(n, 2, 40), other(first), SynthText, "Eine Stunde mehr Schlaf 😴 Nachti", "")
		case d == (Date{2026, time.June, 30}):
			add(at(d, 23, 59), first, SynthText, "Gute Nacht, gleich ist Juli 🌙", "")
			add(time.Date(n1(d).Year, n1(d).Month, n1(d).Day, 0, 0, 0, 0, loc), other(first), SynthText, "Schon Juli! gn8 😘", "")
		case rng.IntN(5) == 0:
			n := d.AddDays(1)
			add(at(n, 0, 5+rng.IntN(50)), first, SynthText, pick(synthNight), "")
			add(at(n, 1, rng.IntN(30)), other(first), SynthText, pick(synthNight), "")
		case rng.IntN(12) > 0:
			add(at(d, 22, rng.IntN(60)), first, SynthText, pick(synthNight), "")
			add(at(d, 23, rng.IntN(50)), other(first), SynthText, pick(synthNight), "")
		}
		sort.SliceStable(day, func(i, j int) bool { return day[i].T.Before(day[j].T) })
		out = append(out, day...)
	}
	return out
}

func n1(d Date) Date { return d.AddDays(1) }

// RenderAndroid renders the conversation as an Android export
// ("24.12.26, 18:05 - Name: Text", LF line endings, system line first).
func RenderAndroid(msgs []SynthMsg) []byte {
	var b bytes.Buffer
	if len(msgs) > 0 {
		fmt.Fprintf(&b, "%s - Nachrichten und Anrufe sind Ende-zu-Ende-verschlüsselt. Niemand außerhalb dieses Chats kann sie lesen oder anhören, nicht einmal WhatsApp. Tippe, um mehr zu erfahren.\n",
			msgs[0].T.Format("02.01.06, 15:04"))
	}
	for _, m := range msgs {
		text := m.Text
		switch m.Kind {
		case SynthMedia:
			text = "<Medien ausgelassen>"
		case SynthDeleted:
			text = "Diese Nachricht wurde gelöscht"
			if m.Author == SynthMe {
				text = "Du hast diese Nachricht gelöscht"
			}
		case SynthEdited:
			text += " <Diese Nachricht wurde bearbeitet>"
		}
		fmt.Fprintf(&b, "%s - %s: %s\n", m.T.Format("02.01.06, 15:04"), m.Author, text)
	}
	return b.Bytes()
}

// RenderIOS renders the conversation as an iOS export
// ("[24.12.26, 18:05:12] Name: Text", CRLF, U+200E marks on special lines,
// encryption notice attributed to the chat partner).
func RenderIOS(msgs []SynthMsg) []byte {
	var b bytes.Buffer
	line := func(s string) { b.WriteString(strings.ReplaceAll(s, "\n", "\r\n") + "\r\n") }
	if len(msgs) > 0 {
		line(fmt.Sprintf("[%s] %s: %sNachrichten und Anrufe sind Ende-zu-Ende-verschlüsselt. Niemand außerhalb dieses Chats kann sie lesen oder anhören, nicht einmal WhatsApp.",
			msgs[0].T.Format("02.01.06, 15:04:05"), SynthPartner, lrm))
	}
	for _, m := range msgs {
		stamp := "[" + m.T.Format("02.01.06, 15:04:05") + "] "
		switch m.Kind {
		case SynthMedia:
			line(lrm + stamp + m.Author + ": " + lrm + m.Media)
		case SynthDeleted:
			text := "Diese Nachricht wurde gelöscht."
			if m.Author == SynthMe {
				text = "Du hast diese Nachricht gelöscht."
			}
			line(lrm + stamp + m.Author + ": " + lrm + text)
		case SynthEdited:
			line(stamp + m.Author + ": " + m.Text + " " + lrm + "<Diese Nachricht wurde bearbeitet.>")
		default:
			line(stamp + m.Author + ": " + m.Text)
		}
	}
	return b.Bytes()
}
