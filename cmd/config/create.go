package config

import (
	_ "embed"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"maileryze/internal/cfg"
)

//go:embed sample_config.toml
var sampleConfig []byte

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Generate a default configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			cfgFile := viper.ConfigFileUsed()

			useDefault, err := cmd.Flags().GetBool("default")
			if err != nil {
				log.Fatal(err)
			}
			if cfgFile == "" || useDefault {
				cfgFile = cfg.DefaultConfigPath()
			}

			force, err := cmd.Flags().GetBool("force")
			if err != nil {
				log.Fatal(err)
			}

			if !fileExists(cfgFile) || force {
				log.Printf("Writing config file: %s\n", cfgFile)
				err := os.WriteFile(cfgFile, sampleConfig, 0644)
				cobra.CheckErr(err)
				return
			}

			log.Println("Config file already exists: ", cfgFile)
		},
	}
	cmd.Flags().BoolP("default", "d", false, "Use the default configuration file")
	cmd.Flags().BoolP("force", "f", false, "Overwrite existing config file")
	return cmd
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
