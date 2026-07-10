package service

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/tenant/entity"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrTenantNotFound   = errors.New(404, "TENANT_NOT_FOUND", "Tenant not found")
	ErrTenantSuspended  = errors.New(403, "TENANT_SUSPENDED", "Tenant account is suspended")
	ErrSlugTaken        = errors.New(409, "SLUG_TAKEN", "Tenant slug is already taken")
)

type TenantService struct {
	col    *mongo.Collection
	members *mongo.Collection
}

func New(db *database.MongoDB) *TenantService {
	return &TenantService{
		col:     db.Collection("tenants"),
		members: db.Collection("tenant_members"),
	}
}

func (s *TenantService) EnsureIndexes(ctx context.Context) error {
	tenantIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_slug")},
		{Keys: bson.D{{Key: "domain", Value: 1}}, Options: options.Index().SetSparse(true).SetName("idx_domain")},
		{Keys: bson.D{{Key: "ownerId", Value: 1}}, Options: options.Index().SetName("idx_owner")},
		{Keys: bson.D{{Key: "status", Value: 1}}, Options: options.Index().SetName("idx_status")},
	}
	memberIdx := []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_tenant_user")},
	}
	_, err := s.col.Indexes().CreateMany(ctx, tenantIdx)
	if err != nil {
		return err
	}
	_, err = s.members.Indexes().CreateMany(ctx, memberIdx)
	return err
}

// Create provisions a new tenant.
func (s *TenantService) Create(ctx context.Context, slug, name string, ownerID uuid.UUID, plan entity.TenantPlan) (*entity.Tenant, error) {
	exists, _ := s.col.CountDocuments(ctx, bson.M{"slug": slug})
	if exists > 0 {
		return nil, ErrSlugTaken
	}
	now := time.Now()
	t := &entity.Tenant{
		ID:        uuid.New(),
		Slug:      slug,
		Name:      name,
		Status:    entity.TenantStatusActive,
		Plan:      plan,
		OwnerID:   ownerID,
		MaxUsers:  planMaxUsers(plan),
		Settings:  defaultSettings(),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if plan == entity.TenantPlanFree {
		trialEnd := now.Add(14 * 24 * time.Hour)
		t.Status = entity.TenantStatusTrial
		t.TrialEndsAt = &trialEnd
	}
	if _, err := s.col.InsertOne(ctx, t); err != nil {
		return nil, err
	}
	// Add owner as admin member
	_ = s.AddMember(ctx, t.ID, ownerID, "admin")
	return t, nil
}

// FindBySlug looks up a tenant by slug (used in subdomain resolution).
func (s *TenantService) FindBySlug(ctx context.Context, slug string) (*entity.Tenant, error) {
	var t entity.Tenant
	err := s.col.FindOne(ctx, bson.M{"slug": slug, "deletedAt": nil}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, ErrTenantNotFound
	}
	return &t, err
}

// FindByDomain looks up a tenant by custom domain.
func (s *TenantService) FindByDomain(ctx context.Context, domain string) (*entity.Tenant, error) {
	var t entity.Tenant
	err := s.col.FindOne(ctx, bson.M{"domain": domain, "deletedAt": nil}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, ErrTenantNotFound
	}
	return &t, err
}

// FindByID fetches a tenant by ObjectID.
func (s *TenantService) FindByID(ctx context.Context, id uuid.UUID) (*entity.Tenant, error) {
	var t entity.Tenant
	err := s.col.FindOne(ctx, bson.M{"_id": id, "deletedAt": nil}).Decode(&t)
	if err == mongo.ErrNoDocuments {
		return nil, ErrTenantNotFound
	}
	return &t, err
}

// AddMember adds a user to a tenant with the given role.
func (s *TenantService) AddMember(ctx context.Context, tenantID, userID uuid.UUID, role string) error {
	m := &entity.TenantMember{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		JoinedAt: time.Now(),
	}
	_, err := s.members.InsertOne(ctx, m)
	if mongo.IsDuplicateKeyError(err) {
		return nil // already a member
	}
	return err
}

// IsMember returns true if the user belongs to the tenant.
func (s *TenantService) IsMember(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	n, err := s.members.CountDocuments(ctx, bson.M{"tenantId": tenantID, "userId": userID})
	return n > 0, err
}

// MemberRole returns the user's role within the tenant.
func (s *TenantService) MemberRole(ctx context.Context, tenantID, userID uuid.UUID) (string, error) {
	var m entity.TenantMember
	err := s.members.FindOne(ctx, bson.M{"tenantId": tenantID, "userId": userID}).Decode(&m)
	if err != nil {
		return "", err
	}
	return m.Role, nil
}

func planMaxUsers(p entity.TenantPlan) int {
	switch p {
	case entity.TenantPlanFree, entity.TenantPlanStarter:
		return 5
	case entity.TenantPlanPro:
		return 50
	default:
		return 1000
	}
}

func defaultSettings() entity.TenantSettings {
	return entity.TenantSettings{
		AllowPublicSignup:  true,
		RequireEmailVerify: true,
		Require2FA:         false,
		MaxSessionsPerUser: 5,
	}
}
