package errors

import (
	"fmt"
	"runtime"

	"github.com/spf13/viper"
)

const (
	funcInfoFormat = "{%s:%d} [%s]"
	wrapFormat     = "%s\n%w"
)

func getFuncInfo(pc uintptr, file string, line int) string {
	if !viper.GetBool("debug") {
		return ""
	}

	f := runtime.FuncForPC(pc)
	if f == nil {
		return fmt.Sprintf(funcInfoFormat, file, line, "unknown")
	}
	return fmt.Sprintf(funcInfoFormat, file, line, f.Name())
}

func wrap(err error, msg string) error {
	if !viper.GetBool("debug") {
		return fmt.Errorf("%s: %w", msg, err)
	}

	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return fmt.Errorf(wrapFormat, msg, err)
	}

	stack := fmt.Sprintf("%s %s", getFuncInfo(pc, file, line), msg)
	return fmt.Errorf(wrapFormat, stack, err)
}

func WrapWithMessage(err error, msg string) error {
	if err == nil {
		return nil
	}
	return wrap(err, msg)
}

func Wrap(err error) error {
	if err == nil {
		return nil
	}
	return wrap(err, "")
}
