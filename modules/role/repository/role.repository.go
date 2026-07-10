package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/role/entity"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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
