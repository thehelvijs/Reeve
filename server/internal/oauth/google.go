// Package oauth implements the OAuth 2.0 authorization-code flow against
// Google's endpoints with the standard library only. It deliberately reads the
// profile from the userinfo endpoint rather than parsing an id_token, so no JWT
// or JWKS handling is needed.
package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Google's production endpoints.
const (
	GoogleAuthURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	GoogleTokenURL    = "https://oauth2.googleapis.com/token"
	GoogleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"
)

// requestTimeout bounds each leg of the exchange.
const requestTimeout = 15 * time.Second

// maxBody caps how much a provider response may be, so a hostile or broken
// endpoint cannot balloon the server's memory.
const maxBody = 1 << 20

// Config is one OAuth client. The three URLs default to Google's and are
// overridable so tests can point at a local stub.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Client       *http.Client
}

// Google returns a Config wired to Google's endpoints.
func Google(clientID, clientSecret, redirectURL string) Config {
	return Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		AuthURL:      GoogleAuthURL,
		TokenURL:     GoogleTokenURL,
		UserInfoURL:  GoogleUserInfoURL,
	}
}

// Profile is the subset of the provider's user info this app acts on.
type Profile struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
}

// Validate reports whether the client is configured well enough to use.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return errors.New("client id is required")
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		return errors.New("client secret is required")
	}
	if strings.TrimSpace(c.RedirectURL) == "" {
		return errors.New("redirect url is required")
	}
	return nil
}

// AuthCodeURL builds the URL the browser is sent to. prompt=select_account
// keeps a shared browser from silently reusing the last Google session.
func (c Config) AuthCodeURL(state string) string {
	q := url.Values{
		"client_id":     {c.ClientID},
		"redirect_uri":  {c.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"prompt":        {"select_account"},
	}
	return c.AuthURL + "?" + q.Encode()
}

func (c Config) httpClient() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: requestTimeout}
}

// Exchange trades an authorization code for an access token.
func (c Config) Exchange(ctx context.Context, code string) (string, error) {
	if code == "" {
		return "", errors.New("no authorization code")
	}
	form := url.Values{
		"code":          {code},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"redirect_uri":  {c.RedirectURL},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	body, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	var out struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("token exchange: malformed response: %w", err)
	}
	if out.AccessToken == "" {
		return "", errors.New("token exchange: response carried no access token")
	}
	return out.AccessToken, nil
}

// UserInfo fetches the signed-in user's profile. An unverified email is
// rejected: it would let anyone claim a colleague's address.
func (c Config) UserInfo(ctx context.Context, accessToken string) (Profile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.UserInfoURL, nil)
	if err != nil {
		return Profile{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	body, err := c.do(req)
	if err != nil {
		return Profile{}, fmt.Errorf("userinfo: %w", err)
	}
	var p Profile
	if err := json.Unmarshal(body, &p); err != nil {
		return Profile{}, fmt.Errorf("userinfo: malformed response: %w", err)
	}
	p.Email = strings.TrimSpace(strings.ToLower(p.Email))
	if p.Email == "" || !strings.Contains(p.Email, "@") {
		return Profile{}, errors.New("userinfo: no email on the account")
	}
	// The subject is what an account is pinned to; without it there is nothing
	// stable to recognize this identity by on the next sign-in.
	if strings.TrimSpace(p.Subject) == "" {
		return Profile{}, errors.New("userinfo: the provider returned no subject id")
	}
	if !p.EmailVerified {
		return Profile{}, errors.New("userinfo: the provider has not verified that email address")
	}
	return p, nil
}

func (c Config) do(req *http.Request) ([]byte, error) {
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("provider returned %d", resp.StatusCode)
	}
	return body, nil
}

// DomainAllowed reports whether email's domain passes an allow-list. An empty
// list allows any domain.
func DomainAllowed(email string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	_, domain, ok := strings.Cut(strings.ToLower(email), "@")
	if !ok {
		return false
	}
	for _, a := range allowed {
		if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(a, "@")), domain) {
			return true
		}
	}
	return false
}

// ParseDomains splits an operator-entered domain list on commas or whitespace.
func ParseDomains(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == ';'
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(f), "@"))
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}
