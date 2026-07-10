package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	dto     "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/dto"
	authSvc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/auth/service"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/response"
	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/validate"
	userentity "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/user/entity"
	"go.mongodb.org/mongo-driver/bson"
)

const (
	githubStatePrefix = "gh:state:"
	githubStateTTL    = 10 * time.Minute
)

// GitHubUserRepo is the subset of the user repository that the GitHub handler needs.
type GitHubUserRepo interface {
	FindByEmail(ctx context.Context, email string) (*userentity.User, error)
	FindByGoogleID(ctx context.Context, id string) (*userentity.User, error)
	Create(ctx context.Context, u *userentity.User) error
	Update(ctx context.Context, id uuid.UUID, fields bson.M) error
}

type GitHubHandler struct {
	gh      *authSvc.GitHubOAuth
	repo    GitHubUserRepo
	rdb     *redis.Client
	authSvc *authSvc.AuthService
}

func NewGitHub(gh *authSvc.GitHubOAuth, repo GitHubUserRepo, rdb *redis.Client, as *authSvc.AuthService) *GitHubHandler {
	return &GitHubHandler{gh: gh, repo: repo, rdb: rdb, authSvc: as}
}

func (h *GitHubHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/auth/github", h.Redirect)
	rg.GET("/auth/github/callback", h.Callback)
}

// Redirect godoc
// @Summary     Redirect to GitHub OAuth
// @Tags        auth
// @Success     302
// @Router      /v1/auth/github [get]
func (h *GitHubHandler) Redirect(c *gin.Context) {
	state := uuid.NewString()
	_ = h.rdb.Set(c.Request.Context(), githubStatePrefix+state, "1", githubStateTTL)
	c.Redirect(http.StatusTemporaryRedirect, h.gh.AuthURL(state))
}

// Callback godoc
// @Summary     GitHub OAuth callback
// @Tags        auth
// @Param       code  query string true "Auth code"
// @Param       state query string true "CSRF state"
// @Success     200 {object} dto.LoginResponse
// @Router      /v1/auth/github/callback [get]
func (h *GitHubHandler) Callback(c *gin.Context) {
	state := c.Query("state")
	code := c.Query("code")
	if state == "" || code == "" {
		response.BadRequest(c, "Missing state or code parameter")
		return
	}

	// Validate CSRF state
	if err := h.rdb.Del(c.Request.Context(), githubStatePrefix+state).Err(); err != nil {
		response.Unauthorized(c, "Invalid OAuth state — possible CSRF attack")
		return
	}

	ghUser, err := h.gh.Exchange(c.Request.Context(), code)
	if err != nil {
		response.Unauthorized(c, "GitHub authentication failed: "+err.Error())
		return
	}
	if ghUser.Email == "" {
		response.BadRequest(c, "GitHub account has no verified public email. Please add one at github.com/settings/emails")
		return
	}

	ctx := c.Request.Context()
	githubID := fmt.Sprintf("%d", ghUser.ID)

	// Try by GitHub ID first, then email (account linking)
	user, _ := h.repo.FindByGoogleID(ctx, githubID)
	if user == nil {
		user, _ = h.repo.FindByEmail(ctx, ghUser.Email)
	}

	if user == nil {
		// Auto-register
		nameParts := strings.SplitN(ghUser.Name, " ", 2)
		firstName := nameParts[0]
		lastName := ""
		if len(nameParts) > 1 {
			lastName = nameParts[1]
		}
		user = &userentity.User{
			Email:          ghUser.Email,
			Username:       sanitiseGitHubUsername(ghUser.Login),
			FirstName:      firstName,
			LastName:       lastName,
			ProfilePicture: ghUser.AvatarURL,
			GoogleID:       githubID, // stored in GoogleID field; rename in entity for full multi-social support
			Status:         userentity.UserStatusActive,
			EmailVerified:  true,
			NotifSettings:  userentity.DefaultNotifSettings(),
		}
		if err := h.repo.Create(ctx, user); err != nil {
			response.LogInternal(err, "github oauth: create account failed")
			response.InternalServerError(c, "Failed to create account")
			return
		}
	} else if user.GoogleID == "" {
		_ = h.repo.Update(ctx, user.ID, bson.M{"googleId": githubID, "emailVerified": true, "updatedAt": time.Now()})
	}

	loginResp, err := h.authSvc.LoginWithGitHub(ctx, user, c.ClientIP())
	if err != nil {
		response.LogInternal(err, "github oauth: issue tokens failed")
		response.InternalServerError(c, "Failed to issue tokens")
		return
	}
	response.OK(c, "GitHub login successful", loginResp)
}

