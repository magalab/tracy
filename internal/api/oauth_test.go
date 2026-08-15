package api

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/panda/tracy/internal/ingest"
	"github.com/panda/tracy/internal/storage/meta"
	sqlitestore "github.com/panda/tracy/internal/storage/sqlite"
	tracestore "github.com/panda/tracy/internal/storage/trace/sqlite"
)

func TestJWTOAuthTokenExchange(t *testing.T) {
	ctx := context.Background()
	metaDB, err := sqlitestore.Open(ctx, t.TempDir()+"/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	defer metaDB.Close()
	metaStore := meta.NewStore(metaDB)
	if err := metaStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := metaStore.CreateWorkspace(ctx, meta.Workspace{ID: "default", Name: "Default", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	if err := metaStore.CreateOAuthApp(ctx, meta.OAuthApp{ID: "app", ClientID: "client-1", WorkspaceID: "default", PublicKeyID: "key-1", PublicKey: string(publicPEM), Enabled: true, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	traceDB, err := sqlitestore.Open(ctx, t.TempDir()+"/traces.db")
	if err != nil {
		t.Fatal(err)
	}
	defer traceDB.Close()
	traceStore := tracestore.NewStore(traceDB)
	if err := traceStore.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	writer := ingest.NewWriter(traceStore, 1, time.Hour, 1)
	defer writer.Close(ctx)
	server := NewServer(metaStore, writer, traceStore, slog.Default()).Routes()

	claims := map[string]any{"iss": "client-1", "aud": "127.0.0.1:8080", "iat": now.Unix(), "exp": now.Add(time.Minute).Unix(), "jti": "jti-1"}
	jwt := signTestJWT(t, privateKey, map[string]any{"alg": "RS256", "kid": "key-1", "typ": "JWT"}, claims)
	body, _ := json.Marshal(map[string]any{"client_id": "client-1", "grant_type": jwtBearerGrantType, "duration_seconds": 300})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/permission/oauth2/token", bytes.NewReader(body))
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Authorization", "Bearer "+jwt)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tokenResponse); err != nil {
		t.Fatal(err)
	}
	if tokenResponse.AccessToken == "" || tokenResponse.TokenType != "Bearer" || tokenResponse.ExpiresIn != 300 {
		t.Fatalf("response=%+v", tokenResponse)
	}
	key, err := metaStore.Authenticate(ctx, tokenResponse.AccessToken)
	if err != nil || key.WorkspaceID != "default" {
		t.Fatalf("issued token auth key=%+v err=%v", key, err)
	}

	replayReq := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/permission/oauth2/token", bytes.NewReader(body))
	replayReq.Host = "127.0.0.1:8080"
	replayReq.Header.Set("Authorization", "Bearer "+jwt)
	replayRes := httptest.NewRecorder()
	server.ServeHTTP(replayRes, replayReq)
	if replayRes.Code != http.StatusUnauthorized {
		t.Fatalf("replayed JWT status=%d body=%s", replayRes.Code, replayRes.Body.String())
	}

	claims["aud"] = "wrong-host"
	badJWT := signTestJWT(t, privateKey, map[string]any{"alg": "RS256", "kid": "key-1", "typ": "JWT"}, claims)
	req = httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/permission/oauth2/token", bytes.NewReader(body))
	req.Host = "127.0.0.1:8080"
	req.Header.Set("Authorization", "Bearer "+badJWT)
	res = httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("bad audience status=%d body=%s", res.Code, res.Body.String())
	}
}

func signTestJWT(t *testing.T, privateKey *rsa.PrivateKey, header, claims map[string]any) string {
	t.Helper()
	encode := func(value any) string {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return base64.RawURLEncoding.EncodeToString(data)
	}
	message := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return message + "." + base64.RawURLEncoding.EncodeToString(signature)
}
