// Package clock is the one place the app asks what time it is on a path where a
// test must be able to answer differently.
//
// A LEAF package: it imports only the standard library, so any service, handler
// or middleware can depend on it.
//
// The override rides on the request CONTEXT, not on a package-level variable. A
// settable global would be shared by every in-flight request and by every
// parallel test in the same binary, turning "what time is it" into a data race.
package clock

import (
	"context"
	"time"
)

type ctxKey struct{}

// WithNow returns a context whose Now() is t. Set only by the test-clock
// middleware (itself off unless ALLOW_TEST_CLOCK=true) and directly by unit and
// integration tests, which never involve a header.
func WithNow(ctx context.Context, t time.Time) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}

// Now is the current instant for this request: the override when one is present,
// otherwise the real clock. Code on a date-sensitive path should call this once
// and pass the result down — two calls are two instants.
func Now(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ctxKey{}).(time.Time); ok {
		return t
	}
	return time.Now()
}
