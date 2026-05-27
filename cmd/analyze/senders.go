package analyze

import (
	"fmt"

	"github.com/spf13/cobra"

	"maileryze/internal/db"
)

func newSendersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "senders",
		Short: "Show senders ranked by email volume with their subjects",
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, _ := cmd.Root().PersistentFlags().GetString("alias")
			if alias == "" {
				return fmt.Errorf("--alias is required")
			}

			min, _ := cmd.Flags().GetInt("min")
			all, _ := cmd.Flags().GetBool("all")

			database, err := db.Open()
			if err != nil {
				return err
			}
			defer database.Close()

			senders, err := db.GetSenders(database, alias, min, !all)
			if err != nil {
				return err
			}

			if len(senders) == 0 {
				fmt.Println("No data found. Run 'load' first.")
				return nil
			}

			for _, s := range senders {
				fmt.Printf("%s <%s> (%d)\n", s.Name, s.Address, s.Count)
				for _, subject := range s.Subjects {
					fmt.Printf("  - %s\n", subject)
				}
				fmt.Println()
			}

			return nil
		},
	}
	cmd.Flags().Int("min", 20, "Minimum number of emails to show a sender")
	cmd.Flags().Bool("all", false, "Include senders without an unsubscribe mechanism")
	return cmd
}
