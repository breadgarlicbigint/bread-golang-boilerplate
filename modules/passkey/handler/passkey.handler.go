package handler

import (
	"context"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/errors"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/middleware"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/dto"
	pkentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/passkey/entity"
	userentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"github.com/google/uuid"
)

// PasskeySvc is the service interface the handler depends on.
type PasskeySvc interface {
	BeginRegistration(ctx context.Context, user *userentity.User, attachment string) (interface{}, error)
	FinishRegistration(ctx context.Context, user *userentity.User, friendlyName, attachment string, rawResponse json.RawMessage) (*pkentity.Passkey, error)
	BeginLogin(ctx context.Context, user *userentity.User) (interface{}, error)
	FinishLogin(ctx context.Context, user *userentity.User, rawResponse json.RawMessage) (*pkentity.Passkey, error)
	BeginDiscoverableLogin(ctx context.Context) (interface{}, string, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]*pkentity.Passkey, error)
	Delete(ctx context.Context, passkeyID, userID uuid.UUID) error
}

// UserLoader loads a user by ID from the JWT context.
type UserLoader interface {
	GetByID(ctx context.Context, id string) (*userentity.User, error)
}

type PasskeyHandler struct {
	svc   PasskeySvc
	users UserLoader
}

func New(svc PasskeySvc, users UserLoader) *PasskeyHandler {
	return &PasskeyHandler{svc: svc, users: users}
}

// RegisterRoutes mounts all passkey and biometric endpoints.
func (h *PasskeyHandler) RegisterRoutes(rg *gin.RouterGroup, authMw gin.HandlerFunc) {
	// Usernameless / discoverable login (public)
	pub := rg.Group("/auth/passkey")
	pub.POST("/login/begin", h.BeginDiscoverableLogin)
	pub.POST("/login/finish", h.FinishDiscoverableLogin)

	// Identified login (user supplies email first)
	identified := rg.Group("/auth/passkey/identified")
	identified.POST("/begin", h.BeginIdentifiedLogin)
	identified.POST("/finish", h.FinishIdentifiedLogin)

	// Credential management (requires auth)
	me := rg.Group("/me/passkeys", authMw)
	me.POST("/register/begin", h.BeginRegistration)
	me.POST("/register/finish", h.FinishRegistration)
	me.GET("", h.List)
	me.DELETE("/:id", h.Delete)
}

// BeginRegistration godoc
// @Summary     Begin passkey / biometric registration
// @Description Pass ?attachment=platform for Touch ID / Face ID, cross-platform for hardware keys.
// @Security    BearerAuth
// @Tags        passkeys
// @Param       attachment query string false "platform | cross-platform"
// @Success     200 {object} map[string]interface{}
// @Router      /v1/me/passkeys/register/begin [post]
func (h *PasskeyHandler) BeginRegistration(c *gin.Context) {
	user, err := h.loadCaller(c)
	if err != nil {
		response.Unauthorized(c, "Authentication required")
		return
	}
	attachment := c.DefaultQuery("attachment", "platform")
	opts, err := h.svc.BeginRegistration(c.Request.Context(), user, attachment)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Registration challenge ready", opts)
}

// FinishRegistration godoc
// @Summary     Complete passkey / biometric registration
// @Security    BearerAuth
// @Tags        passkeys
// @Accept      json
// @Param       body body dto.FinishRegistrationRequest true "Attestation response"
// @Success     201 {object} dto.PasskeyResponse
// @Router      /v1/me/passkeys/register/finish [post]
func (h *PasskeyHandler) FinishRegistration(c *gin.Context) {
	user, err := h.loadCaller(c)
	if err != nil {
		response.Unauthorized(c, "Authentication required")
		return
	}
	var req dto.FinishRegistrationRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	rawResp, _ := json.Marshal(req.Response)
	attachment := c.DefaultQuery("attachment", "platform")

	pk, err := h.svc.FinishRegistration(c.Request.Context(), user, req.FriendlyName, attachment, rawResp)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.Created(c, "Passkey registered", toResponse(pk))
}

