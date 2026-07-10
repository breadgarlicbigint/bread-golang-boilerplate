package dto

// ── Registration ──────────────────────────────────────────────────────────────

// BeginRegistrationResponse is sent to the client to start a registration ceremony.
type BeginRegistrationResponse struct {
	Challenge         string            `json:"challenge"`
	RelyingParty      RelyingParty      `json:"rp"`
	User              PasskeyUser       `json:"user"`
	PubKeyCredParams  []PubKeyCredParam `json:"pubKeyCredParams"`
	Timeout           int               `json:"timeout"`
	AttestationType   string            `json:"attestation"`
	ExcludeCredentials []CredDescriptor `json:"excludeCredentials"`
	AuthenticatorSelection AuthenticatorSelection `json:"authenticatorSelection"`
	SessionToken      string            `json:"sessionToken"` // opaque token to bind ceremony
}

type RelyingParty struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PasskeyUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type PubKeyCredParam struct {
	Type string `json:"type"`
	Alg  int    `json:"alg"`
}

type AuthenticatorSelection struct {
	AuthenticatorAttachment string `json:"authenticatorAttachment,omitempty"` // "platform" | "cross-platform"
	ResidentKey             string `json:"residentKey"`                       // "required" | "preferred" | "discouraged"
	UserVerification        string `json:"userVerification"`                  // "required" | "preferred" | "discouraged"
}

type CredDescriptor struct {
	Type      string   `json:"type"`
	ID        string   `json:"id"`
	Transports []string `json:"transports,omitempty"`
}

// FinishRegistrationRequest is sent from the client after creating the credential.
type FinishRegistrationRequest struct {
	SessionToken string      `json:"sessionToken" validate:"required"`
	FriendlyName string      `json:"friendlyName" validate:"required,min=1,max=50"`
	Response     interface{} `json:"response"     validate:"required"` // raw AuthenticatorAttestationResponse JSON
}

// ── Authentication ────────────────────────────────────────────────────────────

// BeginLoginResponse is sent to the client to start an authentication ceremony.
type BeginLoginResponse struct {
	Challenge          string           `json:"challenge"`
	Timeout            int              `json:"timeout"`
	RelyingPartyID     string           `json:"rpId"`
	AllowCredentials   []CredDescriptor `json:"allowCredentials"`
	UserVerification   string           `json:"userVerification"`
	SessionToken       string           `json:"sessionToken"`
}

// FinishLoginRequest is sent from the client after signing the challenge.
type FinishLoginRequest struct {
	SessionToken string      `json:"sessionToken" validate:"required"`
	Response     interface{} `json:"response"     validate:"required"` // raw AuthenticatorAssertionResponse JSON
}

// ── List ──────────────────────────────────────────────────────────────────────

type PasskeyResponse struct {
	ID           string `json:"id"`
	FriendlyName string `json:"friendlyName"`
	Attachment   string `json:"attachment"`
	IsBiometric  bool   `json:"isBiometric"`
	BackedUp     bool   `json:"backedUp"`
	LastUsedAt   string `json:"lastUsedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
}
