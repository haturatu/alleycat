package pbapp

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func newChatGPTTestApp(t *testing.T) *pocketbase.PocketBase {
	t.Helper()
	app := pocketbase.NewWithConfig(pocketbase.Config{DefaultDataDir: t.TempDir()})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.ClearBootstrap() })
	if err := ensureCollections(app); err != nil {
		t.Fatal(err)
	}
	return app
}

func chatGPTTestUser(t *testing.T, app core.App, role string) string {
	t.Helper()
	collection, err := app.FindCollectionByNameOrId("cms_users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(collection)
	user.Set("email", role+"@example.com")
	user.Set("name", role)
	user.Set("role", role)
	user.SetPassword("password-for-test")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	token, err := user.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func chatGPTTestRouter(t *testing.T, app *pocketbase.PocketBase) *httptest.Server {
	t.Helper()
	registerChatGPTOAuthAPI(app)
	registerSlugGenerationAPI(app)
	registerTranslationModelsAPI(app)
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.OnServe().Trigger(&core.ServeEvent{App: app, Router: router}, func(*core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func chatGPTTestRequest(t *testing.T, method, endpoint, token, body string) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, endpoint, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, data
}

func TestChatGPTOAuthLoginAndSlug(t *testing.T) {
	app := newChatGPTTestApp(t)
	admin := chatGPTTestUser(t, app, "admin")
	editor := chatGPTTestUser(t, app, "editor")
	api := chatGPTTestRouter(t, app)
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	nonce, challenge := "", ""
	tokenCalls := 0
	grantedScope := "openid resource.invoke chatgpt.tokens.use.direct"
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/jwks":
			_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": "test", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())}}})
		case "/token":
			tokenCalls++
			_ = r.ParseForm()
			actual := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			if r.Form.Get("client_id") != "oaiapp_test" || base64.RawURLEncoding.EncodeToString(actual[:]) != challenge || r.Form.Get("resource") != chatGPTResource {
				t.Error("invalid OAuth code exchange")
			}
			claims := chatGPTIdentity{RegisteredClaims: jwt.RegisteredClaims{Issuer: "https://auth.openai.com", Subject: "account-one", Audience: jwt.ClaimStrings{"oaiapp_test"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), IssuedAt: jwt.NewNumericDate(time.Now())}, Nonce: nonce, Email: "account@example.com"}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			idToken, _ := token.SignedString(key)
			_ = json.NewEncoder(w).Encode(chatGPTTokenResponse{AccessToken: "test-access-token", RefreshToken: "test-refresh-token", IDToken: idToken, TokenType: "Bearer", Scope: grantedScope, ExpiresIn: 3600})
		default:
			http.NotFound(w, r)
		}
	}))
	defer oauth.Close()
	originalToken, originalJWKS := chatGPTTokenURL, chatGPTJWKSURL
	chatGPTTokenURL, chatGPTJWKSURL = oauth.URL+"/token", oauth.URL+"/jwks"
	t.Cleanup(func() { chatGPTTokenURL, chatGPTJWKSURL = originalToken, originalJWKS })
	for _, token := range []string{"", editor} {
		for _, route := range []struct{ method, path string }{{"POST", "login"}, {"GET", "status"}, {"POST", "complete"}, {"DELETE", "connection"}, {"DELETE", "login"}} {
			status, _ := chatGPTTestRequest(t, route.method, api.URL+"/api/ai/chatgpt/"+route.path, token, "{}")
			if status != 403 {
				t.Fatalf("unauthorized %s returned %d", route.path, status)
			}
		}
	}
	status, data := chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/login", admin, "{}")
	if status != 200 {
		t.Fatalf("login: %d %s", status, data)
	}
	var login struct {
		AuthorizationURL string `json:"authorization_url"`
	}
	if err := json.Unmarshal(data, &login); err != nil {
		t.Fatal(err)
	}
	authorize, _ := url.Parse(login.AuthorizationURL)
	params := authorize.Query()
	nonce, challenge = params.Get("nonce"), params.Get("code_challenge")
	if params.Get("client_id") != "dynamic_agent_client" || params.Get("agent_name_hint") != "alleycat" || !strings.HasPrefix(params.Get("ext_agent_host_id"), "urn:uuid:") || nonce == "" || challenge == "" {
		t.Fatal("missing dynamic registration or PKCE parameters")
	}
	callback, _ := url.Parse(params.Get("redirect_uri"))
	callback.RawQuery = url.Values{"state": {"wrong-state"}, "code": {"test-code"}, "client_id": {"oaiapp_test"}}.Encode()
	body, _ := json.Marshal(map[string]string{"callback_url": callback.String()})
	status, _ = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/complete", admin, string(body))
	if status != 400 || tokenCalls != 0 {
		t.Fatal("invalid state reached token exchange")
	}
	callback.RawQuery = url.Values{"state": {params.Get("state")}, "code": {"test-code"}, "client_id": {"oaiapp_test"}}.Encode()
	body, _ = json.Marshal(map[string]string{"callback_url": callback.String()})
	status, data = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/complete", admin, string(body))
	if status != 200 {
		t.Fatalf("complete: %d %s", status, data)
	}
	status, _ = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/complete", admin, string(body))
	if status != 400 || tokenCalls != 1 {
		t.Fatal("authorization code was replayed")
	}
	status, data = chatGPTTestRequest(t, "GET", api.URL+"/api/ai/chatgpt/status", admin, "")
	if status != 200 || !strings.Contains(string(data), `"connected":true`) || strings.Contains(string(data), "test-access-token") || strings.Contains(string(data), "refresh_token") {
		t.Fatalf("unsafe or invalid status: %s", data)
	}
	_, data = chatGPTTestRequest(t, "GET", api.URL+"/api/collections/app_secrets/records", admin, "")
	if strings.Contains(string(data), "chatgpt_oauth") || strings.Contains(string(data), "test-access-token") {
		t.Fatal("OAuth credentials exposed through record API")
	}
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	settings.Set("translation_provider", "chatgpt")
	settings.Set("translation_model", "test-model")
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	inference := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-access-token" {
			t.Error("missing OAuth bearer token")
		}
		var payload map[string]any
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if payload["stream"] != true || payload["store"] != false || payload["model"] != "test-model" || payload["input"] == nil || payload["temperature"] != nil || payload["max_output_tokens"] != nil {
			t.Error("invalid ChatGPT plan request")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"{\\\"slug\\\":\\\"hello-world\\\"}\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n")
	}))
	defer inference.Close()
	originalResponses := chatGPTResponsesURL
	chatGPTResponsesURL = inference.URL
	t.Cleanup(func() { chatGPTResponsesURL = originalResponses })
	status, data = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/slug", editor, `{"title":"こんにちは"}`)
	if status != 200 || !strings.Contains(string(data), `"slug":"hello-world"`) {
		t.Fatalf("slug generation: %d %s", status, data)
	}

	// Reauthorization keeps the issued registration and host identity. Declined
	// plan permission must not replace the current working connection.
	status, data = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/login", admin, "{}")
	if status != 200 {
		t.Fatalf("reauthorize: %d", status)
	}
	if err := json.Unmarshal(data, &login); err != nil {
		t.Fatal(err)
	}
	reauthorize, _ := url.Parse(login.AuthorizationURL)
	newParams := reauthorize.Query()
	if newParams.Get("client_id") != "oaiapp_test" || newParams.Get("agent_name_hint") != "" || newParams.Get("ext_agent_host_id") != params.Get("ext_agent_host_id") {
		t.Fatal("registration was not retained")
	}
	nonce, challenge = newParams.Get("nonce"), newParams.Get("code_challenge")
	grantedScope = "openid"
	callback, _ = url.Parse(newParams.Get("redirect_uri"))
	callback.RawQuery = url.Values{"state": {newParams.Get("state")}, "code": {"next-code"}}.Encode()
	body, _ = json.Marshal(map[string]string{"callback_url": callback.String()})
	status, _ = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/complete", admin, string(body))
	stored, _ := loadChatGPTCredentials(app)
	if status != 400 || stored.AccessToken != "test-access-token" || !hasChatGPTPlanScope(stored.Scope) {
		t.Fatal("declined plan permission replaced working credentials")
	}
	status, data = chatGPTTestRequest(t, "POST", api.URL+"/api/ai/chatgpt/login", admin, "{}")
	if status != 200 {
		t.Fatalf("loopback sign-in: %d", status)
	}
	if err := json.Unmarshal(data, &login); err != nil {
		t.Fatal(err)
	}
	reauthorize, _ = url.Parse(login.AuthorizationURL)
	newParams = reauthorize.Query()
	nonce, challenge = newParams.Get("nonce"), newParams.Get("code_challenge")
	grantedScope = "openid resource.invoke chatgpt.tokens.use.direct"
	callback, _ = url.Parse(newParams.Get("redirect_uri"))
	callback.RawQuery = url.Values{"state": {newParams.Get("state")}, "code": {"loopback-code"}}.Encode()
	status, data = chatGPTTestRequest(t, "GET", callback.String(), "", "")
	if status != 200 || !strings.Contains(string(data), "ChatGPT connected") {
		t.Fatalf("loopback callback: %d %s", status, data)
	}

	originalDiscovery := chatGPTDiscoveryURL
	chatGPTDiscoveryURL = oauth.URL + "/discovery"
	t.Cleanup(func() { chatGPTDiscoveryURL = originalDiscovery })
	status, data = chatGPTTestRequest(t, "DELETE", api.URL+"/api/ai/chatgpt/connection", admin, "")
	stored, _ = loadChatGPTCredentials(app)
	if status != 200 || !strings.Contains(string(data), `"revoked":false`) || stored.AccessToken != "" || stored.RefreshToken != "" || stored.IDToken != "" || stored.ClientID != "oaiapp_test" || stored.HostID == "" {
		t.Fatalf("disconnect: %d %s", status, data)
	}
}

