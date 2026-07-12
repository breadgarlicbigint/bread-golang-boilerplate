package service

import (
	"context"
	"fmt"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	notifEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/notification/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/email"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	notifCol   = "notifications"
	deviceCol  = "device_tokens"
	prefsCol   = "notification_preferences"
)

type NotificationService struct {
	notifCol  *mongo.Collection
	deviceCol *mongo.Collection
	prefsCol  *mongo.Collection
	fcm       *FCMSender
	mailer    *email.Mailer
	log       *zap.Logger
}

func New(db *database.MongoDB, fcm *FCMSender, mailer *email.Mailer, log *zap.Logger) *NotificationService {
	return &NotificationService{
		notifCol:  db.Collection(notifCol),
		deviceCol: db.Collection(deviceCol),
		prefsCol:  db.Collection(prefsCol),
		fcm:       fcm,
		mailer:    mailer,
		log:       log,
	}
}

func (s *NotificationService) EnsureIndexes(ctx context.Context) error {
	notifIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "isRead", Value: 1}}, Options: options.Index().SetName("idx_user_read")},
		{Keys: bson.D{{Key: "userId", Value: 1}, {Key: "createdAt", Value: -1}}, Options: options.Index().SetName("idx_user_created")},
		{Keys: bson.D{{Key: "type", Value: 1}}, Options: options.Index().SetName("idx_type")},
		// TTL: auto-delete notifications older than 90 days
		{Keys: bson.D{{Key: "createdAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(7776000).SetName("idx_ttl_90d")},
	}
	deviceIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetName("idx_user_id")},
		{Keys: bson.D{{Key: "token", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_token_unique")},
	}
	prefsIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_user_id")},
	}

	for _, pair := range []struct {
		col    *mongo.Collection
		models []mongo.IndexModel
	}{
		{s.notifCol, notifIdx},
		{s.deviceCol, deviceIdx},
		{s.prefsCol, prefsIdx},
	} {
		if _, err := pair.col.Indexes().CreateMany(ctx, pair.models); err != nil {
			return err
		}
	}
	return nil
}

// ── Send ──────────────────────────────────────────────────────────────────────

// Send dispatches a notification through the requested channel, checking preferences.
func (s *NotificationService) Send(ctx context.Context, req dto.SendRequest) error {
	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return errors.ErrBadRequest
	}

	// Check user preferences before sending
	if !s.isChannelEnabled(ctx, userID, req.Type, req.Channel) {
		s.log.Debug("notification suppressed by user preference",
			zap.String("userId", req.UserID),
			zap.String("channel", string(req.Channel)),
			zap.String("type", string(req.Type)))
		return nil
	}

	notif := &notifEntity.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      req.Type,
		Channel:   req.Channel,
		Status:    notifEntity.StatusPending,
		Title:     req.Title,
		Body:      req.Body,
		ImageURL:  req.ImageURL,
		Data:      req.Data,
		ActionURL: req.ActionURL,
		IsRead:    false,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var sendErr error
	switch req.Channel {
	case notifEntity.ChannelPush:
		sendErr = s.sendPush(ctx, userID, notif)
	case notifEntity.ChannelEmail:
		sendErr = s.sendEmail(ctx, userID, notif)
	case notifEntity.ChannelInApp:
		// In-app: just persist — client polls or uses WebSocket
		sendErr = nil
	case notifEntity.ChannelSilent:
		sendErr = s.sendSilentPush(ctx, userID, notif)
	default:
		sendErr = fmt.Errorf("unsupported channel: %s", req.Channel)
	}

	if sendErr != nil {
		notif.Status = notifEntity.StatusFailed
		notif.FailReason = sendErr.Error()
		s.log.Warn("notification send failed",
			zap.String("channel", string(req.Channel)),
			zap.String("userId", req.UserID),
			zap.Error(sendErr))
	} else {
		notif.Status = notifEntity.StatusSent
		now := time.Now()
		notif.SentAt = &now
	}

	// Persist in-app + failed notifications for audit
	if req.Channel == notifEntity.ChannelInApp || notif.Status == notifEntity.StatusFailed {
		_, _ = s.notifCol.InsertOne(ctx, notif)
	}
	return sendErr
}

// Broadcast sends to multiple users concurrently (e.g. admin announcements).
func (s *NotificationService) Broadcast(ctx context.Context, req dto.BroadcastRequest) (int, int, error) {
	var success, failed int
	for _, uid := range req.UserIDs {
		err := s.Send(ctx, dto.SendRequest{
			UserID:  uid,
			Type:    req.Type,
			Channel: req.Channel,
			Title:   req.Title,
			Body:    req.Body,
			Data:    req.Data,
		})
		if err != nil {
			failed++
		} else {
			success++
		}
	}
	return success, failed, nil
}

