package analyze

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze locally stored email data",
	}
	cmd.AddCommand(newFindCmd())
	cmd.AddCommand(newSendersCmd())
	return cmd
}
