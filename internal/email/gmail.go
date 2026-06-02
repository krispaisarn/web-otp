package email

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// SendOTP sends a one-time password to the given address via Gmail API.
func SendOTP(ctx context.Context, to, code string) error {
	httpClient, err := oauthClient(ctx)
	if err != nil {
		return err
	}
	svc, err := gmail.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("creating gmail service: %w", err)
	}

	from := envVal("GMAIL_FROM_EMAIL")
	body := fmt.Sprintf(
		"Your one-time password is: %s\r\n\r\nThis code expires in 1 hour. Do not share it with anyone.",
		code,
	)
	raw := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: Your One-Time Password\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, body,
	)

	_, err = svc.Users.Messages.Send("me", &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(raw)),
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("sending email: %w", err)
	}

	return nil
}

// oauthClient builds an HTTP client with Gmail credentials.
//
// Priority order:
//  1. GMAIL_CREDENTIALS  (path to credentials.json or its raw JSON content)
//     + GMAIL_TOKEN      (path to token.json or its raw JSON content)
//  2. Individual env vars: GMAIL_CLIENT_ID, GMAIL_CLIENT_SECRET, GMAIL_REFRESH_TOKEN
//     (each accepts a path or raw value)
func oauthClient(ctx context.Context) (*http.Client, error) {
	if cfg := credentialsFromJSON(); cfg != nil {
		tok := tokenFromJSON()
		if tok.RefreshToken == "" {
			return nil, fmt.Errorf(
				"Gmail token has no refresh_token — run `go run ./cmd/gettoken` and set GMAIL_TOKEN_FILE in .env",
			)
		}
		return cfg.Client(ctx, tok), nil
	}

	refreshToken := envVal("GMAIL_REFRESH_TOKEN")
	if refreshToken == "" {
		return nil, fmt.Errorf(
			"Gmail refresh token not set — set GMAIL_REFRESH_TOKEN (or GMAIL_TOKEN_FILE) in .env",
		)
	}

	cfg := &oauth2.Config{
		ClientID:     envVal("GMAIL_CLIENT_ID"),
		ClientSecret: envVal("GMAIL_CLIENT_SECRET"),
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailSendScope},
	}
	return cfg.Client(ctx, &oauth2.Token{
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}), nil
}

// credentialsFromJSON parses GMAIL_CREDENTIALS_FILE or GMAIL_CREDENTIALS as a
// Google credentials.json (supports both "web" and "installed" app types).
func credentialsFromJSON() *oauth2.Config {
	data := envVal("GMAIL_CREDENTIALS")
	if data == "" {
		return nil
	}
	cfg, err := google.ConfigFromJSON([]byte(data), gmail.GmailSendScope)
	if err != nil {
		return nil
	}
	return cfg
}

// tokenFromJSON parses GMAIL_TOKEN_FILE or GMAIL_TOKEN as a Google token.json.
// Falls back to GMAIL_REFRESH_TOKEN if no token JSON is provided.
func tokenFromJSON() *oauth2.Token {
	if data := envVal("GMAIL_TOKEN"); data != "" {
		var tok oauth2.Token
		if err := json.Unmarshal([]byte(data), &tok); err == nil && tok.RefreshToken != "" {
			return &tok
		}
	}
	return &oauth2.Token{
		RefreshToken: envVal("GMAIL_REFRESH_TOKEN"),
		TokenType:    "Bearer",
	}
}

// envVal returns the env var value as-is, or reads it as a file path if it points to an existing file.
func envVal(key string) string {
	val := os.Getenv(key)
	if val == "" {
		return ""
	}
	if data, err := os.ReadFile(val); err == nil {
		return strings.TrimSpace(string(data))
	}
	return strings.TrimSpace(val)
}
