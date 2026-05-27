package factory

import (
	"fmt"
	"os"
	"path/filepath"

	"maileryze/internal/cfg"
	"maileryze/internal/connector"
	"maileryze/internal/connector/gmail"
	"maileryze/internal/types"
)

// Connect looks up the provider by alias, creates the connector, and logs in.
func Connect(alias string) (connector.WritableConnector, *types.EmailProvider, error) {
	provider, err := cfg.ProviderByAlias(alias)
	if err != nil {
		return nil, nil, err
	}
	conn, err := NewConnector(*provider)
	if err != nil {
		return nil, nil, err
	}
	if err := conn.Login(); err != nil {
		return nil, nil, err
	}
	return conn, provider, nil
}

// NewConnector returns a WritableConnector for the given provider.
func NewConnector(p types.EmailProvider) (connector.WritableConnector, error) {
	switch p.Provider {
	case types.ProviderGmail:
		credsPath := filepath.Join(cfg.DataDir(), "credentials.json")
		creds, err := os.ReadFile(credsPath)
		if err != nil {
			return nil, fmt.Errorf("reading Gmail credentials from %s: %w\nDownload credentials.json from Google Cloud Console and place it there.", credsPath, err)
		}
		return gmail.New(p.Alias, p.Address, creds)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", p.Provider)
	}
}
