package report

import (
	"fmt"

	"github.com/fatih/color"
)

type colorFunc func(string, ...interface{}) string

// Warn prints a formatted warn message to stdout.
func Warn(format string, args ...interface{}) {
	report(color.YellowString, "WARN", format, args...)
}

// Error prints a formatted error message to stdout.
func Error(format string, args ...interface{}) {
	report(color.RedString, "ERROR", format, args...)
}

// Info prints a formatted info message to stdout.
func Info(format string, args ...interface{}) {
	report(color.GreenString, "INFO", format, args...)
}

func report(color colorFunc, lvl string, format string, args ...interface{}) {
	fmt.Printf("%s: %s\n", color(lvl), fmt.Sprintf(format, args...))
}
