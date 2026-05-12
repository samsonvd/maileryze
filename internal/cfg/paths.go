package cfg

import (
	"log"
	"os"
	"path"
)

const ConfigFileStub = "maileryze"
const ConfigFileName = "maileryze.toml"

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return path.Join(home, ".config", ConfigFileName)
}
