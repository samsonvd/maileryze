package analyze

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Analyze locally stored email data",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("analyze called")
			return nil
		},
	}
}
