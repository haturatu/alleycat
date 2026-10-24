package pbapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const chatGPTResource = "https://api.openai.com/v1"

var (
	chatGPTAuthorizeURL  = "https://auth.openai.com/api/accounts/authorize"
	chatGPTTokenURL      = "https://auth.openai.com/api/accounts/oauth/token"
	chatGPTJWKSURL       = "https://auth.openai.com/.well-known/jwks.json"
	chatGPTDiscoveryURL  = "https://auth.openai.com/.well-known/openid-configuration"
	chatGPTClient        = &http.Client{Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	chatGPTCredentialsMu sync.Mutex
)

type chatGPTCredentials struct {
	HostID       string    `json:"host_id"`
	ClientID     string    `json:"client_id"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email"`
	IDToken      string    `json:"id_token"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type chatGPTTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
}

type chatGPTIdentity struct {
	jwt.RegisteredClaims
	Nonce string `json:"nonce"`
	Email string `json:"email"`
}

func loadChatGPTCredentials(app core.App) (chatGPTCredentials, error) {
	var credentials chatGPTCredentials
	secret, err := app.FindFirstRecordByFilter("app_secrets", "id != ''")
	if errors.Is(err, sql.ErrNoRows) {
		return credentials, nil
	}
	if err != nil {
		return credentials, err
	}
	if raw := secret.GetString("chatgpt_oauth"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &credentials); err != nil {
			return credentials, errors.New("ChatGPT connection data is invalid; reconnect in Settings")
		}
	}
	return credentials, nil
}

func saveChatGPTCredentials(app core.App, credentials chatGPTCredentials) error {
	secret, err := app.FindFirstRecordByFilter("app_secrets", "id != ''")
	if errors.Is(err, sql.ErrNoRows) {
		collection, findErr := app.FindCollectionByNameOrId("app_secrets")
		if findErr != nil {
			return findErr
		}
		secret = core.NewRecord(collection)
	} else if err != nil {
		return err
	}
	raw, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	secret.Set("chatgpt_oauth", string(raw))
	return app.Save(secret)
}

func chatGPTRandomValue() string {
	var data [32]byte
	_, _ = rand.Read(data[:])
	return base64.RawURLEncoding.EncodeToString(data[:])
}

func chatGPTExchangeToken(ctx context.Context, form url.Values) (chatGPTTokenResponse, error) {
	var tokens chatGPTTokenResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTTokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokens, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	body, err := chatGPTReadResponse(req, 1024*1024)
	if err != nil {
		return tokens, err
	}
	if json.Unmarshal(body, &tokens) != nil || tokens.AccessToken == "" || tokens.RefreshToken == "" || tokens.ExpiresIn <= 0 || !strings.EqualFold(tokens.TokenType, "Bearer") {
		return tokens, errors.New("ChatGPT did not return renewable OAuth credentials; reconnect in Settings")
	}
	return tokens, nil
}

func chatGPTReadResponse(req *http.Request, limit int64) ([]byte, error) {
	resp, err := chatGPTClient.Do(req)
	if err != nil {
		// Transport errors can include authorization URLs or token request data.
		return nil, errors.New("ChatGPT could not be reached")
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(body)) > limit {
		return nil, errors.New("ChatGPT returned an unreadable or oversized response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ProviderError{Provider: "chatgpt", StatusCode: resp.StatusCode, Body: "ChatGPT request failed; check the connection, account permissions, and plan usage limits"}
	}
	return body, nil
}

func verifyChatGPTIdentity(ctx context.Context, raw, clientID, nonce string) (*chatGPTIdentity, error) {
	claims := &chatGPTIdentity{}
	_, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, chatGPTJWKSURL, nil)
		if err != nil {
			return nil, err
		}
		body, err := chatGPTReadResponse(req, 1024*1024)
		if err != nil {
			return nil, err
		}
		var jwks struct {
			Keys []struct{ Kty, Kid, N, E string }
		}
		if err := json.Unmarshal(body, &jwks); err != nil {
			return nil, err
		}
		for _, key := range jwks.Keys {
			if key.Kty != "RSA" || key.Kid != kid || kid == "" {
				continue
			}
			n, nErr := base64.RawURLEncoding.DecodeString(key.N)
			e, eErr := base64.RawURLEncoding.DecodeString(key.E)
			if nErr == nil && eErr == nil && len(n) >= 256 && len(e) > 0 && len(e) <= 4 {
				return &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}, nil
			}
		}
		return nil, errors.New("no matching OpenAI signing key")
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithIssuer("https://auth.openai.com"), jwt.WithAudience(clientID), jwt.WithExpirationRequired(), jwt.WithIssuedAt(), jwt.WithLeeway(5*time.Second))
	if err != nil || claims.Subject == "" || claims.IssuedAt == nil || nonce != "" && claims.Nonce != nonce {
		return nil, errors.New("ChatGPT identity could not be verified; start sign-in again")
	}
	return claims, nil
}

func chatGPTAccessToken(ctx context.Context, app core.App, rejectedToken string) (string, error) {
	chatGPTCredentialsMu.Lock()
	defer chatGPTCredentialsMu.Unlock()
	credentials, err := loadChatGPTCredentials(app)
	if err != nil {
		return "", err
	}
	if credentials.AccessToken == "" || !hasChatGPTPlanScope(credentials.Scope) {
		return "", errors.New("connect ChatGPT and allow plan usage in Settings")
	}
	if time.Until(credentials.ExpiresAt) > time.Minute && credentials.AccessToken != rejectedToken {
		return credentials.AccessToken, nil
	}
	tokens, err := chatGPTExchangeToken(ctx, url.Values{"grant_type": {"refresh_token"}, "client_id": {credentials.ClientID}, "refresh_token": {credentials.RefreshToken}, "resource": {chatGPTResource}})
	if err != nil {
		return "", err
	}
	if tokens.Scope == "" {
		tokens.Scope = credentials.Scope
	}
	if !hasChatGPTPlanScope(tokens.Scope) {
		return "", errors.New("ChatGPT plan permission was removed; reconnect in Settings")
	}
	if tokens.IDToken != "" {
		identity, err := verifyChatGPTIdentity(ctx, tokens.IDToken, credentials.ClientID, "")
		if err != nil || identity.Subject != credentials.Subject {
			return "", errors.New("ChatGPT account changed during renewal; reconnect in Settings")
		}
		credentials.IDToken = tokens.IDToken
	}
	credentials.AccessToken, credentials.RefreshToken = tokens.AccessToken, tokens.RefreshToken
	credentials.Scope = tokens.Scope
	credentials.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	if err := saveChatGPTCredentials(app, credentials); err != nil {
		return "", err
	}
	return credentials.AccessToken, nil
}

func hasChatGPTPlanScope(scope string) bool {
	for _, item := range strings.Fields(scope) {
		if item == "chatgpt.tokens.use.direct" {
			return true
		}
	}
	return false
}

type chatGPTLoginAttempt struct {
	userID, state, nonce, verifier, redirectURI string
	credentials                                 chatGPTCredentials
	expiresAt                                   time.Time
	listener                                    net.Listener
	err                                         string
	completed                                   bool
}

type chatGPTOAuthService struct {
	app      core.App
	mu       sync.Mutex
	attempts map[string]*chatGPTLoginAttempt
}

func registerChatGPTOAuthAPI(app *pocketbase.PocketBase) {
	service := &chatGPTOAuthService{app: app, attempts: map[string]*chatGPTLoginAttempt{}}
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		admin := func(e *core.RequestEvent) error {
			e.Response.Header().Set("Cache-Control", "no-store")
			if e.Auth == nil || e.Auth.GetString("role") != "admin" {
				return apis.NewForbiddenError("Admin role required.", nil)
			}
			return e.Next()
		}
		se.Router.POST("/api/ai/chatgpt/login", service.start).BindFunc(admin)
		se.Router.DELETE("/api/ai/chatgpt/login", service.cancel).BindFunc(admin)
		se.Router.GET("/api/ai/chatgpt/status", service.status).BindFunc(admin)
		se.Router.POST("/api/ai/chatgpt/complete", service.complete).BindFunc(admin)
		se.Router.DELETE("/api/ai/chatgpt/connection", service.disconnect).BindFunc(admin)
		return se.Next()
	})
}

func (s *chatGPTOAuthService) cancel(e *core.RequestEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if attempt := s.attempts[e.Auth.Id]; attempt != nil {
		attempt.completed = true
		if attempt.listener != nil {
			_ = attempt.listener.Close()
		}
		delete(s.attempts, e.Auth.Id)
	}
	return e.NoContent(http.StatusNoContent)
}

func (s *chatGPTOAuthService) start(e *core.RequestEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old := s.attempts[e.Auth.Id]; old != nil && old.listener != nil {
		old.completed = true
		_ = old.listener.Close()
	}
	chatGPTCredentialsMu.Lock()
	credentials, err := loadChatGPTCredentials(s.app)
	if err == nil && credentials.HostID == "" {
		credentials.HostID = "urn:uuid:" + uuid.NewString()
		err = saveChatGPTCredentials(s.app, credentials)
	}
	chatGPTCredentialsMu.Unlock()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return apis.NewBadRequestError("Could not start the ChatGPT callback listener.", nil)
	}
	attempt := &chatGPTLoginAttempt{userID: e.Auth.Id, state: chatGPTRandomValue(), nonce: chatGPTRandomValue(), verifier: chatGPTRandomValue(), credentials: credentials, expiresAt: time.Now().Add(10 * time.Minute), listener: listener, redirectURI: "http://" + listener.Addr().String() + "/auth/callback"}
	s.attempts[e.Auth.Id] = attempt
	challenge := sha256.Sum256([]byte(attempt.verifier))
	clientID := credentials.ClientID
	if clientID == "" {
		clientID = "dynamic_agent_client"
	}
	params := url.Values{"client_id": {clientID}, "response_type": {"code"}, "redirect_uri": {attempt.redirectURI}, "scope": {"openid profile email offline_access resource.invoke chatgpt.tokens.use.direct"}, "resource": {chatGPTResource}, "ext_agent_host_id": {credentials.HostID}, "state": {attempt.state}, "nonce": {attempt.nonce}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(challenge[:])}}
	if clientID == "dynamic_agent_client" {
		params.Set("agent_name_hint", "alleycat")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		s.mu.Lock()
		err := s.finish(r.Context(), attempt, r.URL.Query())
		s.mu.Unlock()
		if err != nil {
			http.Error(w, "ChatGPT connection failed. Return to alleycat Settings for details.", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, "ChatGPT connected. You can close this tab and return to alleycat Settings.")
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = server.Serve(listener) }()
	time.AfterFunc(10*time.Minute, func() { _ = listener.Close() })
	return e.JSON(http.StatusOK, map[string]any{"authorization_url": chatGPTAuthorizeURL + "?" + params.Encode(), "expires_at": attempt.expiresAt})
}

// The loopback redirect may run on a remote CMS host. A signed-in administrator
// can submit its full callback address when the browser cannot reach that host.
func (s *chatGPTOAuthService) complete(e *core.RequestEvent) error {
	var body struct {
		CallbackURL string `json:"callback_url"`
	}
	if e.BindBody(&body) != nil {
		return apis.NewBadRequestError("A callback address is required.", nil)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	attempt := s.attempts[e.Auth.Id]
	callback, err := url.Parse(body.CallbackURL)
	if attempt == nil || err != nil || callback.Scheme+"://"+callback.Host+callback.Path != attempt.redirectURI || callback.Fragment != "" || callback.User != nil {
		return apis.NewBadRequestError("The callback address does not match this sign-in attempt.", nil)
	}
	if err := s.finish(e.Request.Context(), attempt, callback.Query()); err != nil {
		return apis.NewBadRequestError(err.Error(), nil)
	}
	return e.JSON(http.StatusOK, map[string]bool{"connected": true})
}

func (s *chatGPTOAuthService) finish(ctx context.Context, attempt *chatGPTLoginAttempt, query url.Values) error {
	if s.attempts[attempt.userID] != attempt || attempt.completed || time.Now().After(attempt.expiresAt) || query.Get("state") != attempt.state {
		return errors.New("ChatGPT sign-in expired or could not be verified; start again")
	}
	attempt.completed = true
	defer func() {
		if attempt.listener != nil {
			_ = attempt.listener.Close()
		}
	}()
	fail := func(message string) error { attempt.err = message; return errors.New(message) }
	user, err := s.app.FindRecordById("cms_users", attempt.userID)
	if err != nil || user.GetString("role") != "admin" {
		return fail("The administrator who started sign-in no longer has access")
	}
	if query.Get("error") != "" || query.Get("code") == "" {
		return fail("ChatGPT sign-in was cancelled or did not return a code")
	}
	clientID := query.Get("client_id")
	if attempt.credentials.ClientID != "" {
		if clientID != "" && clientID != attempt.credentials.ClientID {
			return fail("ChatGPT client identity changed; start sign-in again")
		}
		clientID = attempt.credentials.ClientID
	}
	if clientID == "" || clientID == "dynamic_agent_client" {
		return fail("ChatGPT registration did not return a client ID")
	}
	tokens, err := chatGPTExchangeToken(ctx, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {query.Get("code")}, "code_verifier": {attempt.verifier}, "redirect_uri": {attempt.redirectURI}, "resource": {chatGPTResource}})
	if err != nil {
		return fail("ChatGPT code exchange failed; start sign-in again")
	}
	identity, err := verifyChatGPTIdentity(ctx, tokens.IDToken, clientID, attempt.nonce)
	if err != nil {
		return fail(err.Error())
	}
	if attempt.credentials.Subject != "" && identity.Subject != attempt.credentials.Subject {
		return fail("Sign in with the ChatGPT account already connected to this CMS")
	}
	if !hasChatGPTPlanScope(tokens.Scope) {
		return fail("ChatGPT plan usage was not allowed. Sign in again and allow plan usage.")
	}
	credentials := chatGPTCredentials{HostID: attempt.credentials.HostID, ClientID: clientID, Subject: identity.Subject, Email: identity.Email, IDToken: tokens.IDToken, AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken, Scope: tokens.Scope, ExpiresAt: time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)}
	chatGPTCredentialsMu.Lock()
	err = saveChatGPTCredentials(s.app, credentials)
	chatGPTCredentialsMu.Unlock()
	if err != nil {
		return fail("ChatGPT credentials could not be saved; start sign-in again")
	}
	return nil
}

func (s *chatGPTOAuthService) status(e *core.RequestEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	credentials, err := loadChatGPTCredentials(s.app)
	if err != nil {
		return err
	}
	message := ""
	pending := false
	if attempt := s.attempts[e.Auth.Id]; attempt != nil {
		message = attempt.err
		pending = !attempt.completed && time.Now().Before(attempt.expiresAt)
		if !attempt.completed && !pending {
			message = "ChatGPT sign-in expired; start again"
		}
	}
	return e.JSON(http.StatusOK, map[string]any{"connected": credentials.AccessToken != "" && hasChatGPTPlanScope(credentials.Scope), "email": credentials.Email, "pending": pending, "error": message})
}

func (s *chatGPTOAuthService) disconnect(e *core.RequestEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for userID, attempt := range s.attempts {
		attempt.completed = true
		if attempt.listener != nil {
			_ = attempt.listener.Close()
		}
		delete(s.attempts, userID)
	}
	chatGPTCredentialsMu.Lock()
	defer chatGPTCredentialsMu.Unlock()
	credentials, err := loadChatGPTCredentials(s.app)
	if err != nil {
		return err
	}
	revoked := credentials.RefreshToken == ""
	if !revoked {
		req, _ := http.NewRequestWithContext(e.Request.Context(), http.MethodGet, chatGPTDiscoveryURL, nil)
		if body, err := chatGPTReadResponse(req, 1024*1024); err == nil {
			var discovery struct {
				RevocationEndpoint string `json:"revocation_endpoint"`
			}
			parseErr := json.Unmarshal(body, &discovery)
			endpoint, urlErr := url.Parse(discovery.RevocationEndpoint)
			if parseErr == nil && urlErr == nil && endpoint.Scheme == "https" && endpoint.Host == "auth.openai.com" && endpoint.User == nil && endpoint.Fragment == "" {
				form := url.Values{"token": {credentials.RefreshToken}, "token_type_hint": {"refresh_token"}, "client_id": {credentials.ClientID}}
				req, err := http.NewRequestWithContext(e.Request.Context(), http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
				if err == nil {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					_, err = chatGPTReadResponse(req, 1024*1024)
					revoked = err == nil
				}
			}
		}
	}
	credentials.AccessToken, credentials.RefreshToken, credentials.IDToken, credentials.Scope = "", "", "", ""
	credentials.ExpiresAt = time.Time{}
	if err := saveChatGPTCredentials(s.app, credentials); err != nil {
		return err
	}
	return e.JSON(http.StatusOK, map[string]bool{"connected": false, "revoked": revoked})
}
