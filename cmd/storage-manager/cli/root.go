package cli

import (
	"os"
	"time"

	"github.com/off-chain-storage/offchain-storage-manager/storage-manager/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string = ""
	log            = util.NewLogger("cli")
)

func init() {
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version") {
		return
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: time.DateTime,
		FullTimestamp:   true,
	})

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().
		StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/storage-manager/config.yaml)")
	rootCmd.PersistentFlags().
		StringP("output", "o", "", "Output format. Empty for human-readable, 'json', 'json-line' or 'yaml'")
	rootCmd.PersistentFlags().
		BoolP("debug", "d", false, "enable debug mode")
}

func initConfig() {
	if cfgFile == "" {
		cfgFile = os.Getenv("STORAGE_MANAGER_CONFIG")
	}

	if cfgFile != "" {
		log.WithField("config_file", cfgFile).Debug("Using config file")
		viper.SetConfigFile(cfgFile)
	} else {
		log.Debug("No config file specified, using default .storage-manager")
		viper.SetConfigName(".storage-manager")
	}

	if err := viper.ReadInConfig(); err != nil {
		log.WithError(err).Error("Error reading config file")
	} else {
		log.WithField("config_file", viper.ConfigFileUsed()).Info("Using config file")
	}
}

var rootCmd = &cobra.Command{
	Use:   "offchain-storage-manager",
	Short: "offchain-storage-manager - off chain storage manager implementation",
	Long:  `offchain-storage-manager is a implementation of Offchain Storage Manager.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("Failed to execute root command")
		os.Exit(1)
	}
}
