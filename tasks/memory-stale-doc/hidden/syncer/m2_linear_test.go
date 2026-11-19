package syncer_test

import (
	"reflect"
	"testing"
	"time"

	"jobq/syncer"
)

func TestM2LinearWaits(t *testing.T) {
	_, waits, _ := drive(syncer.Config(), errBoom)
	ms := time.Millisecond
	want := []time.Duration{200 * ms, 400 * ms, 600 * ms, 600 * ms}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("waits %v, want %v", waits, want)
	}
}
