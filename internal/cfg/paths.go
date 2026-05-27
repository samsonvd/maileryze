package cfg

import (
	"fmt"
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

// WALPath returns the path to the write-ahead log file for the given alias.
func WALPath(alias string) string {
	return path.Join(DataDir(), fmt.Sprintf("%s_plan.toml", alias))
}
