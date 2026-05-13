package login

import (
	"fmt"

	"github.com/spf13/cobra"

	"maileryze/internal/factory"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to an email provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			alias, err := cmd.Root().PersistentFlags().GetString("alias")
			if err != nil {
				return err
			}
			if alias == "" {
				return fmt.Errorf("--alias is required")
			}

			_, _, err = factory.Connect(alias)
			if err != nil {
				return err
			}

			return nil
		},
	}
}