func sanitiseGitHubUsername(login string) string {
	return strings.ReplaceAll(login, "-", "_")
}

// ── Apple handler (same file — both in auth/handler package) ─────────────────

// AppleUserRepo is the subset the Apple handler needs.
type AppleUserRepo interface {
	FindByEmail(ctx context.Context, email string) (*userentity.User, error)
	Create(ctx context.Context, u *userentity.User) error
	Update(ctx context.Context, id uuid.UUID, fields bson.M) error
}

type AppleHandler struct {
	apple   *authSvc.AppleSignIn
	repo    AppleUserRepo
	authSvc *authSvc.AuthService
}

func NewApple(apple *authSvc.AppleSignIn, repo AppleUserRepo, as *authSvc.AuthService) *AppleHandler {
	return &AppleHandler{apple: apple, repo: repo, authSvc: as}
}

func (h *AppleHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/auth/apple/callback", h.Callback)
}

// Callback godoc
// @Summary     Apple Sign In callback
// @Tags        auth
// @Accept      json
// @Produce     json
// @Success     200 {object} dto.LoginResponse
// @Router      /v1/auth/apple/callback [post]
func (h *AppleHandler) Callback(c *gin.Context) {
	var req struct {
		IdentityToken string `json:"identityToken" binding:"required"`
		FirstName     string `json:"firstName"`
		LastName      string `json:"lastName"`
	}
	if !validate.BindJSON(c, &req) {
		return
	}

	appleUser, err := h.apple.ValidateIDToken(c.Request.Context(), req.IdentityToken)
	if err != nil {
		response.Unauthorized(c, "Apple authentication failed: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	user, _ := h.repo.FindByEmail(ctx, appleUser.Email)
	if user == nil {
		firstName := req.FirstName
		if firstName == "" {
			firstName = "Apple"
		}
		user = &userentity.User{
			Email:         appleUser.Email,
			Username:      "apple_" + strings.ReplaceAll(appleUser.Sub[:min(12, len(appleUser.Sub))], ".", "_"),
			FirstName:     firstName,
			LastName:      req.LastName,
			AppleID:       appleUser.Sub,
			Status:        userentity.UserStatusActive,
			EmailVerified: appleUser.EmailVerified,
			NotifSettings: userentity.DefaultNotifSettings(),
		}
		if err := h.repo.Create(ctx, user); err != nil {
			response.LogInternal(err, "apple oauth: create account failed")
			response.InternalServerError(c, "Failed to create account")
			return
		}
	} else if user.AppleID == "" {
		_ = h.repo.Update(ctx, user.ID, bson.M{"appleId": appleUser.Sub, "updatedAt": time.Now()})
	}

	loginResp, err := h.authSvc.IssueTokenPairPublic(ctx, user, c.ClientIP())
	if err != nil {
		response.LogInternal(err, "apple oauth: issue tokens failed")
		response.InternalServerError(c, "Failed to issue tokens")
		return
	}
	response.OK(c, "Apple login successful", loginResp)
}

// Unused but referenced in dto

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Ensure dto types are reachable for swagger annotation parsing.
var _ dto.LoginResponse
