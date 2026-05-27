package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"maileryze/internal/cfg"
	"maileryze/internal/tui"
	"maileryze/internal/wal"
)

var cfgFile string
var aliasFlag string

var rootCmd = &cobra.Command{
	Use:   "maileryze",
	Short: "Inbox triage tool",
	Long:  "maileryze — take back control of your email.",
	RunE: func(cmd *cobra.Command, args []string) error {
		appConfig := cfg.Load()
		if len(appConfig.Providers) == 0 {
			return fmt.Errorf("no providers configured — add entries to %s", cfg.DefaultConfigPath())
		}

		alias := aliasFlag
		if alias == "" {
			alias = appConfig.Providers[0].Alias
			if len(appConfig.Providers) > 1 {
				fmt.Fprintf(os.Stderr, "Multiple providers configured — using %q. Use --alias to select another.\n", alias)
			}
		} else {
			found := false
			for _, p := range appConfig.Providers {
				if p.Alias == alias {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("alias %q not found in config", alias)
			}
		}

		w, err := wal.Load(cfg.WALPath(alias))
		if err != nil {
			return fmt.Errorf("loading plan: %w", err)
		}

		m := tui.New(alias, w, appConfig)
		p := tea.NewProgram(m, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/maileryze/maileryze.toml)")
	rootCmd.Flags().StringVar(&aliasFlag, "alias", "", "email alias to triage (default: first configured provider)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(cfg.DataDir())
		viper.AddConfigPath(".")
		viper.SetConfigType("toml")
		viper.SetConfigName(cfg.ConfigFileStub)
	}
	viper.AutomaticEnv()
	viper.ReadInConfig() //nolint:errcheck
}
