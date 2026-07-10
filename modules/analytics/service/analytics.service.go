package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/analytics/dto"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

type AnalyticsService struct {
	db *database.MongoDB
}

func New(db *database.MongoDB) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// ── User analytics ────────────────────────────────────────────────────────────

// UserRegistrations returns daily/weekly/monthly new user counts in the period.
func (s *AnalyticsService) UserRegistrations(ctx context.Context, start, end time.Time, granularity string) (*dto.UserRegistrationStats, error) {
	groupID := granularityGroup(granularity, "createdAt")

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"createdAt": bson.M{"$gte": start, "$lt": end},
			"deletedAt": nil,
		}}},
		{{Key: "$group", Value: bson.M{"_id": groupID, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id.year", Value: 1}, {Key: "_id.month", Value: 1}, {Key: "_id.day", Value: 1}}}},
	}

	cur, err := s.db.Collection("users").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var series []dto.TimePoint
	var total int64
	for cur.Next(ctx) {
		var r struct {
			ID    bson.M `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&r); err != nil {
			continue
		}
		series = append(series, dto.TimePoint{Date: formatPeriodID(r.ID, granularity), Count: r.Count})
		total += r.Count
	}

	return &dto.UserRegistrationStats{Series: series, Total: total}, nil
}

// UserChurn returns churn rate for the period.
func (s *AnalyticsService) UserChurn(ctx context.Context, start, end time.Time) (*dto.UserChurnStats, error) {
	col := s.db.Collection("users")
	churned, _ := col.CountDocuments(ctx, bson.M{"deletedAt": bson.M{"$gte": start, "$lt": end}})
	totalEver, _ := col.CountDocuments(ctx, bson.M{"createdAt": bson.M{"$lt": end}})

	rate := 0.0
	if totalEver > 0 {
		rate = math.Round(float64(churned)/float64(totalEver)*10000) / 100
	}
	return &dto.UserChurnStats{ChurnRate: rate, Churned: churned, Total: totalEver}, nil
}

// SignupMethodBreakdown returns share by signUpWith field.
func (s *AnalyticsService) SignupMethodBreakdown(ctx context.Context) ([]dto.SignupMethodBreakdown, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"deletedAt": nil}}},
		{{Key: "$group", Value: bson.M{"_id": "$signUpWith", "count": bson.M{"$sum": 1}}}},
		{{Key: "$group", Value: bson.M{
			"_id":     nil,
			"total":   bson.M{"$sum": "$count"},
			"methods": bson.M{"$push": bson.M{"method": "$_id", "count": "$count"}},
		}}},
		{{Key: "$unwind", Value: "$methods"}},
		{{Key: "$project", Value: bson.M{
			"method":  "$methods.method",
			"count":   "$methods.count",
			"percent": bson.M{"$multiply": bson.A{bson.M{"$divide": bson.A{"$methods.count", "$total"}}, 100}},
		}}},
	}
	return aggregateToSlice[dto.SignupMethodBreakdown](ctx, s.db.Collection("users"), pipeline)
}

// BlockedUserTrend returns daily blocked user events from activity logs.
func (s *AnalyticsService) BlockedUserTrend(ctx context.Context, start, end time.Time) ([]dto.TimePoint, error) {
	groupID := granularityGroup("day", "createdAt")
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"action": "userBlocked", "createdAt": bson.M{"$gte": start, "$lt": end}}}},
		{{Key: "$group", Value: bson.M{"_id": groupID, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id.year", Value: 1}, {Key: "_id.month", Value: 1}, {Key: "_id.day", Value: 1}}}},
	}
	return timeSeriesFromPipeline(ctx, s.db.Collection("activity_logs"), pipeline, "day")
}

// ── Auth / Session analytics ──────────────────────────────────────────────────

// LoginFrequency returns daily login counts across all methods.
func (s *AnalyticsService) LoginFrequency(ctx context.Context, start, end time.Time, granularity string) ([]dto.TimePoint, error) {
	loginActions := bson.A{"userLoginCredential", "userLoginGoogle", "userLoginApple", "userLoginGitHub", "userLoginPasskey", "userLoginBiometric"}
	groupID := granularityGroup(granularity, "createdAt")
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"action": bson.M{"$in": loginActions}, "createdAt": bson.M{"$gte": start, "$lt": end}}}},
		{{Key: "$group", Value: bson.M{"_id": groupID, "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id.year", Value: 1}, {Key: "_id.month", Value: 1}, {Key: "_id.day", Value: 1}}}},
	}
	return timeSeriesFromPipeline(ctx, s.db.Collection("activity_logs"), pipeline, granularity)
}

// LoginMethodBreakdown returns login counts per method.
func (s *AnalyticsService) LoginMethodBreakdown(ctx context.Context, start, end time.Time) ([]dto.LoginMethodBreakdown, error) {
	loginActions := bson.A{"userLoginCredential", "userLoginGoogle", "userLoginApple", "userLoginGitHub", "userLoginPasskey", "userLoginBiometric"}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"action": bson.M{"$in": loginActions}, "createdAt": bson.M{"$gte": start, "$lt": end}}}},
		{{Key: "$group", Value: bson.M{"_id": "$action", "count": bson.M{"$sum": 1}}}},
		{{Key: "$group", Value: bson.M{
			"_id":     nil,
			"total":   bson.M{"$sum": "$count"},
			"methods": bson.M{"$push": bson.M{"method": "$_id", "count": "$count"}},
		}}},
		{{Key: "$unwind", Value: "$methods"}},
		{{Key: "$project", Value: bson.M{
			"method":  "$methods.method",
			"count":   "$methods.count",
			"percent": bson.M{"$multiply": bson.A{bson.M{"$divide": bson.A{"$methods.count", "$total"}}, 100}},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}
	return aggregateToSlice[dto.LoginMethodBreakdown](ctx, s.db.Collection("activity_logs"), pipeline)
}

// LockoutStats returns lockout rate and count.
func (s *AnalyticsService) LockoutStats(ctx context.Context, maxAttempts int) (float64, int64, error) {
	col := s.db.Collection("users")
	total, _ := col.CountDocuments(ctx, bson.M{"deletedAt": nil})
	locked, _ := col.CountDocuments(ctx, bson.M{"deletedAt": nil, "passwordAttempts": bson.M{"$gte": maxAttempts}})
	rate := 0.0
	if total > 0 {
		rate = math.Round(float64(locked)/float64(total)*10000) / 100
	}
	return rate, locked, nil
}

// PasskeyAdoption returns passkey and biometric adoption stats.
func (s *AnalyticsService) PasskeyAdoption(ctx context.Context) (*dto.PasskeyAdoptionStats, error) {
	col := s.db.Collection("passkeys")
	total, _ := col.CountDocuments(ctx, bson.M{})
	biometric, _ := col.CountDocuments(ctx, bson.M{"attachment": "platform"})
	crossPlatform, _ := col.CountDocuments(ctx, bson.M{"attachment": "cross-platform"})
	backedUp, _ := col.CountDocuments(ctx, bson.M{"backedUp": true})

	activeUsers, _ := s.db.Collection("users").CountDocuments(ctx, bson.M{"deletedAt": nil, "status": "active"})

	// Users with at least one passkey
	distinctUsers, _ := s.db.Collection("passkeys").Distinct(ctx, "userId", bson.M{})
	adoptionRate := 0.0
	if activeUsers > 0 {
		adoptionRate = math.Round(float64(len(distinctUsers))/float64(activeUsers)*10000) / 100
	}
	backedUpRate := 0.0
	if total > 0 {
		backedUpRate = math.Round(float64(backedUp)/float64(total)*10000) / 100
	}

	return &dto.PasskeyAdoptionStats{
		TotalPasskeys:     total,
		BiometricPasskeys: biometric,
		CrossPlatformKeys: crossPlatform,
		AdoptionRate:      adoptionRate,
		BackedUpRate:      backedUpRate,
	}, nil
}

// ── Anomaly detection ─────────────────────────────────────────────────────────

// CredentialStuffingSignals finds IPs with many distinct users locked out in a short window.
func (s *AnalyticsService) CredentialStuffingSignals(ctx context.Context, windowMin int, minAccounts int) ([]dto.CredentialStuffingSignal, error) {
	since := time.Now().Add(-time.Duration(windowMin) * time.Minute)
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"action":    "userReachMaxPasswordAttempt",
			"createdAt": bson.M{"$gte": since},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":     "$ipAddress",
			"userIds": bson.M{"$addToSet": "$actorId"},
			"attempts": bson.M{"$sum": 1},
		}}},
		{{Key: "$project", Value: bson.M{
			"ipAddress":   "$_id",
			"uniqueUsers": bson.M{"$size": "$userIds"},
			"attempts":    1,
		}}},
		{{Key: "$match", Value: bson.M{"uniqueUsers": bson.M{"$gte": minAccounts}}}},
		{{Key: "$sort", Value: bson.M{"uniqueUsers": -1}}},
	}
	return aggregateToSlice[dto.CredentialStuffingSignal](ctx, s.db.Collection("activity_logs"), pipeline)
}

// DeviceProliferationSignals finds users with abnormal device counts (z-score > 3).
func (s *AnalyticsService) DeviceProliferationSignals(ctx context.Context) ([]dto.DeviceProliferationSignal, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{"_id": "$userId", "deviceCount": bson.M{"$sum": 1}}}},
		{{Key: "$group", Value: bson.M{
			"_id":    nil,
			"avg":    bson.M{"$avg": "$deviceCount"},
			"stdDev": bson.M{"$stdDevPop": "$deviceCount"},
			"all":    bson.M{"$push": bson.M{"userId": "$_id", "deviceCount": "$deviceCount"}},
		}}},
		{{Key: "$unwind", Value: "$all"}},
		{{Key: "$project", Value: bson.M{
			"userId":      "$all.userId",
			"deviceCount": "$all.deviceCount",
			"zScore": bson.M{"$divide": bson.A{
				bson.M{"$subtract": bson.A{"$all.deviceCount", "$avg"}},
				bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$stdDev", 0}}, 1, "$stdDev"}},
			}},
		}}},
		{{Key: "$match", Value: bson.M{"zScore": bson.M{"$gt": 3}}}},
		{{Key: "$sort", Value: bson.M{"deviceCount": -1}}},
	}
	return aggregateToSlice[dto.DeviceProliferationSignal](ctx, s.db.Collection("devices"), pipeline)
}

// ── Fraud scoring ─────────────────────────────────────────────────────────────

// FraudSignalSummary computes per-user risk scores from multiple signals.
func (s *AnalyticsService) FraudSignalSummary(ctx context.Context, maxAttempts int) (*dto.FraudSummary, error) {
	// Weight map from the spec
	type userScore struct {
		UserID  string
		Score   int
		Signals []string
	}
	scores := make(map[string]*userScore)

	addSignal := func(userID, signal string, weight int) {
		if _, ok := scores[userID]; !ok {
			scores[userID] = &userScore{UserID: userID}
		}
		scores[userID].Score += weight
		scores[userID].Signals = append(scores[userID].Signals, signal)
	}

	// Signal: users approaching lockout
	cur, _ := s.db.Collection("users").Find(ctx, bson.M{
		"deletedAt":        nil,
		"passwordAttempts": bson.M{"$gte": maxAttempts - 1},
	})
	for cur.Next(ctx) {
		var u struct {
			ID uuid.UUID `bson:"_id"`
		}
		_ = cur.Decode(&u)
		addSignal(u.ID.String(), "near_lockout", 20)
	}
	cur.Close(ctx)

	// Signal: credential stuffing IPs
	stuffingSignals, _ := s.CredentialStuffingSignals(ctx, 5, 10)
	for range stuffingSignals {
		// flag would require session IP join — mark as system-level signal
	}

	// Build result
	var signals []dto.FraudSignal
	now := time.Now()
	for _, us := range scores {
		if us.Score < 15 {
			continue
		}
		signals = append(signals, dto.FraudSignal{
			UserID:     us.UserID,
			RiskScore:  us.Score,
			RiskLevel:  riskLevel(us.Score),
			Signals:    us.Signals,
			DetectedAt: now,
		})
	}

	return &dto.FraudSummary{
		HighRiskUsers: signals,
		TotalFlagged:  int64(len(signals)),
	}, nil
}

func riskLevel(score int) string {
	switch {
	case score >= 91:
		return "suspend"
	case score >= 61:
		return "reauth"
	case score >= 31:
		return "review"
	default:
		return "monitor"
	}
}

// ── Mobile verification stats ─────────────────────────────────────────────────

func (s *AnalyticsService) MobileVerificationStats(ctx context.Context) (*dto.MobileVerificationStats, error) {
	col := s.db.Collection("user_mobiles")
	total, _ := col.CountDocuments(ctx, bson.M{})
	verified, _ := col.CountDocuments(ctx, bson.M{"isVerified": true})
	rate := 0.0
	if total > 0 {
		rate = math.Round(float64(verified)/float64(total)*10000) / 100
	}
	return &dto.MobileVerificationStats{
		TotalMobiles:     total,
		Verified:         verified,
		VerificationRate: rate,
	}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func granularityGroup(granularity, field string) bson.M {
	g := bson.M{
		"year":  bson.M{"$year": "$" + field},
		"month": bson.M{"$month": "$" + field},
	}
	if granularity != "month" {
		g["day"] = bson.M{"$dayOfMonth": "$" + field}
	}
	if granularity == "hour" {
		g["hour"] = bson.M{"$hour": "$" + field}
	}
	return g
}

func formatPeriodID(id bson.M, granularity string) string {
	y, _ := id["year"].(int32)
	m, _ := id["month"].(int32)
	switch granularity {
	case "month":
		return fmt.Sprintf("%04d-%02d", y, m)
	default:
		d, _ := id["day"].(int32)
		return fmt.Sprintf("%04d-%02d-%02d", y, m, d)
	}
}

func timeSeriesFromPipeline(ctx context.Context, col *mongo.Collection, pipeline mongo.Pipeline, granularity string) ([]dto.TimePoint, error) {
	cur, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var series []dto.TimePoint
	for cur.Next(ctx) {
		var r struct {
			ID    bson.M `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cur.Decode(&r); err != nil {
			continue
		}
		series = append(series, dto.TimePoint{Date: formatPeriodID(r.ID, granularity), Count: r.Count})
	}
	return series, nil
}

func aggregateToSlice[T any](ctx context.Context, col *mongo.Collection, pipeline mongo.Pipeline) ([]T, error) {
	cur, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var result []T
	return result, cur.All(ctx, &result)
}
