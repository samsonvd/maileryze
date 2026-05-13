package load

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"maileryze/internal/cfg"
	"maileryze/internal/db"
)

func newInspectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inspect",
		Short: "Show stats about locally stored email data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(cfg.DbFilePath()); os.IsNotExist(err) {
				fmt.Println("No local database found. Run 'load' first.")
				return nil
			}

			database, err := db.Open()
			if err != nil {
				return err
			}
			defer database.Close()

			stats, err := db.GetStats(database)
			if err != nil {
				return err
			}

			if len(stats.Aliases) == 0 {
				fmt.Println("Database is empty. Run 'load' first.")
				return nil
			}

			for alias, s := range stats.Aliases {
				fmt.Printf("[%s] (%s)\n", alias, s.Provider)
				fmt.Printf("  Records:      %d\n", s.Count)
				fmt.Printf("  Oldest email: %s\n", s.OldestEmail.Format("2006-01-02"))
				fmt.Printf("  Newest email: %s\n", s.NewestEmail.Format("2006-01-02"))
				fmt.Printf("  Last fetched: %s\n", s.LastFetchedAt.Format("2006-01-02 15:04:05"))
				fmt.Println()
			}

			return nil
		},
	}
}
