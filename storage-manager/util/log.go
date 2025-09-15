package util

import "github.com/sirupsen/logrus"

func NewLogger(prefix string) *logrus.Entry {
	return logrus.WithField("prefix", prefix)
}