func TestChatGPTProviderMigratesExistingSettings(t *testing.T) {
	app := newChatGPTTestApp(t)
	collection, err := app.FindCollectionByNameOrId("settings")
	if err != nil {
		t.Fatal(err)
	}
	collection.Fields.GetByName("translation_provider").(*core.SelectField).Values = []string{"gemini", "opencode-go", "opencode-zen"}
	if err := app.Save(collection); err != nil {
		t.Fatal(err)
	}
	if err := ensureCollections(app); err != nil {
		t.Fatal(err)
	}
	settings, _ := app.FindFirstRecordByFilter("settings", "id != ''")
	settings.Set("translation_provider", "chatgpt")
	if err := app.Save(settings); err != nil {
		t.Fatalf("existing settings cannot select ChatGPT: %v", err)
	}
}

func TestChatGPTModelCatalogUsesOAuthAndServerOrdering(t *testing.T) {
	app := newChatGPTTestApp(t)
	if err := saveChatGPTCredentials(app, chatGPTCredentials{AccessToken: "catalog-token", Scope: "chatgpt.tokens.use.direct", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer catalog-token" {
			t.Error("model catalog did not use the OAuth token")
		}
		_, _ = io.WriteString(w, `{"models":[{"slug":"preferred","visibility":"list"},{"slug":"hidden","visibility":"hide"},{"slug":"another","visibility":"list"},{"slug":"preferred","visibility":"list"}]}`)
	}))
	defer server.Close()
	original := chatGPTModelsURL
	chatGPTModelsURL = server.URL
	t.Cleanup(func() { chatGPTModelsURL = original })
	models, err := fetchChatGPTModels(app)
	if err != nil || strings.Join(models, ",") != "preferred,another" {
		t.Fatalf("models=%v, err=%v", models, err)
	}
}

