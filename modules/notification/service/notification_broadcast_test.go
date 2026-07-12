package service

import (
	"context"
	"errors"
	"testing"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/dto"
	notifEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/entity"
	"go.uber.org/zap"
)

// fakePromoQueue is a minimal PromotionalEmailQueue stub — no broker needed.
type fakePromoQueue struct {
	sent []string // recipients passed to EnqueuePromotionalEmail
	err  error
}

func (f *fakePromoQueue) EnqueuePromotionalEmail(to, _, _, _ string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, to)
	return nil
}

func TestBroadcast_EmailChannelWithPromoQueueEnqueuesAsync(t *testing.T) {
	promo := &fakePromoQueue{}
	svc := &NotificationService{promoQueue: promo, log: zap.NewNop()}

	req := dto.BroadcastRequest{
		UserIDs: []string{"u1", "u2", "u3"},
		Type:    notifEntity.TypeSystem,
		Channel: notifEntity.ChannelEmail,
		Title:   "Big Sale",
		Body:    "50% off everything",
		Data:    map[string]interface{}{"email": "recipient@example.com"},
	}

	success, failed, err := svc.Broadcast(context.Background(), req)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if failed != 0 {
		t.Fatalf("expected 0 failed, got %d", failed)
	}
	if success != len(req.UserIDs) {
		t.Fatalf("expected %d success, got %d", len(req.UserIDs), success)
	}
	if len(promo.sent) != len(req.UserIDs) {
		t.Fatalf("expected %d enqueued emails, got %d", len(req.UserIDs), len(promo.sent))
	}
}

func TestBroadcast_EmailChannelWithoutRecipientAddressFails(t *testing.T) {
	promo := &fakePromoQueue{}
	svc := &NotificationService{promoQueue: promo, log: zap.NewNop()}

	req := dto.BroadcastRequest{
		UserIDs: []string{"u1", "u2"},
		Type:    notifEntity.TypeSystem,
		Channel: notifEntity.ChannelEmail,
		Title:   "Big Sale",
		Body:    "50% off everything",
		// Data["email"] intentionally omitted.
	}

	success, failed, err := svc.Broadcast(context.Background(), req)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if success != 0 || failed != len(req.UserIDs) {
		t.Fatalf("expected all recipients to fail without an address, got success=%d failed=%d", success, failed)
	}
}

func TestBroadcast_EmailChannelEnqueueErrorCountsAsFailed(t *testing.T) {
	promo := &fakePromoQueue{err: errors.New("broker unavailable")}
	svc := &NotificationService{promoQueue: promo, log: zap.NewNop()}

	req := dto.BroadcastRequest{
		UserIDs: []string{"u1"},
		Type:    notifEntity.TypeSystem,
		Channel: notifEntity.ChannelEmail,
		Title:   "Big Sale",
		Body:    "50% off everything",
		Data:    map[string]interface{}{"email": "recipient@example.com"},
	}

	success, failed, err := svc.Broadcast(context.Background(), req)
	if err != nil {
		t.Fatalf("Broadcast: %v", err)
	}
	if success != 0 || failed != 1 {
		t.Fatalf("expected the enqueue error to count as failed, got success=%d failed=%d", success, failed)
	}
}

// The nil-promoQueue fallback (Broadcast's original synchronous per-user
// Send path) isn't covered here — Send touches s.prefsCol/s.deviceCol,
// which need a real *mongo.Collection and so require integration-style
// setup per CLAUDE.md's "never real MongoDB/Redis in unit tests" rule. The
// three cases above cover the new code path this change adds.
