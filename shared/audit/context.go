package audit

import (
	"context"

	"github.com/google/uuid"
)

// RequestInfo is the "who and from where" half of an audit entry. It rides on the
// request context so a service method does not have to thread the actor, IP and
// client platform through every signature — and so those facts can never be taken
// from a request body.
//
// It is what turns "somebody changed this price" into "this user changed it, from
// this device, at this IP, in this request" — the difference between a log and an
// investigation.
type RequestInfo struct {
	TenantID       uuid.UUID
	ActorID        string
	ActorRole      string
	ClientPlatform string
	IPAddress      string
	CorrelationID  string
}

type ctxKey struct{}
type providerKey struct{}

// WithRequest stores the request's audit facts directly (background jobs, tests).
func WithRequest(ctx context.Context, info RequestInfo) context.Context {
	return context.WithValue(ctx, ctxKey{}, info)
}

// WithProvider stores a *lazy* source of the request's audit facts. HTTP needs
// this: the audit context is installed by a global middleware, but the actor
// claims are only set later by the per-group auth middleware. Deferring the read
// until an audit entry is actually written means the claims are always there,
// without depending on middleware ordering.
func WithProvider(ctx context.Context, f func() RequestInfo) context.Context {
	return context.WithValue(ctx, providerKey{}, f)
}

// FromContext reads the request's audit facts. The zero value is returned when
// nothing was set — background jobs have no request, and attribute their entries
// to "system" explicitly.
func FromContext(ctx context.Context) RequestInfo {
	if f, ok := ctx.Value(providerKey{}).(func() RequestInfo); ok && f != nil {
		return f()
	}
	if v, ok := ctx.Value(ctxKey{}).(RequestInfo); ok {
		return v
	}
	return RequestInfo{}
}

// Bind fills the actor/request fields of an entry from the context, leaving any
// value the caller already set. Call it in the recorder, so every writer benefits
// without each call site remembering.
func (e Entry) Bind(info RequestInfo) Entry {
	if e.TenantID == uuid.Nil {
		e.TenantID = info.TenantID
	}
	if e.ActorID == "" {
		e.ActorID = info.ActorID
	}
	if e.ActorRole == "" {
		e.ActorRole = info.ActorRole
	}
	if e.ClientPlatform == "" {
		e.ClientPlatform = info.ClientPlatform
	}
	if e.IPAddress == "" {
		e.IPAddress = info.IPAddress
	}
	if e.CorrelationID == "" {
		e.CorrelationID = info.CorrelationID
	}
	return e
}
