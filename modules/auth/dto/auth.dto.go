package dto

// ── Login ─────────────────────────────────────────────────────────────────────

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	TokenType    string       `json:"tokenType"`
	ExpiresIn    int64        `json:"expiresIn"` // seconds
	User         UserPayload  `json:"user"`
}

type UserPayload struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// ── Register ──────────────────────────────────────────────────────────────────

type RegisterRequest struct {
	Email     string `json:"email"     validate:"required,email"`
	Username  string `json:"username"  validate:"required,min=3,max=30,alphanum"`
	Password  string `json:"password"  validate:"required,min=8"`
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName"  validate:"required"`
}

// ── Token refresh ─────────────────────────────────────────────────────────────

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ── Password ──────────────────────────────────────────────────────────────────

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"    validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

// ── 2FA ───────────────────────────────────────────────────────────────────────

type Enable2FAResponse struct {
	Secret     string   `json:"secret"`
	QRCodeURL  string   `json:"qrCodeUrl"`
	BackupCodes []string `json:"backupCodes"`
}

type Verify2FARequest struct {
	Code string `json:"code" validate:"required,len=6"`
}

// ── Social ────────────────────────────────────────────────────────────────────

type GoogleCallbackRequest struct {
	Code  string `form:"code"  validate:"required"`
	State string `form:"state" validate:"required"`
}

type AppleCallbackRequest struct {
	Code      string `form:"code"`
	IDToken   string `form:"id_token"`
	FirstName string `form:"firstName"`
	LastName  string `form:"lastName"`
}
