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

// NewConnector returns a Connector for the given provider. For Gmail, it reads
// OAuth2 credentials from DataDir()/credentials.json.
func NewConnector(p types.EmailProvider) (connector.Connector[any], error) {
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
