package clock

import (
	"testing"
	"time"
)

func TestFixed(t *testing.T) {
	start := time.Date(2024, 2, 28, 23, 0, 0, 0, time.UTC)
	c := NewFixed(start)
	c.Advance(2 * time.Hour)
	if got := c.Now(); !got.Equal(start.Add(2 * time.Hour)) {
		t.Fatalf("Now = %v", got)
	}
	if got := StartOfDay(c.Now()); got.Day() != 29 || got.Hour() != 0 {
		t.Fatalf("StartOfDay = %v", got)
	}
	if _, ok := OrSystem(nil).(System); !ok {
		t.Fatal("OrSystem(nil) should be System")
	}
}
