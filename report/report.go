package report // import "gopkg.in/src-d/proteus.v1/report"

import (
	"fmt"

	"github.com/fatih/color"
)

var silent bool
var testing bool
var msgStack []string

func Silent() {
	silent = true
}

func TestMode() {
	testing = true
	ResetTestModeStack()
}

func EndTestMode() {
	testing = false
	ResetTestModeStack()
}

func ResetTestModeStack() {
	msgStack = make([]string, 0)
}

func MessageStack() []string {
	return msgStack
}

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
	fmt.Sprintf("%s: %s", color(lvl), fmt.Sprintf(format, args...))

	if testing {
		msgStack = append(msgStack, fmt.Sprintf("%s: %s", lvl, fmt.Sprintf(format, args...)))
	}

	if !silent || lvl == "ERROR" {
		fmt.Println(fmt.Sprintf("%s: %s", color(lvl), fmt.Sprintf(format, args...)))
	}
}
