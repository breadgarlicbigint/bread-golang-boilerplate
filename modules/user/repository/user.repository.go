package repository

import (
	"context"
	"time"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/pagination"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"go.mongodb.org/mongo-driver/bson"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collectionName = "users"

type UserRepository struct {
	col *mongo.Collection
}

func New(db *database.MongoDB) *UserRepository {
	return &UserRepository{col: db.Collection(collectionName)}
}

// EnsureIndexes creates unique indexes on email and username.
func (r *UserRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true).SetSparse(true)},
		{Keys: bson.D{{Key: "googleId", Value: 1}}, Options: options.Index().SetSparse(true)},
		{Keys: bson.D{{Key: "appleId", Value: 1}}, Options: options.Index().SetSparse(true)},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "roleId", Value: 1}}},
		{Keys: bson.D{{Key: "deletedAt", Value: 1}}},
	}
	_, err := r.col.Indexes().CreateMany(ctx, models)
	return err
}

// Create inserts a new user document.
func (r *UserRepository) Create(ctx context.Context, u *entity.User) error {
	u.ID = uuid.New()
	now := time.Now()
	u.CreatedAt = now
	u.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, u)
	return err
}

// FindByID fetches a user by ObjectID (excludes soft-deleted).
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	filter := bson.M{"_id": id, "deletedAt": nil}
	return r.findOne(ctx, filter)
}

// FindByEmail fetches a user by email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	return r.findOne(ctx, bson.M{"email": email, "deletedAt": nil})
}

// FindByUsername fetches a user by username.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	return r.findOne(ctx, bson.M{"username": username, "deletedAt": nil})
}

// FindByGoogleID fetches a user by Google OAuth ID.
func (r *UserRepository) FindByGoogleID(ctx context.Context, googleID string) (*entity.User, error) {
	return r.findOne(ctx, bson.M{"googleId": googleID, "deletedAt": nil})
}

// List returns a paginated list with optional search.
func (r *UserRepository) List(ctx context.Context, q pagination.Query) ([]*entity.User, int64, error) {
	filter := bson.M{"deletedAt": nil}
	if q.Search != "" {
		filter["$or"] = bson.A{
			bson.M{"email": bson.M{"$regex": q.Search, "$options": "i"}},
			bson.M{"username": bson.M{"$regex": q.Search, "$options": "i"}},
			bson.M{"firstName": bson.M{"$regex": q.Search, "$options": "i"}},
			bson.M{"lastName": bson.M{"$regex": q.Search, "$options": "i"}},
		}
	}

	total, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	sortDir := database.DESC
	if q.SortDir == "asc" {
		sortDir = database.ASC
	}
	sortField := "createdAt"
	if q.SortBy != "" {
		sortField = q.SortBy
	}

	opts := options.Find().
		SetSkip(q.Skip()).
		SetLimit(q.Limit()).
		SetSort(database.BuildSortDoc([]string{sortField}, sortDir))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	var users []*entity.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// Update performs a partial update using $set.
func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, fields bson.M) error {
	fields["updatedAt"] = time.Now()
	_, err := r.col.UpdateOne(
		ctx,
		bson.M{"_id": id, "deletedAt": nil},
		bson.M{"$set": fields},
	)
	return err
}

// SoftDelete sets deletedAt instead of removing the document.
func (r *UserRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return r.Update(ctx, id, bson.M{"deletedAt": time.Now()})
}

// ExistsByEmail returns true if a non-deleted user with the email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	count, err := r.col.CountDocuments(ctx, bson.M{"email": email, "deletedAt": nil})
	return count > 0, err
}

// ExistsByUsername returns true if a non-deleted user with the username exists.
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	count, err := r.col.CountDocuments(ctx, bson.M{"username": username, "deletedAt": nil})
	return count > 0, err
}

// IncrementPasswordAttempts atomically increments failed login attempts.
func (r *UserRepository) IncrementPasswordAttempts(ctx context.Context, id uuid.UUID) error {
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id},
		bson.M{"$inc": bson.M{"passwordAttempts": 1}, "$set": bson.M{"updatedAt": time.Now()}})
	return err
}

// ResetPasswordAttempts clears the counter and lockedUntil.
func (r *UserRepository) ResetPasswordAttempts(ctx context.Context, id uuid.UUID) error {
	return r.Update(ctx, id, bson.M{"passwordAttempts": 0, "lockedUntil": nil})
}

func (r *UserRepository) findOne(ctx context.Context, filter bson.M) (*entity.User, error) {
	var u entity.User
	err := r.col.FindOne(ctx, filter).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &u, err
}
