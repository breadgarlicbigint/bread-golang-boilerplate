package service

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

const collection = "activity_logs"

// Direction marks whether a log entry is inbound (client→server) or outbound (server→client).
type Direction string

const (
	DirectionInbound  Direction = "inbound"  // user/client initiated action
	DirectionOutbound Direction = "outbound" // system response / side-effect
)

// ActivityLog is a single audit entry stored in MongoDB.
// Bidirectional: Inbound captures WHAT THE USER DID; Outbound captures WHAT THE SYSTEM DID BACK.
// Linked via CorrelationID — both directions of a request share the same correlationId.
type ActivityLog struct {
	ID            uuid.UUID     `bson:"_id,omitempty"    json:"id"`
	Direction     Direction              `bson:"direction"        json:"direction"`
	CorrelationID string                 `bson:"correlationId"    json:"correlationId"` // links inbound↔outbound
	ActorID       string                 `bson:"actorId"          json:"actorId"`
	ActorType     string                 `bson:"actorType"        json:"actorType"` // user | system | api_key | anonymous
	TenantID      string                 `bson:"tenantId"         json:"tenantId,omitempty"`
	Action        string                 `bson:"action"           json:"action"`
	ResourceID    string                 `bson:"resourceId"       json:"resourceId,omitempty"`
	Resource      string                 `bson:"resource"         json:"resource,omitempty"`
	HTTPMethod    string                 `bson:"httpMethod"       json:"httpMethod,omitempty"`
	HTTPPath      string                 `bson:"httpPath"         json:"httpPath,omitempty"`
	HTTPStatus    int                    `bson:"httpStatus"       json:"httpStatus,omitempty"`
	IPAddress     string                 `bson:"ipAddress"        json:"ipAddress,omitempty"`
	UserAgent     string                 `bson:"userAgent"        json:"userAgent,omitempty"`
	// Outbound-specific fields
	Channel       string                 `bson:"channel"          json:"channel,omitempty"`   // email | push | sms | whatsapp | api
	Recipient     string                 `bson:"recipient"        json:"recipient,omitempty"` // email address, phone number, device token
	Success       *bool                  `bson:"success"          json:"success,omitempty"`   // nil = not applicable
	ErrorMessage  string                 `bson:"errorMessage"     json:"errorMessage,omitempty"`
	LatencyMS     int64                  `bson:"latencyMs"        json:"latencyMs,omitempty"`
	Metadata      map[string]interface{} `bson:"metadata"         json:"metadata,omitempty"`
	CreatedAt     time.Time              `bson:"createdAt"        json:"createdAt"`
}

// Common inbound action constants – used across modules for consistent analytics.
const (
	ActionUserLoginCredential       = "userLoginCredential"
	ActionUserLoginGoogle           = "userLoginGoogle"
	ActionUserLoginApple            = "userLoginApple"
	ActionUserLoginGitHub           = "userLoginGitHub"
	ActionUserLoginPasskey          = "userLoginPasskey"
	ActionUserLoginBiometric        = "userLoginBiometric"
	ActionUserLogout                = "userLogout"
	ActionUserRegister              = "userRegister"
	ActionUserForgotPassword        = "userForgotPassword"
	ActionUserResetPassword         = "userResetPassword"
	ActionUserChangePassword        = "userChangePassword"
	ActionUserReachMaxPasswordAttempt = "userReachMaxPasswordAttempt"
	ActionUserBlocked               = "userBlocked"
	ActionUserUnblocked             = "userUnblocked"
	ActionUserDeleted               = "userDeleted"
	ActionUser2FAEnabled            = "userEnable2FA"
	ActionUser2FADisabled           = "userDisable2FA"
	ActionAdminResetTwoFactor       = "adminUserResetTwoFactor"
	ActionPasskeyRegistered         = "passkeyRegistered"
	ActionPasskeyDeleted            = "passkeyDeleted"
	ActionMobileVerified            = "mobileVerified"
	ActionDeviceRefresh             = "userDeviceRefresh"
	ActionApiKeyCreated             = "apiKeyCreated"
	ActionApiKeyRevoked             = "apiKeyRevoked"
	ActionSessionRevoke             = "adminSessionRevoke"

	// Outbound action constants
	ActionOutEmailSent              = "emailSent"
	ActionOutSMSSent                = "smsSent"
	ActionOutWhatsAppSent           = "whatsappSent"
	ActionOutPushSent               = "pushNotificationSent"
	ActionOutWebhookDelivered       = "webhookDelivered"
	ActionOutAPIResponse            = "apiResponse"
)

