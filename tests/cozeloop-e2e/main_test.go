package e2e

import (
	"context"
	"os"
	"testing"

	cozeloop "github.com/coze-dev/cozeloop-go"
)

// Run with:
// COZELOOP_API_BASE_URL=http://localhost:8080 \
// COZELOOP_WORKSPACE_ID=default COZELOOP_API_TOKEN=... go test ./...
func TestOfficialGoSDKIngest(t *testing.T) {
	base, token, workspace := os.Getenv("COZELOOP_API_BASE_URL"), os.Getenv("COZELOOP_API_TOKEN"), os.Getenv("COZELOOP_WORKSPACE_ID")
	if base == "" || token == "" || workspace == "" {
		t.Skip("set COZELOOP_API_BASE_URL, COZELOOP_API_TOKEN and COZELOOP_WORKSPACE_ID")
	}
	client, err := cozeloop.NewClient(cozeloop.WithAPIBaseURL(base), cozeloop.WithAPIToken(token), cozeloop.WithWorkspaceID(workspace))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := client.StartSpan(context.Background(), "tracy-official-sdk", "custom")
	span.SetInput(ctx, map[string]string{"hello": "tracy"})
	span.SetOutput(ctx, map[string]string{"ok": "true"})
	span.SetServiceName(ctx, "tracy-e2e")
	span.Finish(ctx)
	client.Close(ctx)
}

// Run with an OAuth App whose public key is registered in Tracy:
// COZELOOP_API_BASE_URL=http://localhost:8080 \
// COZELOOP_WORKSPACE_ID=default \
// COZELOOP_JWT_OAUTH_CLIENT_ID=... COZELOOP_JWT_OAUTH_PRIVATE_KEY=... \
// COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID=... go test ./...
func TestOfficialGoSDKJWTOAuth(t *testing.T) {
	base, workspace := os.Getenv("COZELOOP_API_BASE_URL"), os.Getenv("COZELOOP_WORKSPACE_ID")
	if base == "" || workspace == "" || os.Getenv("COZELOOP_JWT_OAUTH_CLIENT_ID") == "" || os.Getenv("COZELOOP_JWT_OAUTH_PRIVATE_KEY") == "" || os.Getenv("COZELOOP_JWT_OAUTH_PUBLIC_KEY_ID") == "" {
		t.Skip("set CozeLoop API base, workspace and JWT OAuth environment variables")
	}
	client, err := cozeloop.NewClient(cozeloop.WithAPIBaseURL(base), cozeloop.WithWorkspaceID(workspace))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := client.StartSpan(context.Background(), "tracy-jwt-oauth-e2e", "custom")
	span.SetInput(ctx, map[string]string{"auth": "jwt"})
	span.Finish(ctx)
	client.Close(ctx)
}