func TestChatGPTRefreshIsSerializedAndRotated(t *testing.T) {
	app := newChatGPTTestApp(t)
	credentials := chatGPTCredentials{ClientID: "oaiapp_test", Subject: "account", AccessToken: "expired", RefreshToken: "original", Scope: "chatgpt.tokens.use.direct", ExpiresAt: time.Now().Add(-time.Hour)}
	if err := saveChatGPTCredentials(app, credentials); err != nil {
		t.Fatal(err)
	}
	var refreshes atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		refreshes.Add(1)
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "original" || r.Form.Get("client_id") != "oaiapp_test" || r.Form.Get("scope") != "" {
			t.Error("invalid refresh request")
		}
		_ = json.NewEncoder(w).Encode(chatGPTTokenResponse{AccessToken: "renewed", RefreshToken: "rotated", TokenType: "Bearer", ExpiresIn: 3600})
	}))
	defer server.Close()
	original := chatGPTTokenURL
	chatGPTTokenURL = server.URL
	t.Cleanup(func() { chatGPTTokenURL = original })
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			token, err := chatGPTAccessToken(context.Background(), app, "")
			if err != nil || token != "renewed" {
				t.Errorf("refresh: %q %v", token, err)
			}
		})
	}
	group.Wait()
	stored, _ := loadChatGPTCredentials(app)
	if refreshes.Load() != 1 || stored.RefreshToken != "rotated" {
		t.Fatalf("refreshes=%d, token rotation persisted=%v", refreshes.Load(), stored.RefreshToken == "rotated")
	}
}

