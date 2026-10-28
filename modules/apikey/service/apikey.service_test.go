package service

import (
	"context"
	"strings"
	"testing"

	apperrors "github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
)

// Create looks a key up by its first 8 chars and the first active match wins.
// With a short prefix only a couple of those are random, so two active keys can
// share a lookup prefix and the second is rejected on every request. freePrefix
// must re-roll past a collision rather than mint a dead key.
func TestFreePrefix_ReRollsPastACollision(t *testing.T) {
	s := &APIKeyService{prefix: "bread"}
	calls := 0
	// The first two candidate prefixes are "taken"; the third is free.
	s.prefixTaken = func(ctx context.Context, prefix8 string) (bool, error) {
		calls++
		return calls <= 2, nil
	}

	raw, err := s.freePrefix(context.Background())
	if err != nil {
		t.Fatalf("freePrefix: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 attempts (2 collisions then free), got %d", calls)
	}
	if !strings.HasPrefix(raw, "bread_") {
		t.Errorf("raw key %q lacks the configured prefix", raw)
	}
}

// When every lookup prefix is already in use, Create must refuse with a bounded
// error rather than loop forever or mint a colliding key.
func TestFreePrefix_ExhaustsAfterMaxAttempts(t *testing.T) {
	s := &APIKeyService{prefix: "bread"}
	calls := 0
	s.prefixTaken = func(ctx context.Context, prefix8 string) (bool, error) {
		calls++
		return true, nil // everything is taken
	}

	_, err := s.freePrefix(context.Background())
	if err != ErrKeyPrefixExhausted {
		t.Fatalf("err = %v, want ErrKeyPrefixExhausted", err)
	}
	if calls != maxPrefixAttempts {
		t.Errorf("expected exactly %d attempts, got %d", maxPrefixAttempts, calls)
	}
	if ae, ok := apperrors.As(err); !ok || ae.Status != 500 {
		t.Errorf("ErrKeyPrefixExhausted should be a 500 AppError, got %v", err)
	}
}
