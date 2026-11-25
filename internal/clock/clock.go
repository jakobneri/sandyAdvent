// Package clock abstracts "now" so the date lock can be tested and, in dev
// mode, overridden via ADVENT_NOW.
package clock

import (
	"fmt"
	"strings"
	"time"
)

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// System is the real wall clock.
type System struct{}

// Now implements Clock.
func (System) Now() time.Time { return time.Now() }

// Fixed always returns the same instant (useful in tests).
type Fixed struct{ T time.Time }

// Now implements Clock.
func (f Fixed) Now() time.Time { return f.T }

// Offset is a clock that runs at normal speed but shifted so that it showed
// Start at the moment it was created. This keeps countdowns ticking while
// ADVENT_NOW is in effect.
type Offset struct {
	base  Clock
	delta time.Duration
}

// NewOffset returns a clock that reads start "now" and then keeps ticking.
func NewOffset(base Clock, start time.Time) *Offset {
	return &Offset{base: base, delta: start.Sub(base.Now())}
}

// Now implements Clock.
func (o *Offset) Now() time.Time { return o.base.Now().Add(o.delta) }

// NowLayouts are the accepted ADVENT_NOW formats (local time in loc).
var NowLayouts = []string{"2006-01-02T15:04", "2006-01-02T15:04:05"}

// ParseNow parses an ADVENT_NOW value such as "2026-12-05T08:00" as a wall
// clock time in loc (Europe/Berlin).
func ParseNow(s string, loc *time.Location) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range NowLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("ADVENT_NOW %q: erwartetes Format 2006-01-02T15:04 (Europe/Berlin)", s)
}
