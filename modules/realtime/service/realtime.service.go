package service

import (
	realtimeContract "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/realtime"
)

// UserTopicPrefix namespaces every user's private WebSocket/SSE channel so
// it can't collide with an arbitrary topic name passed to
// POST /v1/admin/realtime/publish or a client's own subscribe message.
const UserTopicPrefix = "user:"

// UserTopic returns the hub topic a given user's connections auto-subscribe
// to. Exported so the handler package (WebSocket/SSE upgrade) and this
// service (PublishToUser) agree on the same convention.
func UserTopic(userID string) string {
	return UserTopicPrefix + userID
}

// RealtimeService wraps pkg/realtime.Hub with the domain conventions
// (per-user private topics) other modules and the HTTP handler rely on.
type RealtimeService struct {
	hub *realtime.Hub
}

func New(hub *realtime.Hub) *RealtimeService {
	return &RealtimeService{hub: hub}
}

func (s *RealtimeService) PublishToUser(userID string, evt realtimeContract.Event) (int, error) {
	return s.hub.Publish(UserTopic(userID), toHubEvent(evt)), nil
}

func (s *RealtimeService) PublishToTopic(topic string, evt realtimeContract.Event) (int, error) {
	return s.hub.Publish(topic, toHubEvent(evt)), nil
}

func (s *RealtimeService) Stats() realtimeContract.Stats {
	stats := s.hub.Stats()
	return realtimeContract.Stats{
		TopicCount:  stats.TopicCount,
		ClientCount: stats.ClientCount,
		Topics:      stats.Topics,
	}
}

func toHubEvent(evt realtimeContract.Event) realtime.Event {
	return realtime.Event{
		Type:  evt.Type,
		Title: evt.Title,
		Body:  evt.Body,
		Data:  evt.Data,
	}
}

var _ realtimeContract.Service = (*RealtimeService)(nil)
