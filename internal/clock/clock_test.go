package clock

import (
	"testing"
	"time"
)

func TestParseNow(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	got, err := ParseNow("2026-12-05T08:00", loc)
	if err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 12, 5, 7, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if _, err := ParseNow("2026-11-30T23:59:59", loc); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "05.12.2026 08:00", "2026-12-05", "2026-12-05T25:00"} {
		if _, err := ParseNow(bad, loc); err == nil {
			t.Errorf("%q akzeptiert", bad)
		}
	}
}

type fake struct{ t time.Time }

func (f *fake) Now() time.Time { return f.t }

func TestOffsetTicks(t *testing.T) {
	base := &fake{time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)}
	start := time.Date(2026, 12, 5, 7, 0, 0, 0, time.UTC)
	c := NewOffset(base, start)
	if !c.Now().Equal(start) {
		t.Fatalf("got %v", c.Now())
	}
	base.t = base.t.Add(90 * time.Second)
	if want := start.Add(90 * time.Second); !c.Now().Equal(want) {
		t.Fatalf("got %v want %v", c.Now(), want)
	}
}
