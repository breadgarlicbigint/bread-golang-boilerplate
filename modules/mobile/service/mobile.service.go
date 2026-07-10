package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/redis/go-redis/v9"
	actSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/activity/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/mobile/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/sms"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	col            = "user_mobiles"
	otpPrefix      = "otp:"
	otpTTL         = 10 * time.Minute
	otpMaxAttempts = 5
	otpLength      = 6
)

var (
	ErrMobileAlreadyVerified = errors.New(409, "MOBILE_ALREADY_VERIFIED", "This phone number is already verified")
	ErrOTPExpiredOrInvalid   = errors.New(400, "OTP_INVALID", "OTP is invalid or has expired")
	ErrOTPMaxAttempts        = errors.New(429, "OTP_MAX_ATTEMPTS", "Too many incorrect attempts. Please request a new code.")
)

type MobileService struct {
	col      *mongo.Collection
	rdb      *redis.Client
	sender   *sms.Sender
	hasher   *hash.Hasher
	activity *actSvc.ActivityService
}

func New(db *database.MongoDB, rdb *redis.Client, sender *sms.Sender, hasher *hash.Hasher, activity *actSvc.ActivityService) *MobileService {
	return &MobileService{
		col:      db.Collection(col),
		rdb:      rdb,
		sender:   sender,
		hasher:   hasher,
		activity: activity,
	}
}

func (s *MobileService) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetName("idx_user_id")},
		{Keys: bson.D{{Key: "e164", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true).SetName("idx_e164_unique")},
		{Keys: bson.D{{Key: "isVerified", Value: 1}}, Options: options.Index().SetName("idx_verified")},
	}
	_, err := s.col.Indexes().CreateMany(ctx, models)
	return err
}

// SendOTP generates an OTP and sends it via the chosen channel (sms or whatsapp).
func (s *MobileService) SendOTP(ctx context.Context, userID uuid.UUID, e164, correlationID string, channel sms.Channel) error {
	// Check if already verified by another user
	count, _ := s.col.CountDocuments(ctx, bson.M{"e164": e164, "isVerified": true})
	if count > 0 {
		return ErrMobileAlreadyVerified
	}

	code, err := generateOTP(otpLength)
	if err != nil {
		return err
	}

	codeHash, err := s.hasher.Hash(code)
	if err != nil {
		return err
	}

	otpData := entity.MobileOTP{
		E164:      e164,
		CodeHash:  codeHash,
		Attempts:  0,
		ExpiresAt: time.Now().Add(otpTTL),
	}
	b, _ := json.Marshal(otpData)
	if err := s.rdb.Set(ctx, otpPrefix+e164, b, otpTTL).Err(); err != nil {
		return err
	}

	// Send via Twilio
	sendErr := s.sender.SendOTP(ctx, e164, code, channel)
	success := sendErr == nil
	errMsg := ""
	if !success {
		errMsg = sendErr.Error()
	}

	// Log outbound SMS/WhatsApp
	channelName := string(channel)
	if channel == sms.ChannelWhatsApp {
		s.activity.LogOutbound(ctx, actSvc.ActivityLog{
			CorrelationID: correlationID,
			ActorID:       userID.String(),
			ActorType:     "system",
			Action:        actSvc.ActionOutWhatsAppSent,
			Channel:       channelName,
			Recipient:     e164,
			Success:       &success,
			ErrorMessage:  errMsg,
		})
	} else {
		s.activity.LogSMSSent(ctx, correlationID, userID.String(), e164, success, errMsg)
	}

	return sendErr
}

// VerifyOTP validates the submitted code and marks the number as verified.
func (s *MobileService) VerifyOTP(ctx context.Context, userID uuid.UUID, e164, code string) error {
	key := otpPrefix + e164
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return ErrOTPExpiredOrInvalid
	}

	var otpData entity.MobileOTP
	if err := json.Unmarshal(raw, &otpData); err != nil {
		return errors.ErrInternal
	}

	if otpData.Attempts >= otpMaxAttempts {
		_ = s.rdb.Del(ctx, key)
		return ErrOTPMaxAttempts
	}

	if !s.hasher.Compare(code, otpData.CodeHash) {
		otpData.Attempts++
		b, _ := json.Marshal(otpData)
		_ = s.rdb.Set(ctx, key, b, otpTTL)
		return ErrOTPExpiredOrInvalid
	}

	// Code correct — consume it
	_ = s.rdb.Del(ctx, key)

	// Upsert the verified mobile record
	now := time.Now()
	filter := bson.M{"e164": e164}
	update := bson.M{
		"$set": bson.M{
			"userId":     userID,
			"e164":       e164,
			"isVerified": true,
			"verifiedAt": now,
			"updatedAt":  now,
		},
		"$setOnInsert": bson.M{
			"_id":       uuid.New(),
			"createdAt": now,
			"isPrimary": false,
		},
	}
	_, err = s.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
	return err
}

// ListByUser returns all mobile numbers for a user.
func (s *MobileService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*entity.UserMobile, error) {
	cur, err := s.col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var mobiles []*entity.UserMobile
	return mobiles, cur.All(ctx, &mobiles)
}

// SetPrimary marks one mobile number as the primary for a user.
func (s *MobileService) SetPrimary(ctx context.Context, userID uuid.UUID, e164 string) error {
	_, err := s.col.UpdateMany(ctx, bson.M{"userId": userID}, bson.M{"$set": bson.M{"isPrimary": false}})
	if err != nil {
		return err
	}
	_, err = s.col.UpdateOne(ctx,
		bson.M{"userId": userID, "e164": e164},
		bson.M{"$set": bson.M{"isPrimary": true, "updatedAt": time.Now()}})
	return err
}

// Delete removes a mobile number.
func (s *MobileService) Delete(ctx context.Context, userID uuid.UUID, e164 string) error {
	_, err := s.col.DeleteOne(ctx, bson.M{"userId": userID, "e164": e164})
	return err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func generateOTP(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}
