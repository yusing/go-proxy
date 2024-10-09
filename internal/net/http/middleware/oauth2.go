package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	E "github.com/yusing/go-proxy/internal/error"
)

type oAuth2 struct {
	*oAuth2Opts
	m *Middleware
}

type oAuth2Opts struct {
	ClientID     string
	ClientSecret string
	AuthURL      string // Authorization Endpoint
	TokenURL     string // Token Endpoint
}

var OAuth2 = &oAuth2{
	m: &Middleware{withOptions: NewAuthentikOAuth2},
}

func NewAuthentikOAuth2(opts OptionsRaw) (*Middleware, E.NestedError) {
	oauth := new(oAuth2)
	oauth.m = &Middleware{
		impl:   oauth,
		before: oauth.handleOAuth2,
	}
	oauth.oAuth2Opts = &oAuth2Opts{}
	err := Deserialize(opts, oauth.oAuth2Opts)
	if err != nil {
		return nil, err
	}
	b := E.NewBuilder("missing required fields")
	optV := reflect.ValueOf(oauth.oAuth2Opts)
	for _, field := range reflect.VisibleFields(reflect.TypeFor[oAuth2Opts]()) {
		if optV.FieldByName(field.Name).Len() == 0 {
			b.Add(E.Missing(field.Name))
		}
	}
	if b.HasError() {
		return nil, b.Build().Subject("oAuth2")
	}
	return oauth.m, nil
}

func (oauth *oAuth2) handleOAuth2(next http.HandlerFunc, rw ResponseWriter, r *Request) {
	// Check if the user is authenticated (you may use session, cookie, etc.)
	if !userIsAuthenticated(r) {
		// TODO: Redirect to OAuth2 auth URL
		http.Redirect(rw, r, fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code",
			oauth.oAuth2Opts.AuthURL, oauth.oAuth2Opts.ClientID, ""), http.StatusFound)
		return
	}

	// If you have a token in the query string, process it
	if code := r.URL.Query().Get("code"); code != "" {
		// Exchange the authorization code for a token here
		// Use the TokenURL and authenticate the user
		token, err := exchangeCodeForToken(code, oauth.oAuth2Opts, r.RequestURI)
		if err != nil {
			// handle error
			http.Error(rw, "failed to get token", http.StatusUnauthorized)
			return
		}

		// Save token and user info based on your requirements
		saveToken(rw, token)

		// Redirect to the originally requested URL
		http.Redirect(rw, r, "/", http.StatusFound)
		return
	}

	// If user is authenticated, go to the next handler
	next(rw, r)
}

func userIsAuthenticated(r *http.Request) bool {
	// Example: Check for a session or cookie
	session, err := r.Cookie("session_token")
	if err != nil || session.Value == "" {
		return false
	}
	// Validate the session_token if necessary
	return true
}

func exchangeCodeForToken(code string, opts *oAuth2Opts, requestUri string) (string, error) {
	// Prepare the request body
	data := url.Values{
		"client_id":     {opts.ClientID},
		"client_secret": {opts.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {requestUri},
	}
	resp, err := http.PostForm(opts.TokenURL, data)
	if err != nil {
		return "", fmt.Errorf("failed to request token: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("received non-ok status from token endpoint: %s", resp.Status)
	}
	// Decode the response
	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %v", err)
	}
	return tokenResp.AccessToken, nil
}

func saveToken(rw ResponseWriter, token string) {
	// Example: Save token in cookie
	http.SetCookie(rw, &http.Cookie{
		Name:  "auth_token",
		Value: token,
		// set other properties as necessary, such as Secure and HttpOnly
	})
}