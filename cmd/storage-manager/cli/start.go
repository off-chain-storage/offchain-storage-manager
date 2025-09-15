package cli

import (
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	storage_manager "github.com/off-chain-storage/offchain-storage-manager/storage-manager"
)

func init() {
	rootCmd.AddCommand(serveCmd)
}

var serveCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the offchain storage manager",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		app, err := storage_manager.NewManager()
		if err != nil {
			logrus.WithError(err).Fatal("Error initializing")
		}

		err = app.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.WithError(err).Fatal("offchain storage ran into an error and had to shut down.")
		}
	},
}