type ActivityService struct {
	col *mongo.Collection
}

func New(db *database.MongoDB) *ActivityService {
	return &ActivityService{col: db.Collection(collection)}
}

// LogInbound records a user/client initiated action.
func (s *ActivityService) LogInbound(ctx context.Context, e ActivityLog) {
	e.Direction = DirectionInbound
	s.persist(ctx, e)
}

// LogOutbound records a system-initiated side-effect (email sent, push delivered, etc.).
func (s *ActivityService) LogOutbound(ctx context.Context, e ActivityLog) {
	e.Direction = DirectionOutbound
	s.persist(ctx, e)
}

// LogPair logs both directions atomically under the same correlationId.
func (s *ActivityService) LogPair(ctx context.Context, inbound, outbound ActivityLog) {
	s.LogInbound(ctx, inbound)
	s.LogOutbound(ctx, outbound)
}

// LogUserAction is a convenience wrapper for inbound user actions.
func (s *ActivityService) LogUserAction(ctx context.Context, correlationID, actorID, action, resourceID, resource, tenantID, ip, ua string, meta map[string]interface{}) {
	s.LogInbound(ctx, ActivityLog{
		CorrelationID: correlationID,
		ActorID:       actorID,
		ActorType:     "user",
		TenantID:      tenantID,
		Action:        action,
		ResourceID:    resourceID,
		Resource:      resource,
		IPAddress:     ip,
		UserAgent:     ua,
		Metadata:      meta,
	})
}

// LogEmailSent records that an outbound email was dispatched.
func (s *ActivityService) LogEmailSent(ctx context.Context, correlationID, actorID, recipient, subject string, success bool, errMsg string, latencyMS int64) {
	ok := success
	s.LogOutbound(ctx, ActivityLog{
		CorrelationID: correlationID,
		ActorID:       actorID,
		ActorType:     "system",
		Action:        ActionOutEmailSent,
		Channel:       "email",
		Recipient:     recipient,
		Success:       &ok,
		ErrorMessage:  errMsg,
		LatencyMS:     latencyMS,
		Metadata:      map[string]interface{}{"subject": subject},
	})
}

// LogSMSSent records that an outbound SMS was dispatched.
func (s *ActivityService) LogSMSSent(ctx context.Context, correlationID, actorID, phone string, success bool, errMsg string) {
	ok := success
	s.LogOutbound(ctx, ActivityLog{
		CorrelationID: correlationID,
		ActorID:       actorID,
		ActorType:     "system",
		Action:        ActionOutSMSSent,
		Channel:       "sms",
		Recipient:     phone,
		Success:       &ok,
		ErrorMessage:  errMsg,
	})
}

// LogAPIResponse records the system's HTTP response for a request.
func (s *ActivityService) LogAPIResponse(ctx context.Context, correlationID, actorID, method, path string, status int, latencyMS int64) {
	s.LogOutbound(ctx, ActivityLog{
		CorrelationID: correlationID,
		ActorID:       actorID,
		ActorType:     "system",
		Action:        ActionOutAPIResponse,
		Channel:       "api",
		HTTPMethod:    method,
		HTTPPath:      path,
		HTTPStatus:    status,
		LatencyMS:     latencyMS,
	})
}

func (s *ActivityService) persist(ctx context.Context, e ActivityLog) {
	e.ID = uuid.New()
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	_, _ = s.col.InsertOne(ctx, e)
}
