/*
Copyright © 2026 Samson Ventura-Danziger samson@danziger.uk
*/
package cmd

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"maileryze/cmd/config"
	"maileryze/cmd/load"
	"maileryze/cmd/login"
	"maileryze/internal/cfg"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "maileryze",
	Short: "Email analyzer and decluttering tool",
	Long: `Take back control of your email by seeing who is creating the clutter, and then deleting it.
This tool is open-source and your data never leaves your device.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/maileryze.toml)")
	rootCmd.PersistentFlags().StringP("alias", "a", "", "Email provider alias from your config file.")

	rootCmd.AddCommand(config.NewCmd())
	rootCmd.AddCommand(load.NewCmd())
	rootCmd.AddCommand(login.NewCmd())
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		log.Printf("Reading config from %s", viper.ConfigFileUsed())
	} else {
		viper.AddConfigPath(cfg.DefaultConfigPath())
		viper.AddConfigPath(".")
		viper.SetConfigType("toml")
		viper.SetConfigName(cfg.ConfigFileStub)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}
}
