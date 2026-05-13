# maileryze

Take back control of your email by seeing who is creating the clutter, and then deleting it.

All data stays on your device — no external services, no syncing.

## Development setup

### Prerequisites

- [mise](https://mise.jdx.dev) — manages Go and other tool versions

### Install tools and configure git hooks

```sh
mise install
mise run setup
```

### Build

```sh
mise run build
```

### Test and lint

```sh
mise run test
mise run lint
```

## Configuration

Generate a default config file:

```sh
maileryze config create
```

This creates `~/.config/maileryze/maileryze.toml`. Edit it to add your email providers:

```toml
[[providers]]
alias    = 'personal'   # how you refer to this account
provider = 'gmail'
address  = 'you@gmail.com'
```

Multiple providers are supported:

```toml
[[providers]]
alias    = 'personal'
provider = 'gmail'
address  = 'you@gmail.com'

[[providers]]
alias    = 'work'
provider = 'gmail'
address  = 'you@work.com'
```

### Gmail setup

1. Go to [Google Cloud Console](https://console.cloud.google.com) and create a project
2. Enable the **Gmail API**
3. Create **OAuth 2.0 credentials** — type: **Desktop app**
4. Download `credentials.json` and place it at `~/.config/maileryze/credentials.json`

## Usage

All commands accept `-a` / `--alias` to select which provider to act on (where possible).

### Login

Authenticate with an email provider. Opens your browser to complete OAuth.
The token is cached at `~/.config/maileryze/tokens/<alias>.json` for subsequent runs.

```sh
maileryze login -a personal
```

### Load

Fetch emails for a date range and store them locally. `--start` is required; `--end` defaults to today.

```sh
# Load emails from a specific date
maileryze load -a personal -s 2026-01-01

# Load emails between two dates
maileryze load -a personal -s 2026-01-01 -e 2026-03-01
```

Re-running over the same date range is safe — duplicates are skipped.

### Inspect

Show stats about locally stored data.

```sh
maileryze load inspect
```

```
[personal] (gmail)
  Records:      1247
  Oldest email: 2024-01-03
  Newest email: 2026-05-12
  Last fetched: 2026-05-13 14:32:01

[work] (gmail)
  Records:      3891
  Oldest email: 2023-11-15
  Newest email: 2026-05-13
  Last fetched: 2026-05-13 14:35:44
```

### Config

Show the current config and the file it was loaded from:

```sh
maileryze config
```

## Data storage

All app data is stored under `~/.config/maileryze/`:

| Path | Contents |
|------|----------|
| `maileryze.toml` | Configuration |
| `maileryze.db` | Local email database (SQLite) |
| `credentials.json` | OAuth2 client credentials |
| `tokens/<alias>.json` | Cached OAuth2 tokens |
