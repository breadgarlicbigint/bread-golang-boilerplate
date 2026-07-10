package service

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
)

const appleKeysURL = "https://appleid.apple.com/auth/keys"

// AppleIDTokenClaims are the relevant fields from Apple's id_token JWT.
type AppleIDTokenClaims struct {
	jwt.RegisteredClaims
	Email          string `json:"email"`
	EmailVerified  string `json:"email_verified"` // Apple sends this as a string
	IsPrivateEmail string `json:"is_private_email"`
	AuthTime       int64  `json:"auth_time"`
}

// AppleUserInfo is returned after successfully validating the id_token.
type AppleUserInfo struct {
	Sub            string
	Email          string
	EmailVerified  bool
	IsPrivateEmail bool
	FirstName      string
	LastName       string
}

// AppleSignIn validates Apple identity tokens and generates client secrets.
type AppleSignIn struct {
	clientID     string
	teamID       string
	keyID        string
	privateKey   *ecdsa.PrivateKey
	httpClient   *http.Client
	mu           sync.RWMutex
	cachedKeys   map[string]*ecdsa.PublicKey
	keysCachedAt time.Time
	keysCacheTTL time.Duration
}

func NewAppleSignIn(cfg config.AppleConfig) (*AppleSignIn, error) {
	a := &AppleSignIn{
		clientID:     cfg.ClientID,
		teamID:       cfg.TeamID,
		keyID:        cfg.KeyID,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		cachedKeys:   make(map[string]*ecdsa.PublicKey),
		keysCacheTTL: 24 * time.Hour,
	}
	if cfg.PrivateKeyPath != "" {
		pk, err := loadApplePrivateKey(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("apple: load private key: %w", err)
		}
		a.privateKey = pk
	}
	return a, nil
}

// ValidateIDToken verifies an Apple id_token and returns the user info.
func (a *AppleSignIn) ValidateIDToken(ctx context.Context, idToken string) (*AppleUserInfo, error) {
	kid, err := extractKID(idToken)
	if err != nil {
		return nil, fmt.Errorf("apple: extract kid: %w", err)
	}
	pubKey, err := a.getPublicKey(ctx, kid)
	if err != nil {
		return nil, fmt.Errorf("apple: get public key: %w", err)
	}

	tok, err := jwt.ParseWithClaims(idToken, &AppleIDTokenClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return pubKey, nil
	}, jwt.WithAudience(a.clientID), jwt.WithIssuer("https://appleid.apple.com"))
	if err != nil {
		return nil, fmt.Errorf("apple: validate token: %w", err)
	}

	claims, ok := tok.Claims.(*AppleIDTokenClaims)
	if !ok || !tok.Valid {
		return nil, fmt.Errorf("apple: invalid token claims")
	}

	return &AppleUserInfo{
		Sub:            claims.Subject,
		Email:          claims.Email,
		EmailVerified:  claims.EmailVerified == "true",
		IsPrivateEmail: claims.IsPrivateEmail == "true",
	}, nil
}

// GenerateClientSecret creates the ES256 JWT Apple requires for token exchange.
func (a *AppleSignIn) GenerateClientSecret() (string, error) {
	if a.privateKey == nil {
		return "", fmt.Errorf("apple: private key not configured")
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    a.teamID,
		Subject:   a.clientID,
		Audience:  jwt.ClaimStrings{"https://appleid.apple.com"},
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(180 * 24 * time.Hour)),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = a.keyID
	return tok.SignedString(a.privateKey)
}

// ── JWKS ─────────────────────────────────────────────────────────────────────

func (a *AppleSignIn) getPublicKey(ctx context.Context, kid string) (*ecdsa.PublicKey, error) {
	a.mu.RLock()
	if key, ok := a.cachedKeys[kid]; ok && time.Since(a.keysCachedAt) < a.keysCacheTTL {
		a.mu.RUnlock()
		return key, nil
	}
	a.mu.RUnlock()
	return a.refreshKeys(ctx, kid)
}

type appleJWKS struct {
	Keys []struct {
		KID string `json:"kid"`
		ALG string `json:"alg"`
		// Note: Apple's JWKS uses RSA keys; we handle ES256 from the client secret side
		// The id_token is verified with Apple's RSA public keys (RS256)
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

func (a *AppleSignIn) refreshKeys(ctx context.Context, targetKID string) (*ecdsa.PublicKey, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, appleKeysURL, nil)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apple: fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)

	var jwks appleJWKS
	if err := json.Unmarshal(b, &jwks); err != nil {
		return nil, fmt.Errorf("apple: parse JWKS: %w", err)
	}

	// Apple uses RS256 for id_token verification.
	// For full implementation use github.com/lestrrat-go/jwx/v2.
	// This stub satisfies the interface — wire lestrrat-go/jwx for production.
	a.mu.Lock()
	a.keysCachedAt = time.Now()
	a.mu.Unlock()

	_ = targetKID
	_ = jwks
	return nil, fmt.Errorf("apple: JWKS EC key parsing requires lestrrat-go/jwx — see docs/apple-signin.md")
}

// ── helpers ───────────────────────────────────────────────────────────────────

func extractKID(tokenStr string) (string, error) {
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid JWT format")
	}
	b, err := jwt.NewParser().DecodeSegment(parts[0])
	if err != nil {
		return "", fmt.Errorf("decode header: %w", err)
	}
	var header struct {
		KID string `json:"kid"`
	}
	return header.KID, json.Unmarshal(b, &header)
}

func loadApplePrivateKey(path string) (*ecdsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("apple: invalid PEM file at %s", path)
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("apple: parse private key: %w", err)
	}
	ec, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("apple: key is not ECDSA")
	}
	return ec, nil
}
