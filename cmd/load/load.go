package load

import (
	"fmt"
	"maileryze/internal/cfg"
	"maileryze/internal/factory"
	"time"

	"github.com/spf13/cobra"
)

const dateLayout = "2006-01-02"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load email data into local storage.",
		Long: `Connect to the email source and download email data to local storage.
The fetched data is: subject, sender, unsubscribe mechanism, unsubscribe info, provider, provider identifier`,
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, err := cmd.PersistentFlags().GetString("alias")
			if err != nil {
				return err
			}

			startStr, _ := cmd.Flags().GetString("start")
			start, err := time.Parse(dateLayout, startStr)
			if err != nil {
				return fmt.Errorf("invalid --start date %q: expected YYYY-MM-DD", startStr)
			}

			end := time.Now()
			if endStr, _ := cmd.Flags().GetString("end"); endStr != "" {
				end, err = time.Parse(dateLayout, endStr)
				if err != nil {
					return fmt.Errorf("invalid --end date %q: expected YYYY-MM-DD", endStr)
				}
			}

			if !end.After(start) {
				return fmt.Errorf("--end (%s) must be after --start (%s)", end.Format(dateLayout), start.Format(dateLayout))
			}

			provider, err := cfg.ProviderByAlias(alias)
			if err != nil {
				return err
			}

			conn, err := factory.NewConnector(*provider)
			if err != nil {
				return err
			}

			if err := conn.Login(); err != nil {
				return err
			}

			emails, err := conn.Fetch(start, end)
			if err != nil {
				return err
			}

			for _, email := range emails {
				fmt.Println(email.Subject)
			}
			return nil
		},
	}
	cmd.Flags().StringP("alias", "a", "", "Email provider alias from your config file")
	cmd.Flags().StringP("start", "s", "", "Start date, inclusive (YYYY-MM-DD)")
	cmd.Flags().StringP("end", "e", "", "End date, exclusive (YYYY-MM-DD, defaults to today)")
	cobra.CheckErr(cmd.MarkFlagRequired("start"))
	cobra.CheckErr(cmd.MarkFlagRequired("alias"))
	return cmd
}
