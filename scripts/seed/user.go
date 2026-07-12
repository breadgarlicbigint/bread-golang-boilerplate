package main

import (
	"context"
	"fmt"
	"log"
	"time"

	userEntity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/hash"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/database"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// seedUser creates a single user with the given role, hashing password with h.
func seedUser(
	ctx context.Context, db *database.MongoDB, h *hash.Hasher,
	email, username, firstName, lastName, password string,
	roleID uuid.UUID, emailVerified bool,
) *userEntity.User {
	col := db.Collection("users")

	pw, err := h.Hash(password)
	if err != nil {
		log.Printf("  ⚠ hash %s: %v", email, err)
		return nil
	}

	now := time.Now()
	u := &userEntity.User{
		ID:            uuid.New(),
		Email:         email,
		Username:      username,
		PasswordHash:  pw,
		FirstName:     firstName,
		LastName:      lastName,
		Status:        userEntity.UserStatusActive,
		RoleID:        roleID,
		EmailVerified: emailVerified,
		NotifSettings: userEntity.DefaultNotifSettings(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := upsert(ctx, col, bson.M{"email": email}, u); err != nil {
		log.Printf("  ⚠ user %s: %v", email, err)
	} else {
		fmt.Printf("  user: %-30s (password: %s) ✅\n", email, password)
	}
	return u
}