// ── In-app notification queries ───────────────────────────────────────────────

// ListByUser returns paginated in-app notifications for a user.
func (s *NotificationService) ListByUser(ctx context.Context, userID uuid.UUID, page, perPage int, unreadOnly bool) ([]*notifEntity.Notification, int64, error) {
	filter := bson.M{"userId": userID, "channel": notifEntity.ChannelInApp}
	if unreadOnly {
		filter["isRead"] = false
	}

	total, _ := s.notifCol.CountDocuments(ctx, filter)
	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetSkip(int64((page - 1) * perPage)).
		SetLimit(int64(perPage))

	cur, err := s.notifCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	var notifs []*notifEntity.Notification
	return notifs, total, cur.All(ctx, &notifs)
}

// UnreadCount returns the count of unread in-app notifications.
func (s *NotificationService) UnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.notifCol.CountDocuments(ctx, bson.M{
		"userId":  userID,
		"channel": notifEntity.ChannelInApp,
		"isRead":  false,
	})
}

// MarkRead marks one notification as read.
func (s *NotificationService) MarkRead(ctx context.Context, notifID, userID uuid.UUID) error {
	now := time.Now()
	_, err := s.notifCol.UpdateOne(ctx,
		bson.M{"_id": notifID, "userId": userID},
		bson.M{"$set": bson.M{"isRead": true, "readAt": now, "status": notifEntity.StatusRead, "updatedAt": now}})
	return err
}

// MarkAllRead marks all unread in-app notifications for a user as read.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	_, err := s.notifCol.UpdateMany(ctx,
		bson.M{"userId": userID, "channel": notifEntity.ChannelInApp, "isRead": false},
		bson.M{"$set": bson.M{"isRead": true, "readAt": now, "status": notifEntity.StatusRead, "updatedAt": now}})
	return err
}

// ── Device tokens ─────────────────────────────────────────────────────────────

// RegisterDevice upserts an FCM device token for a user.
func (s *NotificationService) RegisterDevice(ctx context.Context, userID uuid.UUID, req dto.RegisterDeviceRequest) error {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"userId":      userID,
			"platform":    req.Platform,
			"deviceModel": req.DeviceModel,
			"appVersion":  req.AppVersion,
			"isActive":    true,
			"lastSeenAt":  now,
		},
		"$setOnInsert": bson.M{
			"_id":       uuid.New(),
			"token":     req.Token,
			"createdAt": now,
		},
	}
	_, err := s.deviceCol.UpdateOne(ctx,
		bson.M{"token": req.Token},
		update,
		options.Update().SetUpsert(true))
	return err
}

// RemoveDevice deactivates a device token (called on logout or token refresh).
func (s *NotificationService) RemoveDevice(ctx context.Context, userID uuid.UUID, token string) error {
	_, err := s.deviceCol.UpdateOne(ctx,
		bson.M{"userId": userID, "token": token},
		bson.M{"$set": bson.M{"isActive": false}})
	return err
}

// ── Preferences ───────────────────────────────────────────────────────────────

// GetPreferences returns user notification preferences, creating defaults if absent.
func (s *NotificationService) GetPreferences(ctx context.Context, userID uuid.UUID) (*notifEntity.NotificationPreferences, error) {
	var prefs notifEntity.NotificationPreferences
	err := s.prefsCol.FindOne(ctx, bson.M{"userId": userID}).Decode(&prefs)
	if err == mongo.ErrNoDocuments {
		defaults := notifEntity.DefaultPreferences(userID)
		_, _ = s.prefsCol.InsertOne(ctx, defaults)
		return &defaults, nil
	}
	return &prefs, err
}

// UpdatePreferences saves updated channel/type preferences.
func (s *NotificationService) UpdatePreferences(ctx context.Context, userID uuid.UUID, req dto.UpdatePreferencesRequest) error {
	update := bson.M{"updatedAt": time.Now()}
	if req.Channels != nil {
		for k, v := range req.Channels {
			update["channels."+k] = v
		}
	}
	if req.Types != nil {
		for typKey, channels := range req.Types {
			for chKey, v := range channels {
				update["types."+typKey+"."+chKey] = v
			}
		}
	}
	_, err := s.prefsCol.UpdateOne(ctx,
		bson.M{"userId": userID},
		bson.M{"$set": update},
		options.Update().SetUpsert(true))
	return err
}

