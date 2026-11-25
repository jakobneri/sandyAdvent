package advent

import (
	"testing"
	"time"
)

func TestDayLists(t *testing.T) {
	if len(QRDays) != 17 || len(VoucherDays) != 7 {
		t.Fatalf("QR=%d voucher=%d", len(QRDays), len(VoucherDays))
	}
	for d := 1; d <= 24; d++ {
		if IsQRDay(d) == IsVoucherDay(d) {
			t.Errorf("Tag %d muss genau eins von QR/Gutschein sein", d)
		}
	}
}

func TestUnlockBoundaries(t *testing.T) {
	for _, day := range QRDays {
		open := UnlockTime(2026, day)
		if open.In(Berlin).Hour() != 0 || open.In(Berlin).Day() != day {
			t.Fatalf("Tag %d: Öffnung %v", day, open)
		}
		before := time.Date(2026, 12, day, 0, 0, 0, 0, Berlin).Add(-time.Second) // 23:59:59 am Vortag
		if IsUnlocked(before, 2026, day) {
			t.Errorf("Tag %d: um %v schon offen", day, before)
		}
		if !IsUnlocked(open, 2026, day) {
			t.Errorf("Tag %d: um %v noch zu", day, open)
		}
	}
}

func TestUnlockUTCInstants(t *testing.T) {
	// 01.12. 00:00 Berlin (CET, UTC+1) = 30.11. 23:00 UTC.
	if IsUnlocked(time.Date(2026, 11, 30, 22, 59, 59, 999e6, time.UTC), 2026, 1) {
		t.Error("30.11. 22:59:59.999 UTC darf Tür 1 nicht öffnen")
	}
	if !IsUnlocked(time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC), 2026, 1) {
		t.Error("30.11. 23:00 UTC muss Tür 1 öffnen")
	}
	// Doors never close again.
	if !IsUnlocked(time.Date(2027, 1, 15, 12, 0, 0, 0, Berlin), 2026, 24) {
		t.Error("Tür 24 muss nach Weihnachten offen bleiben")
	}
}

func TestDaysUntilDSTSafe(t *testing.T) {
	reveal := Date{2026, time.December, 24}
	cases := []struct {
		now  time.Time
		want int
	}{
		{time.Date(2026, 12, 23, 23, 59, 59, 0, Berlin), 1},
		{time.Date(2026, 12, 24, 0, 0, 0, 0, Berlin), 0},
		{time.Date(2026, 12, 1, 0, 0, 0, 0, Berlin), 23},
		{time.Date(2026, 12, 25, 10, 0, 0, 0, Berlin), -1},
		// Before the DST switch on 25.10.2026 (25 hour day): still exact.
		{time.Date(2026, 10, 24, 23, 30, 0, 0, Berlin), 61},
		{time.Date(2026, 10, 25, 0, 30, 0, 0, Berlin), 60},
		{time.Date(2026, 10, 25, 23, 30, 0, 0, Berlin), 60},
		// UTC instant that is already the next day in Berlin.
		{time.Date(2026, 12, 22, 23, 30, 0, 0, time.UTC), 1},
	}
	for _, c := range cases {
		if got := DaysUntil(c.now, reveal); got != c.want {
			t.Errorf("DaysUntil(%v) = %d, want %d", c.now, got, c.want)
		}
	}
	// Across the spring DST switch (29.03.2026, 23 hour day).
	if got := DaysBetween(Date{2026, 3, 28}, Date{2026, 3, 30}); got != 2 {
		t.Errorf("DaysBetween über Sommerzeit = %d", got)
	}
}

func TestBannerText(t *testing.T) {
	reveal := Date{2026, time.December, 24}
	cases := map[time.Time]string{
		time.Date(2026, 12, 1, 8, 0, 0, 0, Berlin):     "Schatzogram – noch 23 Tage",
		time.Date(2026, 12, 22, 23, 59, 0, 0, Berlin):  "Schatzogram – noch 2 Tage",
		time.Date(2026, 12, 23, 0, 0, 0, 0, Berlin):    "Schatzogram – noch 1 Tag",
		time.Date(2026, 12, 23, 23, 59, 59, 0, Berlin): "Schatzogram – noch 1 Tag",
		time.Date(2026, 12, 24, 0, 0, 0, 0, Berlin):    "Schatzogram – heute",
		time.Date(2026, 12, 30, 0, 0, 0, 0, Berlin):    "Schatzogram – heute",
	}
	for now, want := range cases {
		if got := BannerText(now, reveal); got != want {
			t.Errorf("BannerText(%v) = %q, want %q", now, got, want)
		}
	}
}

func TestYearsAgoAndFormat(t *testing.T) {
	on := Date{2026, time.December, 13}
	if got := YearsAgo(Date{2023, time.December, 13}, on); got != 3 {
		t.Errorf("got %d", got)
	}
	if got := YearsAgo(Date{2023, time.December, 14}, on); got != 2 {
		t.Errorf("got %d", got)
	}
	if got := (Date{2023, time.March, 5}).FormatDE(); got != "5. März 2023" {
		t.Errorf("got %q", got)
	}
	if _, err := ParseDate("2023-13-01"); err == nil {
		t.Error("ungültiges Datum akzeptiert")
	}
}
