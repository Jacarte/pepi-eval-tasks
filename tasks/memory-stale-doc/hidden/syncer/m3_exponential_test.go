package syncer_test

import (
	"reflect"
	"testing"
	"time"

	"jobq/syncer"
)

func TestM3ExponentialWaits(t *testing.T) {
	_, waits, _ := drive(syncer.BulkConfig(), errBoom)
	ms := time.Millisecond
	want := []time.Duration{100 * ms, 200 * ms, 400 * ms, 800 * ms, 1000 * ms}
	if !reflect.DeepEqual(waits, want) {
		t.Fatalf("waits %v, want %v", waits, want)
	}
}
