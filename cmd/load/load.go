package load

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "load",
		Short: "Load email data into local storage.",
		Long: `Connect to the email source and download email data to local storage.
The fetched data is: subject, sender, unsubscribe mechanism, unsubscribe info, provider, provider identifier`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("load called")
		},
	}
	return cmd
}