// ── Internal channel dispatchers ──────────────────────────────────────────────

func (s *NotificationService) sendPush(ctx context.Context, userID uuid.UUID, n *notifEntity.Notification) error {
	if s.fcm == nil {
		return fmt.Errorf("FCM not configured")
	}
	tokens, err := s.activeTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return fmt.Errorf("no active device tokens")
	}

	data := make(map[string]string)
	data["notifId"] = n.ID.String()
	data["type"] = string(n.Type)
	if n.ActionURL != "" {
		data["actionUrl"] = n.ActionURL
	}

	_, stale, err := s.fcm.SendMulticast(ctx, tokens, PushPayload{
		Title:    n.Title,
		Body:     n.Body,
		ImageURL: n.ImageURL,
		Data:     data,
		Silent:   false,
	})
	// Remove stale tokens automatically
	for _, tok := range stale {
		_, _ = s.deviceCol.UpdateOne(ctx, bson.M{"token": tok},
			bson.M{"$set": bson.M{"isActive": false}})
	}
	return err
}

func (s *NotificationService) sendSilentPush(ctx context.Context, userID uuid.UUID, n *notifEntity.Notification) error {
	if s.fcm == nil {
		return fmt.Errorf("FCM not configured")
	}
	tokens, err := s.activeTokens(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return nil // silent — no error if no tokens
	}
	data := map[string]string{"type": string(n.Type), "silent": "true"}
	for k, v := range toStringMap(n.Data) {
		data[k] = v
	}
	_, _, err = s.fcm.SendMulticast(ctx, tokens, PushPayload{Silent: true, Data: data})
	return err
}

func (s *NotificationService) sendEmail(ctx context.Context, userID uuid.UUID, n *notifEntity.Notification) error {
	if s.mailer == nil {
		return fmt.Errorf("mailer not configured")
	}
	// Look up email from a users collection — in real code inject a UserRepo
	// Placeholder: assumes email was passed via Data["email"]
	toEmail, _ := n.Data["email"].(string)
	if toEmail == "" {
		return fmt.Errorf("email not available for user %s", userID.String())
	}
	return s.mailer.Send(ctx, email.Message{
		To:      []string{toEmail},
		Subject: n.Title,
		HTML:    fmt.Sprintf("<h2>%s</h2><p>%s</p>", n.Title, n.Body),
		Text:    n.Body,
	})
}

// SendTestEmail sends a minimal message directly through the configured mail
// transport (SES or SMTP, whichever MAIL_DRIVER selects) and returns the raw
// error unwrapped. It exists purely as an admin diagnostic — every other
// email call site in this codebase treats a nil mailer / send failure as
// "skip silently", which is the right behavior for real user flows but makes
// it impossible to tell whether MAIL_DRIVER is actually working. This is the
// one path that reports the real reason (connection refused, auth failure, …)
// back to the caller.
func (s *NotificationService) SendTestEmail(ctx context.Context, to string) error {
	if s.mailer == nil {
		return fmt.Errorf("mail driver not configured — check MAIL_DRIVER and its credentials in .env")
	}
	return s.mailer.Send(ctx, email.Message{
		To:      []string{to},
		Subject: "Bread Boilerplate — test email",
		HTML:    "<p>This is a test email from the admin test-email endpoint. If you received this, your MAIL_DRIVER configuration is working.</p>",
		Text:    "This is a test email from the admin test-email endpoint. If you received this, your MAIL_DRIVER configuration is working.",
	})
}

func (s *NotificationService) isChannelEnabled(ctx context.Context, userID uuid.UUID, t notifEntity.NotifType, ch notifEntity.NotifChannel) bool {
	prefs, err := s.GetPreferences(ctx, userID)
	if err != nil {
		return true // default allow on error
	}
	// Check global channel toggle
	if enabled, ok := prefs.Channels[string(ch)]; ok && !enabled {
		return false
	}
	// Check per-type channel toggle
	if typePrefs, ok := prefs.Types[string(t)]; ok {
		if enabled, ok := typePrefs[string(ch)]; ok && !enabled {
			return false
		}
	}
	return true
}

func (s *NotificationService) activeTokens(ctx context.Context, userID uuid.UUID) ([]string, error) {
	cur, err := s.deviceCol.Find(ctx, bson.M{"userId": userID, "isActive": true})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var tokens []string
	for cur.Next(ctx) {
		var d notifEntity.DeviceToken
		if cur.Decode(&d) == nil {
			tokens = append(tokens, d.Token)
		}
	}
	return tokens, nil
}

func toStringMap(m map[string]interface{}) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
