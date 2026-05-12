package load

import (
	"fmt"
	"maileryze/internal/cfg"
	"maileryze/internal/factory"
	"time"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load email data into local storage.",
		Long: `Connect to the email source and download email data to local storage.
The fetched data is: subject, sender, unsubscribe mechanism, unsubscribe info, provider, provider identifier`,
		Run: func(cmd *cobra.Command, args []string) {
			alias, err := cmd.Flags().GetString("alias")
			cobra.CheckErr(err)

			provider, err := cfg.ProviderByAlias(alias)
			cobra.CheckErr(err)

			conn, err := factory.NewConnector(*provider)
			cobra.CheckErr(err)

			err = conn.Login()
			cobra.CheckErr(err)

			startTime := time.Now().AddDate(0, 0, -1)
			endTime := time.Now()
			emails, err := conn.Fetch(startTime, endTime)

			for _, email := range emails {
				fmt.Println(email.Subject)
			}
		},
	}
	cmd.Flags().StringP("alias", "a", "", "alias for provider")
	return cmd
}
