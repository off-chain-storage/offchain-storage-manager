package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

func SuccessOutput(result interface{}, override string, outputFormat string) {
	fmt.Println(output(result, override, outputFormat))
	os.Exit(0)
}

func output(result interface{}, override string, outputFormat string) string {
	var jsonBytes []byte
	var err error
	switch outputFormat {
	case "json":
		jsonBytes, err = json.MarshalIndent(result, "", "\t")
		if err != nil {
			logrus.WithError(err).Fatal("failed to unmarshal output")
		}
	case "json-line":
		jsonBytes, err = json.Marshal(result)
		if err != nil {
			logrus.WithError(err).Fatal("failed to unmarshal output")
		}
	case "yaml":
		jsonBytes, err = yaml.Marshal(result)
		if err != nil {
			logrus.WithError(err).Fatal("failed to unmarshal output")
		}
	default:
		// nolint
		return override
	}

	return string(jsonBytes)
}
