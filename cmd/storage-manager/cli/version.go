package cli

import (
	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/types"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version.",
	Long:  "The version of offchain storage manager.",
	Run: func(cmd *cobra.Command, args []string) {
		output, _ := cmd.Flags().GetString("output")
		SuccessOutput(map[string]string{
			"version": types.Version,
			"commit":  types.GitCommitHash,
		}, types.Version, output)
	},
}
