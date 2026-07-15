package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime/dto"
	realtimeSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/realtime/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/realtime"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

const (
	clientSendBuffer = 16
	wsPingInterval   = 30 * time.Second
	sseKeepAlive     = 20 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// The web test client (and any other CORS origin) is already allowed
	// blanket access to the HTTP API (see apps/api/app/app.go's cors.Config
	// AllowOrigins: []string{"*"}) — mirror that here rather than rejecting
	// the WebSocket handshake on origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// RealtimeHandler serves live WebSocket/SSE connections plus the admin
// pub/sub test + stats endpoints. It talks to the hub directly (not through
// the realtime.Service contract) because Subscribe/Unsubscribe are
// connection-lifecycle concerns internal to this module — other modules
// only ever need Publish, which is what the contract exposes.
type RealtimeHandler struct {
	hub *realtime.Hub
	log *zap.Logger
}

func New(hub *realtime.Hub, log *zap.Logger) *RealtimeHandler {
	return &RealtimeHandler{hub: hub, log: log}
}

// RegisterRoutes mounts the realtime endpoints. authMw is the standard
// header-only JWT middleware (used for the admin routes, reached via normal
// fetch calls); authWSMw additionally accepts ?token= since neither the
// browser WebSocket nor EventSource API can set an Authorization header.
func (h *RealtimeHandler) RegisterRoutes(rg *gin.RouterGroup, authMw, authWSMw, adminMw gin.HandlerFunc) {
	me := rg.Group("/me", authWSMw)
	me.GET("/ws", h.WebSocket)
	me.GET("/events", h.SSE)

	admin := rg.Group("/admin/realtime", authMw, adminMw)
	admin.POST("/publish", h.Publish)
	admin.GET("/stats", h.Stats)
}

// WebSocket godoc
// @Summary     Open a live WebSocket connection
// @Description Upgrades to WebSocket and auto-subscribes to the caller's private notification channel. Send {"action":"subscribe","topic":"..."} or {"action":"unsubscribe","topic":"..."} JSON frames to join/leave additional arbitrary topics for the generic pub/sub demo. Auth: Authorization header or ?token= query param (browsers can't set headers on the WebSocket handshake).
// @Tags        realtime
// @Security    BearerAuth
// @Param       token query string false "Access token (browser WebSocket clients can't set the Authorization header)"
// @Success     101 {string} string "Switching Protocols"
// @Failure     401 {object} response.ErrorEnvelope
// @Router      /v1/me/ws [get]
func (h *RealtimeHandler) WebSocket(c *gin.Context) {
	userID := c.GetString(middleware.CtxUserID)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("realtime: websocket upgrade failed", zap.Error(err))
		return
	}

	client := newChanClient(clientSendBuffer)
	h.hub.Subscribe(realtimeSvc.UserTopic(userID), client)

	go h.wsWritePump(conn, client)
	h.wsReadPump(conn, client)
}

func (h *RealtimeHandler) wsWritePump(conn *websocket.Conn, client *chanClient) {
	ticker := time.NewTicker(wsPingInterval)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case evt, ok := <-client.send:
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteJSON(evt); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// wsControlMessage is a client->server frame for the generic pub/sub demo —
// join/leave an arbitrary topic beyond the auto-subscribed private channel.
type wsControlMessage struct {
	Action string `json:"action"` // "subscribe" | "unsubscribe"
	Topic  string `json:"topic"`
}

func (h *RealtimeHandler) wsReadPump(conn *websocket.Conn, client *chanClient) {
	defer func() {
		h.hub.UnsubscribeAll(client)
		client.Close()
	}()
	conn.SetReadLimit(4096)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var ctrl wsControlMessage
		if err := json.Unmarshal(msg, &ctrl); err != nil || ctrl.Topic == "" {
			continue
		}
		switch ctrl.Action {
		case "subscribe":
			h.hub.Subscribe(ctrl.Topic, client)
		case "unsubscribe":
			h.hub.Unsubscribe(ctrl.Topic, client)
		}
	}
}

// SSE godoc
// @Summary     Open a live Server-Sent Events stream
// @Description Streams events as text/event-stream. Defaults to the caller's private notification channel; pass ?topic=<name> to instead watch an arbitrary topic (the SSE protocol is one-directional, so unlike /v1/me/ws the topic is fixed for the connection's lifetime). Auth: Authorization header or ?token= query param (browsers' EventSource can't set headers).
// @Tags        realtime
// @Security    BearerAuth
// @Param       token query string false "Access token (EventSource can't set the Authorization header)"
// @Param       topic query string false "Topic to watch instead of the caller's private channel"
// @Produce     text/event-stream
// @Success     200 {string} string "text/event-stream"
// @Failure     401 {object} response.ErrorEnvelope
// @Router      /v1/me/events [get]
func (h *RealtimeHandler) SSE(c *gin.Context) {
	userID := c.GetString(middleware.CtxUserID)
	topic := c.Query("topic")
	if topic == "" {
		topic = realtimeSvc.UserTopic(userID)
	}

	client := newChanClient(clientSendBuffer)
	h.hub.Subscribe(topic, client)
	defer func() {
		h.hub.UnsubscribeAll(client)
		client.Close()
	}()

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(sseKeepAlive)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case evt, ok := <-client.send:
			if !ok {
				return false
			}
			evtType := evt.Type
			if evtType == "" {
				evtType = "message"
			}
			c.SSEvent(evtType, evt)
			return true
		case <-ticker.C:
			c.SSEvent("keepalive", "ping")
			return true
		}
	})
}

// Publish godoc
// @Summary     Publish an event to an arbitrary topic (admin diagnostic)
// @Description Generic pub/sub test endpoint, independent of the notification system — delivers to every WebSocket/SSE client currently subscribed to Topic (including ones that joined via a WebSocket {"action":"subscribe",...} frame or ?topic= SSE query param). Returns how many connections actually received it.
// @Tags        realtime
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       body body dto.PublishRequest true "Topic + event payload"
// @Success     200 {object} dto.PublishResponse
// @Failure     422 {object} response.ErrorEnvelope
// @Router      /v1/admin/realtime/publish [post]
func (h *RealtimeHandler) Publish(c *gin.Context) {
	var req dto.PublishRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	n := h.hub.Publish(req.Topic, realtime.Event{
		Type:  req.Type,
		Title: req.Title,
		Body:  req.Body,
		Data:  req.Data,
	})
	response.OKI18n(c, "realtime.publishSuccess", dto.PublishResponse{Delivered: n})
}

// Stats godoc
// @Summary     Current WebSocket/SSE connection and topic occupancy
// @Tags        realtime
// @Security    BearerAuth
// @Produce     json
// @Success     200 {object} dto.StatsResponse
// @Router      /v1/admin/realtime/stats [get]
func (h *RealtimeHandler) Stats(c *gin.Context) {
	s := h.hub.Stats()
	response.OKI18n(c, "realtime.statsFetched", dto.StatsResponse{
		TopicCount:  s.TopicCount,
		ClientCount: s.ClientCount,
		Topics:      s.Topics,
	})
}
