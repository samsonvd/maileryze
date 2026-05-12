package login

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"

	"maileryze/internal/cfg"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to an email provider",
		Run: func(cmd *cobra.Command, args []string) {
			alias, err := cmd.Root().PersistentFlags().GetString("alias")
			cobra.CheckErr(err)
			if alias == "" {
				cobra.CheckErr("You must provide an alias")
			}

			provider, err := cfg.ProviderByAlias(alias)
			cobra.CheckErr(err)

			log.Println("Using alias:", alias)
			fmt.Println(provider)
		},
	}
}
