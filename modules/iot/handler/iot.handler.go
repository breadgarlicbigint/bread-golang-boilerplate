package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	iotDTO "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot/dto"
	iotEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/iot/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
)

type IoTSvc interface {
	SimulateTelemetry(ctx context.Context, deviceID, metric string, value *float64, unit string) error
	SendCommand(ctx context.Context, deviceID, command string, data map[string]any) error
	ListTelemetry(ctx context.Context, deviceID string, page, perPage int) ([]*iotEntity.DeviceTelemetry, int64, error)
}

type IoTHandler struct {
	svc IoTSvc
}

func New(svc IoTSvc) *IoTHandler {
	return &IoTHandler{svc: svc}
}

// RegisterRoutes mounts all IoT endpoints — admin-only, this is a
// diagnostic/demo surface, not a device-facing API.
func (h *IoTHandler) RegisterRoutes(rg *gin.RouterGroup, authMw, adminMw gin.HandlerFunc) {
	admin := rg.Group("/admin/iot", authMw, adminMw)
	admin.POST("/devices/:deviceId/simulate", h.Simulate)
	admin.POST("/devices/:deviceId/command", h.Command)
	admin.GET("/telemetry", h.ListTelemetry)
}

// Simulate godoc
// @Summary     Simulate a device telemetry reading over MQTT
// @Description Publishes to MQTT topic devices/:deviceId/telemetry exactly as a real device would. A subscriber running in this API process picks it up, persists it, and forwards it onto the realtime "iot:telemetry" topic — watch it arrive live on the Realtime page while this call returns immediately (publish is fire-and-forget; the round trip is asynchronous).
// @Tags        iot
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       deviceId path string true "Device ID"
// @Param       body body iotDTO.SimulateRequest true "Metric + optional value/unit"
// @Success     200 {object} iotDTO.SimulateResponse
// @Failure     422 {object} response.ErrorEnvelope
// @Failure     503 {object} response.ErrorEnvelope "MQTT not configured"
// @Router      /v1/admin/iot/devices/{deviceId}/simulate [post]
func (h *IoTHandler) Simulate(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req iotDTO.SimulateRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SimulateTelemetry(c.Request.Context(), deviceID, req.Metric, req.Value, req.Unit); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "iot.simulateSuccess", iotDTO.SimulateResponse{Published: true})
}

// Command godoc
// @Summary     Publish a command to a device over MQTT
// @Description Publishes to MQTT topic devices/:deviceId/commands. Fire-and-forget — nothing in this boilerplate subscribes to it (a real device would); this exists to demonstrate publishing in the other direction.
// @Tags        iot
// @Security    BearerAuth
// @Accept      json
// @Produce     json
// @Param       deviceId path string true "Device ID"
// @Param       body body iotDTO.CommandRequest true "Command + optional data"
// @Success     200 {object} iotDTO.CommandResponse
// @Failure     422 {object} response.ErrorEnvelope
// @Failure     503 {object} response.ErrorEnvelope "MQTT not configured"
// @Router      /v1/admin/iot/devices/{deviceId}/command [post]
func (h *IoTHandler) Command(c *gin.Context) {
	deviceID := c.Param("deviceId")
	var req iotDTO.CommandRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	if err := h.svc.SendCommand(c.Request.Context(), deviceID, req.Command, req.Data); err != nil {
		response.HandleAppError(c, err)
		return
	}
	response.OKI18n(c, "iot.commandSuccess", iotDTO.CommandResponse{Published: true})
}

// ListTelemetry godoc
// @Summary     List persisted device telemetry readings
// @Description Paginated, most recent first. Optionally filter to one device with ?deviceId=. This is the durable record of what MQTT delivered — the realtime WS/SSE push is "while you're watching", this is "what did I miss".
// @Tags        iot
// @Security    BearerAuth
// @Produce     json
// @Param       deviceId query string false "Filter to one device"
// @Param       page     query int    false "Page number"
// @Param       perPage  query int    false "Items per page"
// @Success     200 {array} iotDTO.TelemetryResponse
// @Router      /v1/admin/iot/telemetry [get]
func (h *IoTHandler) ListTelemetry(c *gin.Context) {
	q := pagination.FromContext(c)
	deviceID := c.Query("deviceId")

	readings, total, err := h.svc.ListTelemetry(c.Request.Context(), deviceID, q.Page, q.PerPage)
	if err != nil {
		response.HandleAppError(c, err)
		return
	}
	meta := q.BuildMeta(total)
	resp := make([]iotDTO.TelemetryResponse, len(readings))
	for i, r := range readings {
		resp[i] = iotDTO.TelemetryResponse{
			ID:         r.ID.String(),
			DeviceID:   r.DeviceID,
			Metric:     r.Metric,
			Value:      r.Value,
			Unit:       r.Unit,
			RecordedAt: r.RecordedAt.UTC().String(),
		}
	}
	response.OKWithMetaI18n(c, "iot.telemetryListed", resp, &response.Meta{
		Total: meta.Total, Page: meta.Page, PerPage: meta.PerPage,
		TotalPage: meta.TotalPage, HasNext: meta.HasNext, HasPrev: meta.HasPrev,
	})
}
