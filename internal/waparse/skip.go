package waparse

import (
	"regexp"
	"strings"
)

// The lists below are data, not logic: extend them when a new WhatsApp
// version writes new placeholder texts. Matching happens after invisible
// direction marks (U+200E …) are removed and spaces are normalized.

// placeholderTexts are whole-message texts that stand for media, deleted or
// otherwise empty messages. They still count as a message of their author
// (activity, streak, hours), but their text is ignored.
var placeholderTexts = []string{
	// Android (Export „ohne Medien“)
	"<Medien ausgelassen>",
	"<Medien weggelassen>",
	"Diese Nachricht wurde gelöscht",
	"Diese Nachricht wurde gelöscht.",
	"Du hast diese Nachricht gelöscht",
	"Du hast diese Nachricht gelöscht.",
	"Diese Nachricht wurde von einem Admin gelöscht",
	"null", // einmal ansehbare Medien
	// iOS
	"Bild weggelassen",
	"Video weggelassen",
	"Audio weggelassen",
	"Sticker weggelassen",
	"GIF weggelassen",
	"Dokument weggelassen",
	"Kontaktkarte weggelassen",
	"Kontakt weggelassen",
	"Verpasster Sprachanruf",
	"Verpasster Videoanruf",
	"Sprachanruf",
	"Videoanruf",
	"Einmal ansehen",
	// englische Exporte (falls das Handy auf Englisch steht)
	"<Media omitted>",
	"This message was deleted",
	"You deleted this message",
	"image omitted",
	"video omitted",
	"audio omitted",
	"sticker omitted",
	"GIF omitted",
	"document omitted",
}

// placeholderPrefixes mark placeholder messages by their beginning
// (attachments with file names, locations, polls).
var placeholderPrefixes = []string{
	"<Anhang: ",
	"<attached: ",
	"Standort: https://",
	"Live-Standort",
	"location: https://",
	"UMFRAGE:",
	"POLL:",
}

// placeholderSuffixes are attachment lines like "IMG-2026….jpg (Datei angehängt)".
var placeholderSuffixes = []string{
	"(Datei angehängt)",
	"(file attached)",
	"Dokument weggelassen",
}

// systemPatterns are WhatsApp notices. Lines without "Name: " are always
// system lines; these patterns additionally catch notices that carry an
// author (iOS attributes the encryption notice to the chat partner). They
// start with WhatsApp's fixed wording, so ordinary messages that merely
// mention e.g. "Sicherheitsnummer" stay.
var systemPatterns = compileAll(
	`^Nachrichten und Anrufe sind Ende-zu-Ende-verschlüsselt`,
	`^Messages and calls are end-to-end encrypted`,
	`^(Deine|Ihre) Sicherheitsnummer (mit|für) .{1,60} hat sich geändert`,
	`^Your security code with .{1,60} changed`,
	`^Du hast selbstlöschende Nachrichten (aktiviert|deaktiviert)`,
	`^Du hast diesen Kontakt (blockiert|freigegeben)`,
	`^Diese Nachricht kann nicht angezeigt werden`,
	`^Warte auf diese Nachricht`,
	`^Wartet auf diese Nachricht`,
	// "Name hat zu einer neuen Nummer gewechselt. Tippe, um …" – the tail is
	// WhatsApp's own wording, nobody types it.
	`^.{1,60} hat zu einer neuen Nummer gewechselt\. Tippe, um`,
)

// markedNoticePatterns begin with an arbitrary name ("Erika hat …"), so a
// normal message like "Mein Bruder hat zu Vodafone gewechselt" would match
// too. They are only applied to message bodies that start with U+200E,
// the invisible mark iOS puts in front of its notices (and placeholders).
var markedNoticePatterns = compileAll(
	`^.{1,60} hat selbstlöschende Nachrichten (aktiviert|deaktiviert)`,
	`^.{1,60} (verwendet|benutzt) jetzt eine andere Nummer`,
	`^.{1,60} hat zu .{1,40} gewechselt`,
	`^.{1,60} ist jetzt ein Business-Konto`,
)

// editedSuffix is appended to edited messages; it is stripped.
var editedSuffix = regexp.MustCompile(`\s*<(Diese Nachricht wurde bearbeitet|This message was edited)\.?>\s*$`)

func compileAll(pats ...string) []*regexp.Regexp {
	out := make([]*regexp.Regexp, len(pats))
	for i, p := range pats {
		out[i] = regexp.MustCompile(`(?i)` + p)
	}
	return out
}

var placeholderSet = func() map[string]bool {
	m := map[string]bool{}
	for _, s := range placeholderTexts {
		m[strings.ToLower(s)] = true
	}
	return m
}()

// isPlaceholder reports whether a (cleaned) message text is a media/deleted
// placeholder.
func isPlaceholder(text string) bool {
	t := strings.TrimSpace(text)
	if placeholderSet[strings.ToLower(t)] {
		return true
	}
	for _, p := range placeholderPrefixes {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	for _, s := range placeholderSuffixes {
		if strings.HasSuffix(t, s) {
			return true
		}
	}
	return false
}

// isSystem reports whether a (cleaned) message text is a WhatsApp notice.
// marked says the raw body started with U+200E (iOS notice marker); only
// then are the name-led notice patterns applied.
func isSystem(text string, marked bool) bool {
	for _, re := range systemPatterns {
		if re.MatchString(text) {
			return true
		}
	}
	if marked {
		for _, re := range markedNoticePatterns {
			if re.MatchString(text) {
				return true
			}
		}
	}
	return false
}
