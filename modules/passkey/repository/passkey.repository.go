package repository

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/entity"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const col = "passkeys"

type PasskeyRepository struct {
	col *mongo.Collection
}

func New(db *database.MongoDB) *PasskeyRepository {
	return &PasskeyRepository{col: db.Collection(col)}
}

func (r *PasskeyRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetName("idx_user_id")},
		{Keys: bson.D{{Key: "credentialIdBase64", Value: 1}}, Options: options.Index().SetUnique(true).SetName("idx_cred_id_unique")},
		{Keys: bson.D{{Key: "tenantId", Value: 1}}, Options: options.Index().SetSparse(true).SetName("idx_tenant_id")},
	}
	_, err := r.col.Indexes().CreateMany(ctx, models)
	return err
}

func (r *PasskeyRepository) Create(ctx context.Context, p *entity.Passkey) error {
	p.ID = uuid.New()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, p)
	return err
}

func (r *PasskeyRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Passkey, error) {
	cur, err := r.col.Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var passkeys []*entity.Passkey
	return passkeys, cur.All(ctx, &passkeys)
}

func (r *PasskeyRepository) FindByCredentialID(ctx context.Context, credIDBase64 string) (*entity.Passkey, error) {
	var p entity.Passkey
	err := r.col.FindOne(ctx, bson.M{"credentialIdBase64": credIDBase64}).Decode(&p)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &p, err
}

func (r *PasskeyRepository) UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error {
	now := time.Now()
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$set": bson.M{"signCount": signCount, "lastUsedAt": now, "updatedAt": now}})
	return err
}

func (r *PasskeyRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "userId": userID})
	return err
}
