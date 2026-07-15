package realtime

import "testing"

type fakeClient struct {
	id   string
	buf  chan Event
	full bool
}

func newFakeClient(id string) *fakeClient {
	return &fakeClient{id: id, buf: make(chan Event, 8)}
}

func (f *fakeClient) ID() string { return f.id }

func (f *fakeClient) Send(evt Event) bool {
	if f.full {
		return false
	}
	select {
	case f.buf <- evt:
		return true
	default:
		return false
	}
}

func TestPublishDeliversOnlyToTopicSubscribers(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	b := newFakeClient("b")
	h.Subscribe("topic1", a)
	h.Subscribe("topic2", b)

	n := h.Publish("topic1", Event{Type: "test"})
	if n != 1 {
		t.Fatalf("expected 1 delivery, got %d", n)
	}
	select {
	case evt := <-a.buf:
		if evt.Topic != "topic1" {
			t.Errorf("expected topic %q, got %q", "topic1", evt.Topic)
		}
	default:
		t.Fatal("client a did not receive the event")
	}
	select {
	case <-b.buf:
		t.Fatal("client b should not have received the event")
	default:
	}
}

func TestPublishFansOutToMultipleSubscribers(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	b := newFakeClient("b")
	h.Subscribe("topic1", a)
	h.Subscribe("topic1", b)

	n := h.Publish("topic1", Event{Type: "test"})
	if n != 2 {
		t.Fatalf("expected 2 deliveries, got %d", n)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	h.Subscribe("topic1", a)
	h.Unsubscribe("topic1", a)

	n := h.Publish("topic1", Event{Type: "test"})
	if n != 0 {
		t.Fatalf("expected 0 deliveries after unsubscribe, got %d", n)
	}
}

func TestUnsubscribeAllRemovesFromEveryTopic(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	h.Subscribe("topic1", a)
	h.Subscribe("topic2", a)
	h.UnsubscribeAll(a)

	if n := h.Publish("topic1", Event{}); n != 0 {
		t.Errorf("topic1: expected 0 deliveries, got %d", n)
	}
	if n := h.Publish("topic2", Event{}); n != 0 {
		t.Errorf("topic2: expected 0 deliveries, got %d", n)
	}
	stats := h.Stats()
	if stats.TopicCount != 0 {
		t.Errorf("expected empty topics map after last subscriber leaves, got %d", stats.TopicCount)
	}
}

func TestPublishDropsDeadClients(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	a.full = true
	h.Subscribe("topic1", a)

	n := h.Publish("topic1", Event{Type: "test"})
	if n != 0 {
		t.Fatalf("expected 0 deliveries to a full client, got %d", n)
	}
	stats := h.Stats()
	if stats.TopicCount != 0 {
		t.Errorf("expected dead client to be dropped from the topic, topic count = %d", stats.TopicCount)
	}
}

func TestStatsCountsDistinctClientsAcrossTopics(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	h.Subscribe("topic1", a)
	h.Subscribe("topic2", a)
	b := newFakeClient("b")
	h.Subscribe("topic1", b)

	stats := h.Stats()
	if stats.TopicCount != 2 {
		t.Errorf("expected 2 topics, got %d", stats.TopicCount)
	}
	if stats.ClientCount != 2 {
		t.Errorf("expected 2 distinct clients, got %d", stats.ClientCount)
	}
	if stats.Topics["topic1"] != 2 {
		t.Errorf("expected 2 subscribers on topic1, got %d", stats.Topics["topic1"])
	}
	if stats.Topics["topic2"] != 1 {
		t.Errorf("expected 1 subscriber on topic2, got %d", stats.Topics["topic2"])
	}
}

func TestSubscribeIsIdempotent(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	h.Subscribe("topic1", a)
	h.Subscribe("topic1", a)

	stats := h.Stats()
	if stats.Topics["topic1"] != 1 {
		t.Errorf("expected 1 subscriber after duplicate Subscribe, got %d", stats.Topics["topic1"])
	}
}

func TestPublishSetsTopicAndTimestampWhenMissing(t *testing.T) {
	h := NewHub()
	a := newFakeClient("a")
	h.Subscribe("topic1", a)

	h.Publish("topic1", Event{Type: "test"})
	evt := <-a.buf
	if evt.Topic != "topic1" {
		t.Errorf("expected Topic to be set to %q, got %q", "topic1", evt.Topic)
	}
	if evt.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}
