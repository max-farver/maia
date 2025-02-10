package codecov

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "pr-codecov",
	Short: "Generate a coverage report for the current PR.",
	RunE: func(cmd *cobra.Command, args []string) error {
		diff, err := GetDiff()
		if err != nil {
			return err
		}

		coverage, err := GetCoverage(diff, "output.txt")
		if err != nil {
			return err
		}

		fmt.Println(coverage)
		return nil
	},
}
