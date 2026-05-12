package types

type Provider string

const ProviderGmail Provider = "gmail"

type EmailProvider struct {
	Address  string
	Alias    string
	Provider Provider
}
