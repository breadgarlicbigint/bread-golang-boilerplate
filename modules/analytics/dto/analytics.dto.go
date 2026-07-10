package dto

import "time"

// DateRangeQuery binds ?startDate=&endDate= query params.
type DateRangeQuery struct {
	StartDate string `form:"startDate" binding:"required"` // RFC3339
	EndDate   string `form:"endDate"   binding:"required"`
	Granularity string `form:"granularity"` // day | week | month  (default: day)
}

func (q *DateRangeQuery) Dates() (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, q.StartDate)
	if err != nil {
		start, err = time.Parse("2006-01-02", q.StartDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	end, err := time.Parse(time.RFC3339, q.EndDate)
	if err != nil {
		end, err = time.Parse("2006-01-02", q.EndDate)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return start, end, nil
}

// ── Time series ───────────────────────────────────────────────────────────────

type TimePoint struct {
	Date  string  `json:"date"`
	Count int64   `json:"count"`
	Value float64 `json:"value,omitempty"`
}

// ── User analytics ────────────────────────────────────────────────────────────

type UserRegistrationStats struct {
	Series    []TimePoint `json:"series"`
	Total     int64       `json:"total"`
	GrowthPct float64     `json:"growthPct"`
}

type UserChurnStats struct {
	ChurnRate  float64     `json:"churnRate"`
	Churned    int64       `json:"churned"`
	Total      int64       `json:"total"`
	Series     []TimePoint `json:"series"`
}

type SignupMethodBreakdown struct {
	Method  string  `json:"method"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type UserStatusDistribution struct {
	Status  string `json:"status"`
	Count   int64  `json:"count"`
}

type UserSummary struct {
	Registrations  UserRegistrationStats   `json:"registrations"`
	Churn          UserChurnStats          `json:"churn"`
	SignupMethods  []SignupMethodBreakdown  `json:"signupMethods"`
	StatusDist     []UserStatusDistribution `json:"statusDistribution"`
	BlockedTrend   []TimePoint             `json:"blockedTrend"`
	VerificationRate float64               `json:"emailVerificationRate"`
}

// ── Auth / Session analytics ──────────────────────────────────────────────────

type LoginMethodBreakdown struct {
	Method  string  `json:"method"`
	Count   int64   `json:"count"`
	Percent float64 `json:"percent"`
}

type SessionStats struct {
	ActiveSessions    int64   `json:"activeSessions"`
	AvgSessionsPerUser float64 `json:"avgSessionsPerUser"`
	MaxSessions       int32   `json:"maxSessions"`
}

type AuthSummary struct {
	LoginFrequency []TimePoint            `json:"loginFrequency"`
	LoginMethods   []LoginMethodBreakdown `json:"loginMethods"`
	LockoutRate    float64                `json:"lockoutRate"`
	LockedUsers    int64                  `json:"lockedUsers"`
	SessionStats   SessionStats           `json:"sessionStats"`
}

// ── Anomaly detection ─────────────────────────────────────────────────────────

type ImpossibleTravelEvent struct {
	UserID    string  `json:"userId"`
	Country1  string  `json:"country1"`
	Country2  string  `json:"country2"`
	DistanceKM float64 `json:"distanceKm"`
	TimeDeltaMin float64 `json:"timeDeltaMin"`
	ImpliedSpeedKMH float64 `json:"impliedSpeedKmh"`
	DetectedAt time.Time `json:"detectedAt"`
}

type CredentialStuffingSignal struct {
	IPAddress   string `json:"ipAddress"`
	UniqueUsers int64  `json:"uniqueUsers"`
	Attempts    int64  `json:"attempts"`
}

type DeviceProliferationSignal struct {
	UserID      string  `json:"userId"`
	DeviceCount int64   `json:"deviceCount"`
	ZScore      float64 `json:"zScore"`
}

type AnomalySummary struct {
	ImpossibleTravel    []ImpossibleTravelEvent     `json:"impossibleTravel"`
	CredentialStuffing  []CredentialStuffingSignal  `json:"credentialStuffing"`
	DeviceProliferation []DeviceProliferationSignal `json:"deviceProliferation"`
	LockoutSpike        []TimePoint                 `json:"lockoutSpike"`
}

// ── Fraud detection ───────────────────────────────────────────────────────────

type FraudSignal struct {
	UserID      string  `json:"userId"`
	RiskScore   int     `json:"riskScore"`
	RiskLevel   string  `json:"riskLevel"` // monitor | review | reauth | suspend
	Signals     []string `json:"signals"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type FraudSummary struct {
	HighRiskUsers     []FraudSignal `json:"highRiskUsers"`
	TotalFlagged      int64         `json:"totalFlagged"`
	SuspendedToday    int64         `json:"suspendedToday"`
}

// ── Passkey / biometrics ──────────────────────────────────────────────────────

type PasskeyAdoptionStats struct {
	TotalPasskeys     int64   `json:"totalPasskeys"`
	BiometricPasskeys int64   `json:"biometricPasskeys"`
	CrossPlatformKeys int64   `json:"crossPlatformKeys"`
	AdoptionRate      float64 `json:"adoptionRate"` // % of active users with at least 1 passkey
	BackedUpRate      float64 `json:"backedUpRate"` // % synced passkeys
}

// ── Mobile verification ───────────────────────────────────────────────────────

type MobileVerificationStats struct {
	TotalMobiles     int64   `json:"totalMobiles"`
	Verified         int64   `json:"verified"`
	VerificationRate float64 `json:"verificationRate"`
	ByCountry        []struct {
		Country string  `json:"country"`
		Total   int64   `json:"total"`
		Verified int64  `json:"verified"`
		Rate    float64 `json:"rate"`
	} `json:"byCountry"`
}
