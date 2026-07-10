package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/breadgarlicbigint/bread-golang-boilerplate/shared/config"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
)

// GitHubUserInfo is the subset of the GitHub user API we care about.
type GitHubUserInfo struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`     // GitHub username
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// GitHubEmail is returned by the /user/emails endpoint.
type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

type GitHubOAuth struct {
	cfg *oauth2.Config
}

func NewGitHubOAuth(cfg config.GitHubConfig) *GitHubOAuth {
	return &GitHubOAuth{
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"user:email", "read:user"},
			Endpoint:     githuboauth.Endpoint,
		},
	}
}

// AuthURL returns the GitHub OAuth redirect URL with a CSRF state token.
func (g *GitHubOAuth) AuthURL(state string) string {
	return g.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange swaps an auth code for tokens and fetches the user profile.
func (g *GitHubOAuth) Exchange(ctx context.Context, code string) (*GitHubUserInfo, error) {
	token, err := g.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("github oauth: exchange: %w", err)
	}

	client := g.cfg.Client(ctx, token)

	info, err := fetchUser(client)
	if err != nil {
		return nil, err
	}

	// If the profile email is empty (user has set it private), fetch from /user/emails
	if info.Email == "" {
		info.Email, err = fetchPrimaryEmail(client)
		if err != nil {
			return nil, err
		}
	}
	return info, nil
}

func fetchUser(client *http.Client) (*GitHubUserInfo, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var info GitHubUserInfo
	return &info, json.Unmarshal(b, &info)
}

func fetchPrimaryEmail(client *http.Client) (string, error) {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var emails []GitHubEmail
	if err := json.Unmarshal(b, &emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("github oauth: no verified primary email found")
}
