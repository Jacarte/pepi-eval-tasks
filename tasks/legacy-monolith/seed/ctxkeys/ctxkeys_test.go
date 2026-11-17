package ctxkeys

import (
	"context"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	ctx := context.Background()
	if RequestIDFrom(ctx) != "" || TraceIDFrom(ctx) != "" {
		t.Fatal("empty context should carry nothing")
	}
	if ActorFrom(ctx) != "system" || TenantFrom(ctx) != "default" {
		t.Fatal("unexpected defaults")
	}
	ctx = WithRequestID(ctx, "req-1")
	ctx = WithTraceID(ctx, "trace-9")
	ctx = WithActor(ctx, "u-7")
	ctx = WithTenant(ctx, "eu")
	if RequestIDFrom(ctx) != "req-1" || TraceIDFrom(ctx) != "trace-9" {
		t.Fatalf("ids = %q %q", RequestIDFrom(ctx), TraceIDFrom(ctx))
	}
	if ActorFrom(ctx) != "u-7" || TenantFrom(ctx) != "eu" {
		t.Fatal("actor/tenant lost")
	}
	if got, _ := ctx.Value(RequestID).(string); got != "req-1" {
		t.Fatalf("raw lookup = %q", got)
	}
}

func TestKeyNames(t *testing.T) {
	for k, want := range map[key]string{RequestID: "request_id", TraceID: "trace_id", Actor: "actor", Tenant: "tenant", key(42): "unknown"} {
		if k.String() != want {
			t.Errorf("%d.String() = %q", int(k), k.String())
		}
	}
}
