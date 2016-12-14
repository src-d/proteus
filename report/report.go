package report

import (
	"fmt"

	"github.com/fatih/color"
)

type Color func(string, ...interface{}) string

func Warn(format string, args ...interface{}) {
	report(color.YellowString, "WARN", format, args...)
}

func Error(format string, args ...interface{}) {
	report(color.RedString, "ERROR", format, args...)
}

func Info(format string, args ...interface{}) {
	report(color.GreenString, "INFO", format, args...)
}

func report(color Color, lvl string, format string, args ...interface{}) {
	fmt.Printf("%s: %s\n", color(lvl), fmt.Sprintf(format, args...))
}
