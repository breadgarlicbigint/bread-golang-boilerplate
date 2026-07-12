package router

import (
	"errors"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
)

// fakePublisher is a minimal queue.Publisher stub that records what it was
// asked to do, so tests can assert routing without a real broker.
type fakePublisher struct {
	name        string
	enqueued    []string // task types passed to Enqueue
	emails      []string // "to" passed to EnqueueEmail
	promoEmails []string // "to" passed to EnqueuePromotionalEmail
	closed      bool
	enqueueErr  error
	closeErr    error
}

func (f *fakePublisher) Enqueue(taskType string, _ any) error {
	f.enqueued = append(f.enqueued, taskType)
	return f.enqueueErr
}

func (f *fakePublisher) EnqueueEmail(to, _, _, _ string) error {
	f.emails = append(f.emails, to)
	return f.enqueueErr
}

func (f *fakePublisher) EnqueuePromotionalEmail(to, _, _, _ string) error {
	f.promoEmails = append(f.promoEmails, to)
	return f.enqueueErr
}

func (f *fakePublisher) Close() error {
	f.closed = true
	return f.closeErr
}

var _ queue.Publisher = (*fakePublisher)(nil)

func TestRouter_UnroutedTaskTypeUsesDefault(t *testing.T) {
	def := &fakePublisher{name: "default"}
	r := New(def)

	if err := r.Enqueue(queue.TaskCleanupSessions, nil); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if len(def.enqueued) != 1 || def.enqueued[0] != queue.TaskCleanupSessions {
		t.Fatalf("expected default publisher to receive TaskCleanupSessions, got %v", def.enqueued)
	}
}

func TestRouter_EnqueueEmailUsesTransactionalRoute(t *testing.T) {
	def := &fakePublisher{name: "default"}
	tx := &fakePublisher{name: "transactional"}
	r := New(def)
	r.Route(queue.TaskSendEmail, tx)

	if err := r.EnqueueEmail("user@example.com", "subject", "<p>hi</p>", "hi"); err != nil {
		t.Fatalf("EnqueueEmail: %v", err)
	}
	if len(tx.emails) != 1 || tx.emails[0] != "user@example.com" {
		t.Fatalf("expected transactional publisher to receive the email, got %v", tx.emails)
	}
	if len(def.emails) != 0 {
		t.Fatalf("expected default publisher to receive nothing, got %v", def.emails)
	}
}

func TestRouter_EnqueuePromotionalEmailUsesPromotionalRoute(t *testing.T) {
	def := &fakePublisher{name: "default"}
	promo := &fakePublisher{name: "promotional"}
	r := New(def)
	r.Route(queue.TaskSendPromotionalEmail, promo)

	if err := r.EnqueuePromotionalEmail("bulk@example.com", "subject", "<p>hi</p>", "hi"); err != nil {
		t.Fatalf("EnqueuePromotionalEmail: %v", err)
	}
	if len(promo.promoEmails) != 1 || promo.promoEmails[0] != "bulk@example.com" {
		t.Fatalf("expected promotional publisher to receive the email, got %v", promo.promoEmails)
	}
	if len(def.promoEmails) != 0 {
		t.Fatalf("expected default publisher to receive nothing, got %v", def.promoEmails)
	}
}

func TestRouter_EnqueuePromotionalEmailFallsBackToDefaultWhenUnrouted(t *testing.T) {
	def := &fakePublisher{name: "default"}
	r := New(def)

	if err := r.EnqueuePromotionalEmail("bulk@example.com", "subject", "<p>hi</p>", "hi"); err != nil {
		t.Fatalf("EnqueuePromotionalEmail: %v", err)
	}
	if len(def.promoEmails) != 1 {
		t.Fatalf("expected default publisher to receive the promotional email when unrouted, got %v", def.promoEmails)
	}
}

func TestRouter_CloseClosesEachUniquePublisherOnce(t *testing.T) {
	def := &fakePublisher{name: "default"}
	shared := &fakePublisher{name: "shared"}
	r := New(def)
	r.Route(queue.TaskSendEmail, shared)
	r.Route(queue.TaskSendPromotionalEmail, shared) // same publisher, two routes

	if err := r.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !def.closed {
		t.Fatal("expected default publisher to be closed")
	}
	if !shared.closed {
		t.Fatal("expected shared publisher to be closed")
	}
	// closers should have deduped `shared` to one entry despite two routes.
	if len(r.closers) != 2 {
		t.Fatalf("expected 2 unique closers (default + shared), got %d", len(r.closers))
	}
}

func TestRouter_CloseReturnsFirstErrorButStillClosesTheRest(t *testing.T) {
	failErr := errors.New("close failed")
	def := &fakePublisher{name: "default", closeErr: failErr}
	promo := &fakePublisher{name: "promotional"}
	r := New(def)
	r.Route(queue.TaskSendPromotionalEmail, promo)

	err := r.Close()
	if !errors.Is(err, failErr) {
		t.Fatalf("expected Close to surface the default publisher's error, got %v", err)
	}
	if !def.closed || !promo.closed {
		t.Fatalf("expected both publishers closed regardless of error, got def.closed=%v promo.closed=%v", def.closed, promo.closed)
	}
}