// BeginDiscoverableLogin godoc
// @Summary     Begin usernameless passkey login
// @Tags        auth
// @Success     200 {object} map[string]interface{}
// @Router      /v1/auth/passkey/login/begin [post]
func (h *PasskeyHandler) BeginDiscoverableLogin(c *gin.Context) {
	opts, sessionToken, err := h.svc.BeginDiscoverableLogin(c.Request.Context())
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Login challenge ready", gin.H{"options": opts, "sessionToken": sessionToken})
}

// FinishDiscoverableLogin godoc
// @Summary     Complete usernameless passkey login
// @Tags        auth
// @Accept      json
// @Param       body body dto.FinishLoginRequest true "Assertion response"
// @Success     200
// @Router      /v1/auth/passkey/login/finish [post]
func (h *PasskeyHandler) FinishDiscoverableLogin(c *gin.Context) {
	var req dto.FinishLoginRequest
	if !validate.BindJSON(c, &req) {
		return
	}
	rawResp, _ := json.Marshal(req.Response)
	pk, err := h.svc.FinishLogin(c.Request.Context(), nil, rawResp)
	if err != nil {
		handleErr(c, err)
		return
	}
	response.OK(c, "Login successful", gin.H{"passkeyId": pk.ID, "userId": pk.UserID})
}

func (h *PasskeyHandler) BeginIdentifiedLogin(c *gin.Context) {
	response.OK(c, "Wire UserLoader.FindByEmail to complete identified login", nil)
}

func (h *PasskeyHandler) FinishIdentifiedLogin(c *gin.Context) {
	response.OK(c, "Wire UserLoader.FindByEmail to complete identified login", nil)
}

// List godoc
// @Summary     List my passkeys
// @Security    BearerAuth
// @Tags        passkeys
// @Success     200 {array} dto.PasskeyResponse
// @Router      /v1/me/passkeys [get]
func (h *PasskeyHandler) List(c *gin.Context) {
	user, err := h.loadCaller(c)
	if err != nil {
		response.Unauthorized(c, "Authentication required")
		return
	}
	passkeys, err := h.svc.ListByUser(c.Request.Context(), user.ID)
	if err != nil {
		handleErr(c, err)
		return
	}
	resp := make([]dto.PasskeyResponse, len(passkeys))
	for i, pk := range passkeys {
		resp[i] = toResponse(pk)
	}
	response.OK(c, "Passkeys fetched", resp)
}

// Delete godoc
// @Summary     Remove a passkey
// @Security    BearerAuth
// @Tags        passkeys
// @Param       id path string true "Passkey ID"
// @Success     204
// @Router      /v1/me/passkeys/{id} [delete]
func (h *PasskeyHandler) Delete(c *gin.Context) {
	user, err := h.loadCaller(c)
	if err != nil {
		response.Unauthorized(c, "Authentication required")
		return
	}
	passkeyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid passkey ID")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), passkeyID, user.ID); err != nil {
		handleErr(c, err)
		return
	}
	response.NoContent(c)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (h *PasskeyHandler) loadCaller(c *gin.Context) (*userentity.User, error) {
	return h.users.GetByID(c.Request.Context(), c.GetString(middleware.CtxUserID))
}

func toResponse(pk *pkentity.Passkey) dto.PasskeyResponse {
	lastUsed := ""
	if pk.LastUsedAt != nil {
		lastUsed = pk.LastUsedAt.UTC().String()
	}
	return dto.PasskeyResponse{
		ID:           pk.ID.String(),
		FriendlyName: pk.FriendlyName,
		Attachment:   string(pk.Attachment),
		IsBiometric:  pk.IsBiometric(),
		BackedUp:     pk.BackedUp,
		LastUsedAt:   lastUsed,
		CreatedAt:    pk.CreatedAt.UTC().String(),
	}
}

func handleErr(c *gin.Context, err error) {
	if ae, ok := errors.As(err); ok {
		response.Error(c, ae.Status, ae.Message)
		return
	}
	response.LogInternal(err, "unexpected error")
	response.InternalServerError(c, "An unexpected error occurred")
}
