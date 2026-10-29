package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/storage"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/audit"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
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
	// VersionCode is a monotonic integer (Android's own, derived by CI). The
	// server never computes it — it only refuses to move it backwards.
	VersionCode int                `bson:"versionCode"         json:"versionCode"`
	MinVersion  string             `bson:"minVersion"          json:"minVersion"`  // minimum allowed, e.g. "2.0.0"
	ForceUpdate bool               `bson:"forceUpdate"         json:"forceUpdate"` // if true, block below minVersion
	// APKKey is the storage key of the current artefact, never a URL — it is
	// resolved to a fetchable (presigned) URL at read time.
	APKKey string `bson:"apkKey,omitempty" json:"apkKey,omitempty"`
	// RetiredAPKKeys holds older artefacts still in storage, newest first;
	// bounded by apkRetention, anything evicted from it is deleted.
	RetiredAPKKeys []string           `bson:"retiredApkKeys,omitempty" json:"-"`
	APKSize        int64              `bson:"apkSize,omitempty"        json:"apkSize,omitempty"`
	SHA256         string             `bson:"sha256,omitempty"         json:"sha256,omitempty"`
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
	// SHA256 lets a client reject a truncated or tampered download before
	// handing it to the OS installer.
	SHA256      string `json:"sha256,omitempty"`
	ForceUpdate bool   `json:"forceUpdate"`
}

var (
	ErrVersionNotFound = errors.NewI18n(404, "VERSION_NOT_FOUND", "appVersion.notFound", "No version policy found for this platform")
	// ErrNotMonotonic keeps a release installable: Android refuses an APK whose
	// versionCode is not greater than the installed one, at install time on the
	// device, so without this the failure surfaces days later as "the update does
	// nothing" rather than here as a failed publish.
	ErrNotMonotonic = errors.NewI18n(409, "APP_VERSION_NOT_MONOTONIC", "appVersion.notMonotonic", "This build's version is not newer than the one already published")
	ErrInvalidVersion = errors.NewI18n(400, "APP_VERSION_INVALID", "appVersion.invalidVersion", "Version must be in MAJOR.MINOR.PATCH form")
	ErrStorageUnavailable = errors.NewI18n(503, "STORAGE_UNAVAILABLE", "appVersion.storageUnavailable", "No storage driver is configured for build artefacts")
)

// apkRetention is how many build artefacts are kept per platform — the current
// one plus (apkRetention-1) older ones, so a bad release can be rolled back to
// the immediately previous build.
const apkRetention = 2

// ReleaseInput is one published build; everything in it comes from CI.
type ReleaseInput struct {
	Platform     Platform
	Version      string
	VersionCode  int
	Mandatory    bool
	ReleaseNotes string
	Filename     string
	ContentType  string
	Artifact     io.Reader
}

type AppVersionService struct {
	col     *mongo.Collection
	store   storage.Storage
	audit   audit.Recorder
	log     *zap.Logger
	nowFunc func() time.Time
}

func New(db *database.MongoDB) *AppVersionService {
	return &AppVersionService{col: db.Collection("app_versions"), nowFunc: time.Now}
}

// WithStorage supplies the driver that holds build artefacts. Without it the
// release endpoint cannot store an APK and Check simply resolves no download URL
// — the correct behaviour for a deployment that publishes no artefact.
func (s *AppVersionService) WithStorage(st storage.Storage) *AppVersionService {
	s.store = st
	return s
}

// WithAudit records each publish (an app-bricking lever) on the change trail.
func (s *AppVersionService) WithAudit(a audit.Recorder) *AppVersionService {
	s.audit = a
	return s
}

// WithLogger wires the logger used for best-effort storage warnings.
func (s *AppVersionService) WithLogger(l *zap.Logger) *AppVersionService {
	s.log = l
	return s
}

func (s *AppVersionService) now() time.Time {
	if s.nowFunc != nil {
		return s.nowFunc()
	}
	return time.Now()
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

// ── Release pipeline ────────────────────────────────────────────────────────

// DownloadURL answers where the platform's current build can be fetched, or ""
// when nothing is published. It backs a stable link (safe to put in an email):
// the email cannot hold a presigned URL, which expires, so it holds a route that
// resolves one fresh when somebody follows it.
func (s *AppVersionService) DownloadURL(ctx context.Context, platform Platform, requestBase string) (string, error) {
	var av AppVersion
	if err := s.col.FindOne(ctx, bson.M{"platform": platform}).Decode(&av); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		return "", err
	}
	return s.downloadURL(ctx, &av, requestBase), nil
}

// downloadURL resolves the stored key to something the device can fetch. A
// failure is logged and swallowed: a client that cannot be told WHERE to get the
// update is still better off knowing it needs one (it keeps the stored StoreURL).
func (s *AppVersionService) downloadURL(ctx context.Context, av *AppVersion, requestBase string) string {
	if av.APKKey == "" || s.store == nil {
		return av.StoreURL
	}
	url, err := s.store.PresignDownload(ctx, av.APKKey, requestBase)
	if err != nil {
		if s.log != nil {
			s.log.Warn("appversion: could not resolve download url", zap.String("key", av.APKKey), zap.Error(err))
		}
		return av.StoreURL
	}
	return url
}