func TestChatGPTRejectsInvalidIdentity(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{"kty": "RSA", "kid": "test", "n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": "AQAB"}}})
	}))
	defer server.Close()
	original := chatGPTJWKSURL
	chatGPTJWKSURL = server.URL
	t.Cleanup(func() { chatGPTJWKSURL = original })
	for _, tc := range []struct {
		name, issuer, audience, nonce string
		expiry                        time.Time
		unsigned                      bool
	}{
		{"valid", "https://auth.openai.com", "oaiapp_test", "nonce", time.Now().Add(time.Hour), false},
		{"issuer", "https://evil.example", "oaiapp_test", "nonce", time.Now().Add(time.Hour), false},
		{"audience", "https://auth.openai.com", "other-client", "nonce", time.Now().Add(time.Hour), false},
		{"nonce", "https://auth.openai.com", "oaiapp_test", "wrong", time.Now().Add(time.Hour), false},
		{"expired", "https://auth.openai.com", "oaiapp_test", "nonce", time.Now().Add(-time.Hour), false},
		{"unsigned", "https://auth.openai.com", "oaiapp_test", "nonce", time.Now().Add(time.Hour), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := chatGPTIdentity{RegisteredClaims: jwt.RegisteredClaims{Issuer: tc.issuer, Subject: "account", Audience: jwt.ClaimStrings{tc.audience}, ExpiresAt: jwt.NewNumericDate(tc.expiry), IssuedAt: jwt.NewNumericDate(time.Now())}, Nonce: tc.nonce}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test"
			var raw string
			if tc.unsigned {
				token.Method = jwt.SigningMethodNone
				raw, _ = token.SignedString(jwt.UnsafeAllowNoneSignatureType)
			} else {
				raw, _ = token.SignedString(key)
			}
			_, err := verifyChatGPTIdentity(context.Background(), raw, "oaiapp_test", "nonce")
			if (err == nil) != (tc.name == "valid") {
				t.Fatalf("identity validation: %v", err)
			}
		})
	}
}

func TestParseChatGPTResponseStream(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, stream, want string }{
		{"completed", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"{\\\"slug\\\":\\\"hello\\\"}\"}\n\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\n", `{"slug":"hello"}`},
		{"truncated", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"{\\\"slug\\\":\\\"hello\\\"}\"}\n\n", ""},
		{"failed", "data: {\"type\":\"response.failed\"}\n\n", ""},
		{"incomplete", "data: {\"type\":\"response.incomplete\"}\n\n", ""},
		{"done without completion", "data: [DONE]\n\n", ""},
		{"invalid", "data: bad-json\n\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseChatGPTResponseStream([]byte(tc.stream))
			if tc.want == "" && err == nil || tc.want != "" && (err != nil || got != tc.want) {
				t.Fatalf("stream result: %q, %v", got, err)
			}
		})
	}
}
