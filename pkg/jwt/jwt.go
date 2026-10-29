package jwt

import (
	"crypto/ecdsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenType distinguishes access from refresh tokens.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims is the JWT payload stored in every token.
//
// TenantID is the tenant the user belongs to ("" ⇒ a platform-scoped user with
// no tenant, e.g. a super admin); Permissions is the user's flattened
// per-action permission set ("*" grants everything). Server-side authorization
// reads these — clients only use them to hide controls.
type Claims struct {
	jwt.RegisteredClaims
	UserID      string    `json:"uid"`
	SessionID   string    `json:"sid"`
	Role        string    `json:"role"`
	TenantID    string    `json:"tid,omitempty"`
	Permissions []string  `json:"perms,omitempty"`
	TokenType   TokenType `json:"type"`
}

// Manager holds the key pairs for both token types.
type Manager struct {
	accessPriv  *ecdsa.PrivateKey
	accessPub   *ecdsa.PublicKey
	refreshPriv *ecdsa.PrivateKey
	refreshPub  *ecdsa.PublicKey
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

func New(
	accessPrivPath, accessPubPath string,
	refreshPrivPath, refreshPubPath string,
	accessTTL, refreshTTL time.Duration,
) (*Manager, error) {
	ap, err := loadPrivKey(accessPrivPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: access private key: %w", err)
	}
	aPub, err := loadPubKey(accessPubPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: access public key: %w", err)
	}
	rp, err := loadPrivKey(refreshPrivPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: refresh private key: %w", err)
	}
	rPub, err := loadPubKey(refreshPubPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: refresh public key: %w", err)
	}
	return &Manager{
		accessPriv:  ap,
		accessPub:   aPub,
		refreshPriv: rp,
		refreshPub:  rPub,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}, nil
}

// IssueAccess creates a signed ES256 access token. tenantID is "" for a
// platform-scoped user (no tenant); permissions is the user's flattened set.
func (m *Manager) IssueAccess(userID, sessionID, role, tenantID string, permissions []string) (string, time.Time, error) {
	exp := time.Now().Add(m.accessTTL)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
		UserID:      userID,
		SessionID:   sessionID,
		Role:        role,
		TenantID:    tenantID,
		Permissions: permissions,
		TokenType:   AccessToken,
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(m.accessPriv)
	return tok, exp, err
}

// IssueRefresh creates a signed ES512 refresh token. The refresh token carries
// tenantID + permissions too so a refresh can re-mint an access token without a
// DB round-trip; on rotation the auth service re-reads the user for freshness.
func (m *Manager) IssueRefresh(userID, sessionID, role, tenantID string, permissions []string) (string, time.Time, error) {
	exp := time.Now().Add(m.refreshTTL)
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
		UserID:      userID,
		SessionID:   sessionID,
		Role:        role,
		TenantID:    tenantID,
		Permissions: permissions,
		TokenType:   RefreshToken,
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodES512, claims).SignedString(m.refreshPriv)
	return tok, exp, err
}

// ParseAccess validates and parses an access token.
func (m *Manager) ParseAccess(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.accessPub, jwt.SigningMethodES256)
}

// ParseRefresh validates and parses a refresh token.
func (m *Manager) ParseRefresh(tokenStr string) (*Claims, error) {
	return m.parse(tokenStr, m.refreshPub, jwt.SigningMethodES512)
}

func (m *Manager) parse(tokenStr string, pub *ecdsa.PublicKey, method jwt.SigningMethod) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method != method {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pub, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("jwt: invalid token")
	}
	return claims, nil
}

func loadPrivKey(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseECPrivateKeyFromPEM(b)
}

func loadPubKey(path string) (*ecdsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jwt.ParseECPublicKeyFromPEM(b)
}
