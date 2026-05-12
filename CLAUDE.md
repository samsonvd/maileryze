# maileryze

Go CLI tool for email analysis and decluttering. Data stays local — no external services.

## Commands

```bash
go run ./main.go {cli args}   # Run the CLI
go build ./...                # Build
go test ./...                 # Run tests
go vet ./...                  # Lint
```

## CLI Structure

Built with [Cobra](https://github.com/spf13/cobra) + [Viper](https://github.com/spf13/viper).

```
maileryze
└── config              # Show current config (cmd/config.go)
    └── create          # Generate default config file (cmd/create.go)
```

Entry point: `main.go` → `cmd.Execute()` (`cmd/root.go`)

## Configuration

TOML format. Viper searches in order:
1. `--config` flag (explicit path)
2. `~/.config/maileryze.toml`
3. `./maileryze.toml`

Config struct defined in `cmd/types.go`.

## Gotchas

- `EmailProvider` fields in `cmd/types.go` are unexported (`address`, `alias`, `provider`). `mapstructure` cannot decode into unexported fields, so `loadConfig()` silently returns empty providers. Fields must be exported and tagged (`mapstructure:"..."`) to work.
