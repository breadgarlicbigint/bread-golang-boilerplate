package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/redis/go-redis/v9"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	pkentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/entity"
	userentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/google/uuid"
)

const (
	regSessionPrefix  = "passkey:reg:"
	authSessionPrefix = "passkey:auth:"
	ceremonyTTL       = 5 * time.Minute
)

// ── WebAuthn user adapter ─────────────────────────────────────────────────────

type webAuthnUser struct {
	user     *userentity.User
	passkeys []*pkentity.Passkey
}

func (w *webAuthnUser) WebAuthnID() []byte          { return []byte(w.user.ID.String()) }
func (w *webAuthnUser) WebAuthnName() string        { return w.user.Email }
func (w *webAuthnUser) WebAuthnDisplayName() string { return w.user.FullName() }
func (w *webAuthnUser) WebAuthnIcon() string        { return "" }

func (w *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, 0, len(w.passkeys))
	for _, pk := range w.passkeys {
		transports := make([]protocol.AuthenticatorTransport, 0, len(pk.Transport))
		for _, t := range pk.Transport {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              pk.CredentialID,
			PublicKey:       pk.PublicKey,
			AttestationType: pk.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				SignCount: pk.SignCount,
			},
		})
	}
	return creds
}

// ── Repository interface ──────────────────────────────────────────────────────

type PasskeyRepo interface {
	Create(ctx context.Context, p *pkentity.Passkey) error
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]*pkentity.Passkey, error)
	FindByCredentialID(ctx context.Context, credIDBase64 string) (*pkentity.Passkey, error)
	UpdateSignCount(ctx context.Context, id uuid.UUID, signCount uint32) error
	Delete(ctx context.Context, id, userID uuid.UUID) error
}

// ── Service ───────────────────────────────────────────────────────────────────

type PasskeyService struct {
	repo    PasskeyRepo
	webAuth *webauthn.WebAuthn
	rdb     *redis.Client
}

func New(repo PasskeyRepo, rdb *redis.Client, rpID, rpOrigin, rpName string) (*PasskeyService, error) {
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpName,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, fmt.Errorf("passkey: webauthn init: %w", err)
	}
	return &PasskeyService{repo: repo, webAuth: wa, rdb: rdb}, nil
}

// ── Registration ──────────────────────────────────────────────────────────────

func (s *PasskeyService) BeginRegistration(ctx context.Context, user *userentity.User, attachment string) (interface{}, error) {
	passkeys, err := s.repo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	waUser := &webAuthnUser{user: user, passkeys: passkeys}

	creation, session, err := s.webAuth.BeginRegistration(waUser,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.AuthenticatorAttachment(attachment),
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			UserVerification:        protocol.VerificationRequired,
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("passkey: begin registration: %w", err)
	}

	sessionBytes, _ := json.Marshal(session)
	_ = s.rdb.Set(ctx, regSessionPrefix+user.ID.String(), sessionBytes, ceremonyTTL)
	return creation, nil
}

func (s *PasskeyService) FinishRegistration(ctx context.Context, user *userentity.User, friendlyName, attachment string, rawResponse json.RawMessage) (*pkentity.Passkey, error) {
	passkeys, err := s.repo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	waUser := &webAuthnUser{user: user, passkeys: passkeys}

	key := regSessionPrefix + user.ID.String()
	sessionBytes, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, errors.NewI18n(400, "CEREMONY_EXPIRED", "passkey.ceremonyExpired", "Registration session expired. Please start again.")
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		return nil, errors.ErrInternal
	}
	_ = s.rdb.Del(ctx, key)

	// ParseCredentialCreationResponseBody takes an io.Reader
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(rawResponse))
	if err != nil {
		return nil, errors.NewI18n(400, "PASSKEY_INVALID", "passkey.invalidAttestation", "Invalid attestation response: "+err.Error())
	}

	credential, err := s.webAuth.CreateCredential(waUser, session, parsed)
	if err != nil {
		return nil, errors.NewI18n(400, "PASSKEY_INVALID", "passkey.invalidAttestation", "Passkey registration failed: "+err.Error())
	}

	transports := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	pk := &pkentity.Passkey{
		UserID:             user.ID,
		CredentialID:       credential.ID,
		CredentialIDBase64: base64.URLEncoding.EncodeToString(credential.ID),
		PublicKey:          credential.PublicKey,
		AttestationType:    credential.AttestationType,
		Transport:          transports,
		Attachment:         pkentity.AuthenticatorAttachment(attachment),
		SignCount:          credential.Authenticator.SignCount,
		FriendlyName:       friendlyName,
		AAGUID:             fmt.Sprintf("%x", credential.Authenticator.AAGUID),
		BackedUp:           credential.Flags.BackupEligible,
	}

	if err := s.repo.Create(ctx, pk); err != nil {
		return nil, err
	}
	return pk, nil
}

