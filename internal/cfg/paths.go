package cfg

import (
	"log"
	"os"
	"path"
)

const ConfigFileStub = "maileryze"
const ConfigFileName = "maileryze.toml"

// DataDir returns the directory used for all app data (config, tokens, credentials).
func DataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(home, ".config", "maileryze")
}

// DefaultConfigPath returns the full path to the default config file.
func DefaultConfigPath() string {
	return path.Join(DataDir(), ConfigFileName)
}
