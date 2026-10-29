package service

// Audit trail — the append-only record of WHO changed WHAT, with the document
// before and after. Separate collection from activity_logs on purpose:
// activity_logs is the HTTP transcript (one row per request, no entity state),
// audit_logs is the change history read to settle a dispute or spot abuse. They
// join on correlationId. See shared/audit for the Entry shape.

import (
	"context"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/audit"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
)

const auditCollection = "audit_logs"

// AuditService writes and reads the audit trail.
type AuditService struct {
	col *mongo.Collection
	log *zap.Logger
}

func NewAudit(db *database.MongoDB, log *zap.Logger) *AuditService {
	return &AuditService{col: db.Collection(auditCollection), log: log}
}

// EnsureIndexes covers the ways the trail is read: a tenant's timeline, one
// record's history, one actor's activity, and the high-risk filter. **No TTL
// index** — an investigation looks back months, so retention is a documented
// operational decision, not a silent default.
func (s *AuditService) EnsureIndexes(ctx context.Context) error {
	_, err := s.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "at", Value: -1}}, Options: options.Index().SetName("idx_tenant_at")},
		{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "entityType", Value: 1}, {Key: "entityId", Value: 1}, {Key: "at", Value: -1}}, Options: options.Index().SetName("idx_tenant_entity_at")},
		{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "actorId", Value: 1}, {Key: "at", Value: -1}}, Options: options.Index().SetName("idx_tenant_actor_at")},
		{Keys: bson.D{{Key: "tenantId", Value: 1}, {Key: "risk", Value: 1}, {Key: "at", Value: -1}}, Options: options.Index().SetName("idx_tenant_risk_at")},
	})
	return err
}

// Record appends one entry. Best-effort by contract (audit.Recorder): a failure
// is logged and swallowed, because losing the user's actual work to an audit
// problem is the worse outcome.
func (s *AuditService) Record(ctx context.Context, e audit.Entry) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	// Actor / IP / platform / correlation come from the request context, so no
	// call site has to remember them and none of them can come from a body.
	e = e.Bind(audit.FromContext(ctx))
	if _, err := s.col.InsertOne(ctx, e); err != nil && s.log != nil {
		s.log.Error("audit: write failed — change is NOT in the trail",
			zap.String("action", e.Action), zap.String("entityId", e.EntityID), zap.Error(err))
	}
}

// AuditFilter narrows a trail query. Every field is optional; TenantID is applied
// by the caller (never trusted from a client).
type AuditFilter struct {
	ActorID    string
	EntityType string
	EntityID   string
	Action     string
	Risk       string
}

func auditFilter(tenantID uuid.UUID, f AuditFilter) bson.M {
	filter := bson.M{"tenantId": tenantID}
	if f.ActorID != "" {
		filter["actorId"] = f.ActorID
	}
	if f.EntityType != "" {
		filter["entityType"] = f.EntityType
	}
	if f.EntityID != "" {
		filter["entityId"] = f.EntityID
	}
	if f.Action != "" {
		filter["action"] = f.Action
	}
	if f.Risk != "" {
		filter["risk"] = f.Risk
	}
	return filter
}

// GroupFields is the audit trail's grouping whitelist. Every entry is one of a
// small set of actions, entity types, risk levels and actor roles — exactly what
// makes a trail worth grouping. Free-text fields (actorId, entityId, note) are
// deliberately absent: grouping by them yields one group per row.
var GroupFields = pagination.GroupFields{
	"action":     {Path: "action", Kind: pagination.GroupString},
	"entityType": {Path: "entityType", Kind: pagination.GroupString},
	"risk":       {Path: "risk", Kind: pagination.GroupString},
	"actorRole":  {Path: "actorRole", Kind: pagination.GroupString},
}

// ListGrouped returns one grouped page of the trail — the same records as List,
// wrapped with each entry's group path, newest-first within each group.
func (s *AuditService) ListGrouped(ctx context.Context, tenantID uuid.UUID, f AuditFilter, q pagination.Query) ([]pagination.GroupedRow, int64, []pagination.GroupCount, error) {
	levels, err := GroupFields.Levels(q)
	if err != nil {
		return nil, 0, nil, err
	}
	return pagination.GroupedFind[audit.Entry](ctx, s.col, auditFilter(tenantID, f), levels, q, bson.E{Key: "at", Value: -1})
}

// ListGroupPaged returns one page of the VISIBLE sequence (headers + rows of
// open groups). The default sort stays `at` descending, carried through as a
// full bson.E so grouping does not silently reverse the newest-first trail.
func (s *AuditService) ListGroupPaged(ctx context.Context, tenantID uuid.UUID, f AuditFilter, q pagination.Query) ([]any, int64, int64, error) {
	levels, err := GroupFields.Levels(q)
	if err != nil {
		return nil, 0, 0, err
	}
	return pagination.GroupPagedFind[audit.Entry](ctx, s.col, auditFilter(tenantID, f), levels, pagination.GroupSummary{}, q, bson.E{Key: "at", Value: -1})
}

// List returns one page of the tenant's audit trail, newest first.
func (s *AuditService) List(ctx context.Context, tenantID uuid.UUID, f AuditFilter, q pagination.Query) ([]audit.Entry, int64, error) {
	filter := auditFilter(tenantID, f)
	total, err := s.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().SetSort(bson.D{{Key: "at", Value: -1}})
	if q.PerPage > 0 {
		opts = opts.SetSkip(q.Skip()).SetLimit(q.Limit())
	}
	cur, err := s.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	out := make([]audit.Entry, 0)
	return out, total, cur.All(ctx, &out)
}

// EntityHistory returns the full trail for one record, oldest first — read as a
// timeline on a record's detail view.
func (s *AuditService) EntityHistory(ctx context.Context, tenantID uuid.UUID, entityType, entityID string) ([]audit.Entry, error) {
	cur, err := s.col.Find(ctx,
		bson.M{"tenantId": tenantID, "entityType": entityType, "entityId": entityID},
		options.Find().SetSort(bson.D{{Key: "at", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]audit.Entry, 0)
	return out, cur.All(ctx, &out)
}

// HighRiskSince powers a digest of watch-worthy changes.
func (s *AuditService) HighRiskSince(ctx context.Context, tenantID uuid.UUID, since string) ([]audit.Entry, error) {
	filter := bson.M{"tenantId": tenantID, "risk": audit.RiskHigh}
	if since != "" {
		filter["at"] = bson.M{"$gte": since}
	}
	cur, err := s.col.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	out := make([]audit.Entry, 0)
	return out, cur.All(ctx, &out)
}
