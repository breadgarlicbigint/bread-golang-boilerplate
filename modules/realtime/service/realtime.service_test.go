package service

import (
	"testing"

	realtimeContract "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/realtime"
)

// fakeClient is a minimal pkg/realtime.Client for exercising RealtimeService
// against a real (in-process, no external deps) Hub.
type fakeClient struct {
	id  string
	buf chan realtime.Event
}

func newFakeClient(id string) *fakeClient {
	return &fakeClient{id: id, buf: make(chan realtime.Event, 4)}
}

func (f *fakeClient) ID() string { return f.id }

func (f *fakeClient) Send(evt realtime.Event) bool {
	select {
	case f.buf <- evt:
		return true
	default:
		return false
	}
}

func TestUserTopicNamingConvention(t *testing.T) {
	got := UserTopic("abc-123")
	want := "user:abc-123"
	if got != want {
		t.Errorf("UserTopic(%q) = %q, want %q", "abc-123", got, want)
	}
}

func TestPublishToUserDeliversToSubscribedClient(t *testing.T) {
	hub := realtime.NewHub()
	svc := New(hub)
	client := newFakeClient("c1")
	hub.Subscribe(UserTopic("user-1"), client)

	n, err := svc.PublishToUser("user-1", realtimeContract.Event{
		Type:  "notification",
		Title: "Hello",
		Body:  "World",
		Data:  map[string]any{"id": "abc"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 delivery, got %d", n)
	}
	select {
	case evt := <-client.buf:
		if evt.Type != "notification" || evt.Title != "Hello" || evt.Body != "World" {
			t.Errorf("event fields not preserved: %+v", evt)
		}
		data, ok := evt.Data.(map[string]any)
		if !ok || data["id"] != "abc" {
			t.Errorf("event data not preserved: %+v", evt.Data)
		}
	default:
		t.Fatal("client did not receive the event")
	}
}

func TestPublishToUserReturnsZeroWhenNobodyConnected(t *testing.T) {
	hub := realtime.NewHub()
	svc := New(hub)

	n, err := svc.PublishToUser("nobody-connected", realtimeContract.Event{Type: "notification"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 deliveries, got %d", n)
	}
}

func TestPublishToTopicDeliversToArbitraryTopic(t *testing.T) {
	hub := realtime.NewHub()
	svc := New(hub)
	client := newFakeClient("c1")
	hub.Subscribe("custom-topic", client)

	n, err := svc.PublishToTopic("custom-topic", realtimeContract.Event{Type: "custom"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 delivery, got %d", n)
	}
}

func TestPublishToUserDoesNotLeakToOtherUsers(t *testing.T) {
	hub := realtime.NewHub()
	svc := New(hub)
	other := newFakeClient("other")
	hub.Subscribe(UserTopic("user-2"), other)

	if _, err := svc.PublishToUser("user-1", realtimeContract.Event{Type: "notification"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	select {
	case <-other.buf:
		t.Fatal("a different user's client should not have received the event")
	default:
	}
}

func TestStatsReflectsHubState(t *testing.T) {
	hub := realtime.NewHub()
	svc := New(hub)
	a := newFakeClient("a")
	b := newFakeClient("b")
	hub.Subscribe(UserTopic("user-1"), a)
	hub.Subscribe("custom-topic", a)
	hub.Subscribe("custom-topic", b)

	stats := svc.Stats()
	if stats.TopicCount != 2 {
		t.Errorf("expected 2 topics, got %d", stats.TopicCount)
	}
	if stats.ClientCount != 2 {
		t.Errorf("expected 2 distinct clients, got %d", stats.ClientCount)
	}
	if stats.Topics["custom-topic"] != 2 {
		t.Errorf("expected 2 subscribers on custom-topic, got %d", stats.Topics["custom-topic"])
	}
}
