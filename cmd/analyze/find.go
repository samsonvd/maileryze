package analyze

import (
	"fmt"

	"github.com/spf13/cobra"

	"maileryze/internal/db"
)

func newFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find <query>",
		Short: "Search emails by sender or subject",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, _ := cmd.Root().PersistentFlags().GetString("alias")
			if alias == "" {
				return fmt.Errorf("--alias is required")
			}

			all, _ := cmd.Flags().GetBool("all")

			database, err := db.Open()
			if err != nil {
				return err
			}
			defer database.Close()

			matches, err := db.FindEmails(database, alias, args[0], !all)
			if err != nil {
				return err
			}

			if len(matches) == 0 {
				fmt.Printf("No results for %q\n", args[0])
				return nil
			}

			fmt.Printf("%d results for %q\n\n", len(matches), args[0])

			var lastAddress string
			for _, m := range matches {
				if m.SenderAddress != lastAddress {
					fmt.Printf("%s <%s>\n", m.SenderName, m.SenderAddress)
					lastAddress = m.SenderAddress
				}
				fmt.Printf("  - %s\n", m.Subject)
			}

			return nil
		},
	}
	cmd.Flags().Bool("all", false, "Include emails without an unsubscribe mechanism")
	return cmd
}
