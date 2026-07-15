package dto

// PublishRequest is the admin generic pub/sub test payload — publishes an
// arbitrary event to an arbitrary topic, independent of the notification
// system. Any WebSocket/SSE client that has subscribed to Topic receives it.
type PublishRequest struct {
	Topic string         `json:"topic" validate:"required"`
	Type  string         `json:"type"  validate:"required"`
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Data  map[string]any `json:"data"`
}

type PublishResponse struct {
	Delivered int `json:"delivered"`
}

type StatsResponse struct {
	TopicCount  int            `json:"topicCount"`
	ClientCount int            `json:"clientCount"`
	Topics      map[string]int `json:"topics"`
}