// ── Authentication ────────────────────────────────────────────────────────────

func (s *PasskeyService) BeginLogin(ctx context.Context, user *userentity.User) (interface{}, error) {
	passkeys, err := s.repo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if len(passkeys) == 0 {
		return nil, errors.NewI18n(400, "NO_PASSKEYS", "passkey.noPasskeys", "No passkeys registered for this account")
	}
	waUser := &webAuthnUser{user: user, passkeys: passkeys}

	assertion, session, err := s.webAuth.BeginLogin(waUser)
	if err != nil {
		return nil, fmt.Errorf("passkey: begin login: %w", err)
	}

	sessionBytes, _ := json.Marshal(session)
	_ = s.rdb.Set(ctx, authSessionPrefix+user.ID.String(), sessionBytes, ceremonyTTL)
	return assertion, nil
}

func (s *PasskeyService) FinishLogin(ctx context.Context, user *userentity.User, rawResponse json.RawMessage) (*pkentity.Passkey, error) {
	if user == nil {
		return nil, errors.ErrUnauthorized
	}
	passkeys, err := s.repo.FindByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	waUser := &webAuthnUser{user: user, passkeys: passkeys}

	key := authSessionPrefix + user.ID.String()
	sessionBytes, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, errors.NewI18n(400, "CEREMONY_EXPIRED", "passkey.ceremonyExpired", "Login session expired. Please start again.")
	}
	var session webauthn.SessionData
	if err := json.Unmarshal(sessionBytes, &session); err != nil {
		return nil, errors.ErrInternal
	}
	_ = s.rdb.Del(ctx, key)

	// ParseCredentialRequestResponseBody takes an io.Reader
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(rawResponse))
	if err != nil {
		return nil, errors.NewI18n(400, "PASSKEY_AUTH_FAILED", "passkey.authFailed", "Invalid assertion response: "+err.Error())
	}

	credential, err := s.webAuth.ValidateLogin(waUser, session, parsed)
	if err != nil {
		return nil, errors.NewI18n(401, "PASSKEY_AUTH_FAILED", "passkey.authFailed", "Passkey authentication failed: "+err.Error())
	}

	credIDBase64 := base64.URLEncoding.EncodeToString(credential.ID)
	pk, err := s.repo.FindByCredentialID(ctx, credIDBase64)
	if err != nil || pk == nil {
		return nil, errors.ErrUnauthorized
	}
	_ = s.repo.UpdateSignCount(ctx, pk.ID, credential.Authenticator.SignCount)
	pk.SignCount = credential.Authenticator.SignCount
	return pk, nil
}

func (s *PasskeyService) BeginDiscoverableLogin(ctx context.Context) (interface{}, string, error) {
	sessionID := fmt.Sprintf("disc:%d", time.Now().UnixNano())
	assertion, session, err := s.webAuth.BeginDiscoverableLogin()
	if err != nil {
		return nil, "", err
	}
	sessionBytes, _ := json.Marshal(session)
	_ = s.rdb.Set(ctx, authSessionPrefix+sessionID, sessionBytes, ceremonyTTL)
	return assertion, sessionID, nil
}

func (s *PasskeyService) ListByUser(ctx context.Context, userID uuid.UUID) ([]*pkentity.Passkey, error) {
	return s.repo.FindByUserID(ctx, userID)
}

func (s *PasskeyService) Delete(ctx context.Context, passkeyID, userID uuid.UUID) error {
	return s.repo.Delete(ctx, passkeyID, userID)
}
