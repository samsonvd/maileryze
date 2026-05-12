package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"maileryze/internal/cfg"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show the current config",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Config: %s\n", viper.ConfigFileUsed())

			c := cfg.Load()

			fmt.Printf("\nProviders (%d):\n", len(c.Providers))
			for _, p := range c.Providers {
				fmt.Printf("  [%s]\n", p.Alias)
				fmt.Printf("    Address:  %s\n", p.Address)
				fmt.Printf("    Provider: %s\n", p.Provider)
			}
		},
	}
	cmd.AddCommand(newCreateCmd())
	return cmd
}
