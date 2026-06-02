// gettoken runs the Gmail OAuth2 flow and writes token.json.
//
// Usage:
//
//	go run ./cmd/gettoken -credentials credentials.json -out token.json
//
// The credentials.json must be an OAuth 2.0 Client ID of type "Desktop app"
// (formerly "Installed application") downloaded from Google Cloud Console.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

// credentialsFile mirrors the shape of credentials.json from Google Cloud Console.
type credentialsFile struct {
	Web       *credEntry `json:"web"`
	Installed *credEntry `json:"installed"`
}

type credEntry struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func main() {
	credFile := flag.String("credentials", "credentials.json", "path to credentials.json (Desktop app type)")
	outFile := flag.String("out", "token.json", "where to write token.json")
	flag.Parse()

	data, err := os.ReadFile(*credFile)
	if err != nil {
		log.Fatalf("reading %s: %v\n\nDownload from: console.cloud.google.com → APIs & Services → Credentials", *credFile, err)
	}

	var creds credentialsFile
	if err := json.Unmarshal(data, &creds); err != nil {
		log.Fatalf("parsing credentials.json: %v", err)
	}

	entry := creds.Installed
	if entry == nil {
		entry = creds.Web
	}
	if entry == nil || entry.ClientID == "" {
		log.Fatal("credentials.json must contain a 'web' or 'installed' key with client_id/client_secret.\n" +
			"Make sure you downloaded an OAuth 2.0 Client ID (not a Service Account key).")
	}

	// Bind to a random free port — Desktop app credentials allow any localhost port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("starting local server: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://localhost:%d/callback", port)

	cfg := &oauth2.Config{
		ClientID:     entry.ClientID,
		ClientSecret: entry.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailSendScope},
		RedirectURL:  redirectURL,
	}

	authURL := cfg.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("\n─────────────────────────────────────────────────────────────────────")
	fmt.Println("  Opening browser for Google authorization…")
	fmt.Println("  If it doesn't open, visit this URL manually:")
	fmt.Println()
	fmt.Println(" ", authURL)
	fmt.Println("─────────────────────────────────────────────────────────────────────")
	openBrowser(authURL)
	fmt.Println("\n  Waiting for authorization (listening on", redirectURL, ")…")

	// Catch the OAuth2 callback on localhost.
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			msg := r.URL.Query().Get("error")
			errCh <- fmt.Errorf("authorization denied: %s", msg)
			fmt.Fprintf(w, "<h2>Authorization failed: %s. You can close this tab.</h2>", msg)
			return
		}
		fmt.Fprintln(w, "<h2>Authorization successful! You can close this tab.</h2>")
		codeCh <- code
	})

	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()

	var code string
	select {
	case code = <-codeCh:
	case authErr := <-errCh:
		log.Fatal(authErr)
	}
	_ = srv.Close()

	fmt.Println("  Authorization code received. Exchanging for tokens…")

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		log.Fatalf("exchanging code: %v", err)
	}
	if tok.RefreshToken == "" {
		log.Fatal("no refresh_token in response — revoke app access at myaccount.google.com/permissions and try again")
	}

	f, err := os.OpenFile(*outFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("creating %s: %v", *outFile, err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(tok); err != nil {
		log.Fatalf("writing token: %v", err)
	}

	fmt.Printf("\n✓  token.json written to: %s\n", *outFile)
	fmt.Println("   Add to your .env:")
	fmt.Printf("   GMAIL_CREDENTIALS_FILE=%s\n", *credFile)
	fmt.Printf("   GMAIL_TOKEN_FILE=%s\n", *outFile)
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "linux":
		cmd, args = "xdg-open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return
	}
	_ = exec.Command(cmd, args...).Start()
}
