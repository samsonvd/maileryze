package gmail

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/mail"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googleGmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"

	"maileryze/internal/cfg"
	"maileryze/internal/connector"
)

// MessageData holds Gmail-specific metadata stored in ProviderDetails.Data.
type MessageData struct {
	ID       string
	ThreadID string
	Labels   []string
}

type GmailConnector struct {
	alias     string
	address   string
	oauthConf *oauth2.Config
	service   *googleGmail.Service
}

// New creates a GmailConnector from an OAuth2 credentials JSON file (downloaded
// from Google Cloud Console). The alias is used to namespace the stored token.
func New(alias, address string, credentials []byte) (*GmailConnector, error) {
	conf, err := google.ConfigFromJSON(credentials, googleGmail.GmailReadonlyScope)
	if err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return &GmailConnector{
		alias:     alias,
		address:   address,
		oauthConf: conf,
	}, nil
}

// Login completes the OAuth2 flow if no valid token is cached, then
// initialises the Gmail API service. Must be called before Fetch.
func (g *GmailConnector) Login() error {
	token, err := g.loadToken()
	if err != nil || !token.Valid() {
		token, err = g.authorize()
		if err != nil {
			return fmt.Errorf("authorizing: %w", err)
		}
		if err := g.saveToken(token); err != nil {
			return fmt.Errorf("saving token: %w", err)
		}
	}

	client := g.oauthConf.Client(context.Background(), token)
	svc, err := googleGmail.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("creating gmail service: %w", err)
	}
	g.service = svc
	return nil
}

// Fetch retrieves all messages in the given time window and returns their
// metadata. Only subject, sender, and unsubscribe headers are fetched.
func (g *GmailConnector) Fetch(start, end time.Time) ([]connector.EmailContent[any], error) {
	if g.service == nil {
		return nil, fmt.Errorf("not logged in: call Login() first")
	}

	query := fmt.Sprintf("after:%s before:%s",
		start.Format("2006/01/02"),
		end.Format("2006/01/02"))

	var results []connector.EmailContent[any]
	err := g.service.Users.Messages.List("me").Q(query).Pages(
		context.Background(),
		func(page *googleGmail.ListMessagesResponse) error {
			for _, m := range page.Messages {
				msg, err := g.service.Users.Messages.Get("me", m.Id).
					Format("metadata").
					MetadataHeaders("Subject", "From", "List-Unsubscribe").
					Do()
				if err != nil {
					return fmt.Errorf("fetching message %s: %w", m.Id, err)
				}
				results = append(results, parseMessage(msg))
			}
			return nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	return results, nil
}

func parseMessage(msg *googleGmail.Message) connector.EmailContent[any] {
	headers := make(map[string]string, len(msg.Payload.Headers))
	for _, h := range msg.Payload.Headers {
		headers[h.Name] = h.Value
	}
	return connector.EmailContent[any]{
		Subject:     headers["Subject"],
		Sender:      parseSender(headers["From"]),
		Unsubscribe: parseUnsubscribe(headers["List-Unsubscribe"]),
		Provider: connector.ProviderDetails[any]{
			Identifier: msg.Id,
			Data: MessageData{
				ID:       msg.Id,
				ThreadID: msg.ThreadId,
				Labels:   msg.LabelIds,
			},
		},
	}
}

func parseSender(from string) connector.EmailSender {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		return connector.EmailSender{Address: from}
	}
	return connector.EmailSender{Name: addr.Name, Address: addr.Address}
}

// parseUnsubscribe handles the List-Unsubscribe header format:
// "<mailto:u@example.com>, <https://example.com/unsub>"
func parseUnsubscribe(header string) connector.UnsubscribeMechanism {
	var u connector.UnsubscribeMechanism
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "<") || !strings.HasSuffix(part, ">") {
			continue
		}
		inner := part[1 : len(part)-1]
		switch {
		case strings.HasPrefix(inner, "mailto:"):
			u.Email = strings.TrimPrefix(inner, "mailto:")
		case strings.HasPrefix(inner, "http"):
			u.URL = inner
		}
	}
	return u
}

// OAuth2 token persistence

func (g *GmailConnector) tokenPath() string {
	return filepath.Join(cfg.DataDir(), "tokens", g.alias+".json")
}

func (g *GmailConnector) loadToken() (*oauth2.Token, error) {
	f, err := os.Open(g.tokenPath())
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var t oauth2.Token
	return &t, json.NewDecoder(f).Decode(&t)
}

func (g *GmailConnector) saveToken(t *oauth2.Token) error {
	p := g.tokenPath()
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(t)
}

// authorize runs the OAuth2 loopback redirect flow: starts a local HTTP server
// on a random port, opens the browser, and waits for Google to redirect back
// with the authorization code.
func (g *GmailConnector) authorize() (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, fmt.Errorf("starting callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	g.oauthConf.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", port)
	authURL := g.oauthConf.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

	codeCh := make(chan string, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code != "" {
			fmt.Fprint(w, "<html><body><p>Authorization successful — you can close this tab.</p></body></html>")
		} else {
			fmt.Fprintf(w, "<html><body><p>Authorization failed: %s</p></body></html>", r.URL.Query().Get("error"))
		}
		codeCh <- code
	})
	go srv.Serve(listener)                   //nolint:errcheck
	defer srv.Shutdown(context.Background()) //nolint:errcheck

	fmt.Println("Opening browser for Gmail authorization...")
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Could not open browser automatically. Visit this URL:\n\n  %s\n\n", authURL)
	}

	select {
	case code := <-codeCh:
		if code == "" {
			return nil, fmt.Errorf("authorization denied or failed")
		}
		return g.oauthConf.Exchange(context.Background(), code)
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authorization timed out")
	}
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
