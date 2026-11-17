// Package ctxkeys holds the context keys shared by the HTTP layer, the job
// runner and the domain services. It must stay dependency free.
package ctxkeys

import "context"

type key int

const (
	// RequestID carries the correlation id assigned to each inbound request
	// (httpapi) or job execution (scheduling).
	RequestID key = iota + 1
	// TraceID is propagated from the edge proxy for distributed tracing.
	TraceID
	// Actor is the authenticated principal: a user id or a service name.
	Actor
	// Tenant is the storefront the request belongs to.
	Tenant
)

func (k key) String() string {
	switch k {
	case RequestID:
		return "request_id"
	case TraceID:
		return "trace_id"
	case Actor:
		return "actor"
	case Tenant:
		return "tenant"
	}
	return "unknown"
}

// WithRequestID returns a copy of ctx carrying id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestID, id)
}

// RequestIDFrom returns the request id carried by ctx, or "".
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(RequestID).(string)
	return v
}

// WithTraceID returns a copy of ctx carrying the trace id.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, TraceID, id)
}

// TraceIDFrom returns the trace id carried by ctx, or "".
func TraceIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(TraceID).(string)
	return v
}

// WithActor returns a copy of ctx carrying the acting principal.
func WithActor(ctx context.Context, actor string) context.Context {
	return context.WithValue(ctx, Actor, actor)
}

// ActorFrom returns the actor carried by ctx, or "system".
func ActorFrom(ctx context.Context) string {
	if v, ok := ctx.Value(Actor).(string); ok && v != "" {
		return v
	}
	return "system"
}

// WithTenant returns a copy of ctx carrying the tenant.
func WithTenant(ctx context.Context, tenant string) context.Context {
	return context.WithValue(ctx, Tenant, tenant)
}

// TenantFrom returns the tenant carried by ctx, or "default".
func TenantFrom(ctx context.Context) string {
	if v, ok := ctx.Value(Tenant).(string); ok && v != "" {
		return v
	}
	return "default"
}
