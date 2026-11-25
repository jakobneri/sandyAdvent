package waparse

import "github.com/rivo/uniseg"

// Emojis returns the emoji grapheme clusters of s in order. A cluster is one
// emoji as the user sees it: ZWJ sequences (👩‍❤️‍👨), flags (🇩🇪), skin tones
// (👍🏽) and keycaps (1️⃣) stay together.
func Emojis(s string) []string {
	var out []string
	g := uniseg.NewGraphemes(s)
	for g.Next() {
		if c := g.Str(); isEmojiCluster(c) {
			out = append(out, c)
		}
	}
	return out
}

// EmojiKey folds presentation selectors so "❤️" and "❤" count as one emoji.
func EmojiKey(cluster string) string {
	rs := make([]rune, 0, len(cluster))
	for _, r := range cluster {
		if r != 0xFE0F && r != 0xFE0E {
			rs = append(rs, r)
		}
	}
	return string(rs)
}

// EmojiDisplay returns the form of an emoji cluster used for display: a
// single text-default symbol (❤ ☺ ✌ © …) gets U+FE0F appended so it renders
// as a coloured emoji instead of a monochrome text glyph. Everything else is
// returned unchanged.
func EmojiDisplay(cluster string) string {
	rs := []rune(cluster)
	if len(rs) != 1 {
		return cluster
	}
	r := rs[0]
	if (emojiText(r) || (r >= 0x2600 && r <= 0x27BF) || r == 0x3030 || r == 0x303D || r == 0x3297 || r == 0x3299) && !emojiPresentation(r) {
		return cluster + "\uFE0F"
	}
	return cluster
}

// emojiPresentation: BMP symbols shown as emoji even without U+FE0F
// (Unicode property Emoji_Presentation=Yes); they need no selector.
func emojiPresentation(r rune) bool {
	switch {
	case r == 0x231A, r == 0x231B, r >= 0x23E9 && r <= 0x23EC, r == 0x23F0, r == 0x23F3,
		r == 0x25FD, r == 0x25FE, r == 0x2614, r == 0x2615, r >= 0x2648 && r <= 0x2653,
		r == 0x267F, r == 0x2693, r == 0x26A1, r == 0x26AA, r == 0x26AB, r == 0x26BD, r == 0x26BE,
		r == 0x26C4, r == 0x26C5, r == 0x26CE, r == 0x26D4, r == 0x26EA, r == 0x26F2, r == 0x26F3,
		r == 0x26F5, r == 0x26FA, r == 0x26FD, r == 0x2705, r == 0x270A, r == 0x270B, r == 0x2728,
		r == 0x274C, r == 0x274E, r >= 0x2753 && r <= 0x2755, r == 0x2757, r >= 0x2795 && r <= 0x2797,
		r == 0x27B0, r == 0x27BF, r == 0x2B1B, r == 0x2B1C, r == 0x2B50, r == 0x2B55:
		return true
	}
	return false
}

func isEmojiCluster(c string) bool {
	rs := []rune(c)
	if len(rs) == 0 {
		return false
	}
	hasVS16 := false
	for _, r := range rs {
		if r == 0x20E3 { // keycap 1️⃣ #️⃣
			return true
		}
		if r == 0xFE0F {
			hasVS16 = true
		}
	}
	first := rs[0]
	if r := first; r >= 0x1F1E6 && r <= 0x1F1FF { // regional indicators (flags)
		return len(rs) >= 2
	}
	if emojiDefault(first) {
		return true
	}
	return hasVS16 && emojiText(first)
}

// emojiDefault: code points that are shown as emoji without a selector.
func emojiDefault(r rune) bool {
	switch {
	case r >= 0x1F300 && r <= 0x1F5FF, // symbols & pictographs
		r >= 0x1F600 && r <= 0x1F64F, // emoticons
		r >= 0x1F680 && r <= 0x1F6FF, // transport & map
		r >= 0x1F7E0 && r <= 0x1F7F0, // coloured circles/squares
		r >= 0x1F900 && r <= 0x1F9FF, // supplemental symbols
		r >= 0x1FA70 && r <= 0x1FAFF, // symbols ext-A (🫶, 🫪 …)
		r == 0x1F004, r == 0x1F0CF, r == 0x1F18E,
		r >= 0x1F191 && r <= 0x1F19A,
		r >= 0x1F201 && r <= 0x1F251,
		r >= 0x2600 && r <= 0x27BF, // misc symbols + dingbats (☀ ❤ ✨ ✅ …)
		r == 0x231A, r == 0x231B, r >= 0x23E9 && r <= 0x23F3, r >= 0x23F8 && r <= 0x23FA,
		r == 0x2B50, r == 0x2B55, r >= 0x2B1B && r <= 0x2B1C, r >= 0x2B05 && r <= 0x2B07,
		r == 0x3030, r == 0x303D, r == 0x3297, r == 0x3299:
		return true
	}
	return false
}

// emojiText: symbols that are emoji only with U+FE0F (© ® ™ ‼ ↔ …).
func emojiText(r rune) bool {
	return r == 0x00A9 || r == 0x00AE || (r >= 0x2000 && r <= 0x2BFF) || (r >= 0x1F100 && r <= 0x1F2FF)
}
