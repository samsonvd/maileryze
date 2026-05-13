package load

import (
	"fmt"
	"maileryze/internal/db"
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
			alias, err := cmd.Root().PersistentFlags().GetString("alias")
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

			conn, provider, err := factory.Connect(alias)
			if err != nil {
				return err
			}

			database, err := db.Open()
			if err != nil {
				return err
			}
			defer database.Close()

			inserted, skipped := 0, 0
			for result := range conn.Fetch(cmd.Context(), start, end) {
				if result.Err != nil {
					return result.Err
				}
				isNew, err := db.InsertEmail(database, provider.Alias, string(provider.Provider), result.Value)
				if err != nil {
					return err
				}
				if isNew {
					inserted++
				} else {
					skipped++
				}
			}
			fmt.Printf("Done: %d inserted, %d already existed\n", inserted, skipped)
			return nil
		},
	}
	cmd.Flags().StringP("start", "s", "", "Start date, inclusive (YYYY-MM-DD)")
	cmd.Flags().StringP("end", "e", "", "End date, exclusive (YYYY-MM-DD, defaults to today)")
	cobra.CheckErr(cmd.MarkFlagRequired("start"))
	cmd.AddCommand(newInspectCmd())
	return cmd
}
