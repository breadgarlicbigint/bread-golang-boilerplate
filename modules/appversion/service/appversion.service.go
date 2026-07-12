package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Platform identifies the client platform.
type Platform string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWeb     Platform = "web"
)

// UpdateStatus is the server's verdict on a client version.
type UpdateStatus string

const (
	UpdateRequired  UpdateStatus = "required"  // force update — block access
	UpdateAvailable UpdateStatus = "available" // soft nudge — allow access
	UpToDate        UpdateStatus = "up_to_date" // nothing to do
)

// AppVersion stores the version policy for a platform.
type AppVersion struct {
	ID             uuid.UUID `bson:"_id,omitempty"       json:"id"`
	Platform       Platform           `bson:"platform"            json:"platform"`
	CurrentVersion string             `bson:"currentVersion"      json:"currentVersion"` // latest published, e.g. "2.5.0"
	MinVersion     string             `bson:"minVersion"          json:"minVersion"`     // minimum allowed, e.g. "2.0.0"
	ForceUpdate    bool               `bson:"forceUpdate"         json:"forceUpdate"`    // if true, block below minVersion
	ReleaseNotes   string             `bson:"releaseNotes"        json:"releaseNotes,omitempty"`
	StoreURL       string             `bson:"storeUrl"            json:"storeUrl,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt"           json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt"           json:"updatedAt"`
}

// VersionCheckResponse is returned to the client on every request header check.
type VersionCheckResponse struct {
	Status         UpdateStatus `json:"status"`
	CurrentVersion string       `json:"currentVersion"`
	MinVersion     string       `json:"minVersion"`
	ClientVersion  string       `json:"clientVersion"`
	ReleaseNotes   string       `json:"releaseNotes,omitempty"`
	StoreURL       string       `json:"storeUrl,omitempty"`
	ForceUpdate    bool         `json:"forceUpdate"`
}

var ErrVersionNotFound = errors.NewI18n(404, "VERSION_NOT_FOUND", "appVersion.notFound", "No version policy found for this platform")

type AppVersionService struct {
	col *mongo.Collection
}

func New(db *database.MongoDB) *AppVersionService {
	return &AppVersionService{col: db.Collection("app_versions")}
}

func (s *AppVersionService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "platform", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("idx_platform"),
	})
	return err
}

// Check compares the client version against the stored policy.
func (s *AppVersionService) Check(ctx context.Context, platform Platform, clientVersion string) (*VersionCheckResponse, error) {
	var av AppVersion
	if err := s.col.FindOne(ctx, bson.M{"platform": platform}).Decode(&av); err != nil {
		if err == mongo.ErrNoDocuments {
			return &VersionCheckResponse{Status: UpToDate, ClientVersion: clientVersion}, nil
		}
		return nil, err
	}

	cmp, err := compareVersions(clientVersion, av.MinVersion)
	if err != nil {
		return nil, fmt.Errorf("version parse: %w", err)
	}

	resp := &VersionCheckResponse{
		CurrentVersion: av.CurrentVersion,
		MinVersion:     av.MinVersion,
		ClientVersion:  clientVersion,
		ReleaseNotes:   av.ReleaseNotes,
		StoreURL:       av.StoreURL,
		ForceUpdate:    av.ForceUpdate,
	}

	switch {
	case cmp < 0 && av.ForceUpdate:
		resp.Status = UpdateRequired
	case cmp < 0:
		resp.Status = UpdateAvailable
	default:
		latestCmp, _ := compareVersions(clientVersion, av.CurrentVersion)
		if latestCmp < 0 {
			resp.Status = UpdateAvailable
		} else {
			resp.Status = UpToDate
		}
	}

	return resp, nil
}

// Upsert creates or updates the version policy for a platform (admin endpoint).
func (s *AppVersionService) Upsert(ctx context.Context, platform Platform, current, min string, force bool, notes, storeURL string) (*AppVersion, error) {
	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"currentVersion": current,
			"minVersion":     min,
			"forceUpdate":    force,
			"releaseNotes":   notes,
			"storeUrl":       storeURL,
			"updatedAt":      now,
		},
		"$setOnInsert": bson.M{"_id": uuid.New(), "platform": platform, "createdAt": now},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var av AppVersion
	if err := s.col.FindOneAndUpdate(ctx, bson.M{"platform": platform}, update, opts).Decode(&av); err != nil {
		return nil, err
	}
	return &av, nil
}

// List returns version policies for all platforms.
func (s *AppVersionService) List(ctx context.Context) ([]AppVersion, error) {
	cur, err := s.col.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var versions []AppVersion
	return versions, cur.All(ctx, &versions)
}

// ── Semantic version comparison ───────────────────────────────────────────────

// compareVersions returns -1 (a < b), 0 (a == b), 1 (a > b).
func compareVersions(a, b string) (int, error) {
	aParts, err := parseSemVer(a)
	if err != nil {
		return 0, err
	}
	bParts, err := parseSemVer(b)
	if err != nil {
		return 0, err
	}
	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1, nil
		}
		if aParts[i] > bParts[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseSemVer(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		// Strip pre-release suffix
		p = strings.Split(p, "-")[0]
		n, err := strconv.Atoi(p)
		if err != nil {
			return result, fmt.Errorf("invalid version part %q in %q", p, v)
		}
		result[i] = n
	}
	return result, nil
}
