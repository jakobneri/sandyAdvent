// Package advent contains the calendar rules: which days have QR codes,
// when a door unlocks (Europe/Berlin) and the Schatzogram banner countdown.
package advent

import (
	"fmt"
	"time"
	_ "time/tzdata" // the Pi must not depend on system tzdata
)

// Berlin is the fixed time zone of the calendar.
var Berlin = mustLoad("Europe/Berlin")

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// QRDays are the 17 days that have a QR code, a token and a door page.
var QRDays = []int{1, 3, 4, 6, 7, 9, 10, 12, 13, 15, 16, 18, 19, 21, 22, 23, 24}

// VoucherDays are printed vouchers: no route, no QR code, no token.
var VoucherDays = []int{2, 5, 8, 11, 14, 17, 20}

// IsQRDay reports whether day has a QR door.
func IsQRDay(day int) bool {
	for _, d := range QRDays {
		if d == day {
			return true
		}
	}
	return false
}

// IsVoucherDay reports whether day is a printed voucher day.
func IsVoucherDay(day int) bool {
	for _, d := range VoucherDays {
		if d == day {
			return true
		}
	}
	return false
}

// UnlockTime is the instant door day of year opens: December <day>, 00:00 Berlin.
func UnlockTime(year, day int) time.Time {
	return time.Date(year, time.December, day, 0, 0, 0, 0, Berlin)
}

// IsUnlocked reports whether the door is open at now.
func IsUnlocked(now time.Time, year, day int) bool {
	return !now.Before(UnlockTime(year, day))
}

// Date is a civil calendar date without time or zone.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// ParseDate parses "YYYY-MM-DD".
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("ungültiges Datum %q (erwartet YYYY-MM-DD)", s)
	}
	return Date{t.Year(), t.Month(), t.Day()}, nil
}

// DateOf returns the Berlin calendar date of t.
func DateOf(t time.Time) Date {
	y, m, d := t.In(Berlin).Date()
	return Date{y, m, d}
}

// String formats as YYYY-MM-DD.
func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day) }

// IsZero reports whether d is unset.
func (d Date) IsZero() bool { return d.Year == 0 }

// dayNumber converts to a day count; computed in UTC so DST never matters.
func (d Date) dayNumber() int64 {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC).Unix() / 86400
}

// DaysBetween returns the number of calendar days from a to b (b - a).
func DaysBetween(a, b Date) int { return int(b.dayNumber() - a.dayNumber()) }

// DaysUntil returns the number of Berlin calendar days from now until target.
func DaysUntil(now time.Time, target Date) int { return DaysBetween(DateOf(now), target) }

// BannerText is the discreet Schatzogram teaser (docs/interface.md §2).
func BannerText(now time.Time, reveal Date) string {
	x := DaysUntil(now, reveal)
	if x <= 0 {
		return "Schatzogram – heute"
	}
	return "Schatzogram – noch " + Count(x, "Tag", "Tage")
}

// ShowsBanner reports whether door day shows the Schatzogram banner: not on
// 23 (teaser) and 24 (reveal), where the teaser/reveal takes over.
func ShowsBanner(day int) bool { return day != 23 && day != 24 }

// Plural returns one for n == 1 (also -1) and many otherwise ("0 Tage").
func Plural(n int, one, many string) string {
	if n == 1 || n == -1 {
		return one
	}
	return many
}

// Count formats n with the correctly inflected unit: "1 Tag", "2 Tage".
func Count(n int, one, many string) string { return fmt.Sprintf("%d %s", n, Plural(n, one, many)) }

// YearsAgo returns the number of full years between then and on.
func YearsAgo(then, on Date) int {
	years := on.Year - then.Year
	if on.Month < then.Month || (on.Month == then.Month && on.Day < then.Day) {
		years--
	}
	return years
}

var monthsDE = [...]string{"Januar", "Februar", "März", "April", "Mai", "Juni", "Juli",
	"August", "September", "Oktober", "November", "Dezember"}

// MonthDE returns the German month name.
func MonthDE(m time.Month) string { return monthsDE[m-1] }

// FormatDE formats a date like "13. Dezember 2023".
func (d Date) FormatDE() string { return fmt.Sprintf("%d. %s %d", d.Day, MonthDE(d.Month), d.Year) }