// Release publishes one build: it streams the artefact into storage FIRST, then
// upserts the platform's policy to point at it SECOND — so a failure between the
// two leaves an orphaned file (harmless, evicted by the next release) rather than
// a policy advertising a download that does not exist.
func (s *AppVersionService) Release(ctx context.Context, in ReleaseInput) (*AppVersion, error) {
	if _, err := parseSemVer(in.Version); err != nil {
		return nil, ErrInvalidVersion
	}
	if in.VersionCode <= 0 {
		return nil, ErrNotMonotonic
	}

	var prev AppVersion
	hasPrev := true
	if err := s.col.FindOne(ctx, bson.M{"platform": in.Platform}).Decode(&prev); err != nil {
		if err != mongo.ErrNoDocuments {
			return nil, err
		}
		hasPrev = false
	}
	if hasPrev && in.VersionCode <= prev.VersionCode {
		return nil, ErrNotMonotonic
	}

	now := s.now()
	set := bson.M{
		"currentVersion": in.Version,
		"versionCode":    in.VersionCode,
		"forceUpdate":    in.Mandatory,
		"releaseNotes":   in.ReleaseNotes,
		"updatedAt":      now,
	}
	// minVersion is the only thing that enforces: a mandatory release raises the
	// floor to itself; a non-mandatory one carries the existing floor forward so
	// it nudges without locking anyone out.
	switch {
	case in.Mandatory:
		set["minVersion"] = in.Version
	case hasPrev:
		set["minVersion"] = prev.MinVersion
	default:
		set["minVersion"] = in.Version
	}

	if in.Artifact != nil {
		if s.store == nil {
			return nil, ErrStorageUnavailable
		}
		key, size, digest, err := s.storeArtifact(ctx, in)
		if err != nil {
			return nil, err
		}
		set["apkKey"] = key
		set["apkSize"] = size
		set["sha256"] = digest
		set["retiredApkKeys"] = retire(prev.APKKey, prev.RetiredAPKKeys)
	}

	update := bson.M{
		"$set":         set,
		"$setOnInsert": bson.M{"_id": uuid.New(), "platform": in.Platform, "createdAt": now},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	var av AppVersion
	if err := s.col.FindOneAndUpdate(ctx, bson.M{"platform": in.Platform}, update, opts).Decode(&av); err != nil {
		return nil, err
	}

	// Eviction runs AFTER the policy is committed, so a file is only deleted once
	// nothing points at it. Best-effort.
	s.evict(ctx, prev, av)

	if s.audit != nil {
		s.audit.Record(ctx, audit.New("appVersion.release", "app_versions", string(in.Platform),
			audit.Snapshot(policySnapshot(prev, hasPrev)), audit.Snapshot(policySnapshot(av, true))))
	}
	return &av, nil
}

const apkContentType = "application/vnd.android.package-archive"

// storeArtifact streams the upload into storage while hashing it, so the bytes
// are read exactly once.
func (s *AppVersionService) storeArtifact(ctx context.Context, in ReleaseInput) (key string, size int64, digest string, err error) {
	h := sha256.New()
	counter := &countingReader{r: io.TeeReader(in.Artifact, h)}

	filename := path.Base(strings.TrimSpace(in.Filename))
	if filename == "" || filename == "." || filename == "/" {
		filename = fmt.Sprintf("%s-%s.apk", in.Platform, in.Version)
	}
	contentType := artifactContentType(filename, in.ContentType)
	folder := path.Join("app-builds", string(in.Platform), in.Version)
	key, err = s.store.Upload(ctx, folder, filename, counter, contentType)
	if err != nil {
		return "", 0, "", err
	}
	return key, counter.n, hex.EncodeToString(h.Sum(nil)), nil
}

// artifactContentType is the type the artefact is STORED with. An .apk is always
// an Android package whatever the multipart part claimed, so a phone's browser
// handed a zip type doesn't save it as app-release.apk.zip.
func artifactContentType(filename, declared string) string {
	if strings.EqualFold(path.Ext(filename), ".apk") || declared == "" {
		return apkContentType
	}
	return declared
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// retire pushes the outgoing current key onto the retired list, newest first,
// trimmed so the TOTAL kept (current + retired) is apkRetention.
func retire(outgoing string, existing []string) []string {
	if outgoing == "" {
		return existing
	}
	next := append([]string{outgoing}, existing...)
	if len(next) > apkRetention-1 {
		next = next[:apkRetention-1]
	}
	return next
}

// evict deletes artefacts the new policy no longer references (best-effort).
func (s *AppVersionService) evict(ctx context.Context, prev, next AppVersion) {
	if s.store == nil {
		return
	}
	kept := map[string]struct{}{}
	if next.APKKey != "" {
		kept[next.APKKey] = struct{}{}
	}
	for _, k := range next.RetiredAPKKeys {
		kept[k] = struct{}{}
	}
	candidates := append([]string{prev.APKKey}, prev.RetiredAPKKeys...)
	for _, k := range candidates {
		if k == "" {
			continue
		}
		if _, still := kept[k]; still {
			continue
		}
		if err := s.store.Delete(ctx, k); err != nil && s.log != nil {
			s.log.Warn("appversion: could not evict old build", zap.String("key", k), zap.Error(err))
		}
	}
}

// policySnapshot is what the audit trail records — the artefact KEY, never the
// resolved URL (which expires).
func policySnapshot(av AppVersion, exists bool) map[string]any {
	if !exists {
		return map[string]any{}
	}
	return map[string]any{
		"currentVersion": av.CurrentVersion,
		"versionCode":    av.VersionCode,
		"minVersion":     av.MinVersion,
		"forceUpdate":    av.ForceUpdate,
		"apkKey":         av.APKKey,
		"sha256":         av.SHA256,
	}
}
