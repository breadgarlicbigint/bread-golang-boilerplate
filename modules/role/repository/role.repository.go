package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collection = "roles"

type RoleRepository struct {
	col *mongo.Collection
}

func New(db *database.MongoDB) *RoleRepository {
	return &RoleRepository{col: db.Collection(collection)}
}

// FindByID returns a role by its UUID primary key.
func (r *RoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.Role, error) {
	var role entity.Role
	if err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("role %s not found", id)
		}
		return nil, err
	}
	return &role, nil
}

// FindBySlug returns a role by its slug string (e.g. "admin", "user").
func (r *RoleRepository) FindBySlug(ctx context.Context, slug string) (*entity.Role, error) {
	var role entity.Role
	if err := r.col.FindOne(ctx, bson.M{"slug": slug}).Decode(&role); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("role %q not found", slug)
		}
		return nil, err
	}
	return &role, nil
}

// FindAll returns every role, sorted by name — used to populate role-selection
// dropdowns (e.g. the admin "create user" form).
func (r *RoleRepository) FindAll(ctx context.Context) ([]entity.Role, error) {
	cur, err := r.col.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	roles := make([]entity.Role, 0)
	if err := cur.All(ctx, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}
