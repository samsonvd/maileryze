# maileryze

Sort out your email clutter by seeing who is creating it and then deleting it.

All data is fetched live and kept in memory - no persistence, no caching, no external services.

_This is a TUI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)._

## Usage

### Configuration

Create a config file `~/.config/maileryze/maileryze.toml`. Edit it to add your email providers:

```toml
[[providers]]
alias    = 'personal'   # how you refer to this account
provider = 'gmail'
address  = 'you@gmail.com'
```

Multiple providers are supported (untested):

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

### Run it

Install the binary and run it. You can get the latest release from the releases page and download the binary directly, or you can build it yourself.

Pre-built binaries are available for Linux and macOS on the [releases page](https://github.com/samsonvd/maileryze/releases).

Build from source:
```sh
go install github.com/samsonvd/maileryze@latest
```

For detailed instructions on how to use maileryze, see the [USAGE.md](USAGE.md) file.

## Development setup

### Prerequisites

- [mise](https://mise.jdx.dev) — manages Go and other tool versions

### Development

```sh
mise install
mise run setup
```

### Build, Test and Lint

```sh
mise run build
mise run test
mise run lint
```
