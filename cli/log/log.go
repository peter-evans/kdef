// Package log implements a logging interface.
package log

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	Quiet   = false
	Verbose = false
)

// Infof prints an info level message.
func Infof(format string, args ...interface{}) {
	if !Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// InfoWithKeyf prints an info level message with prefixed key.
func InfoWithKeyf(key string, format string, args ...interface{}) {
	if !Quiet {
		k := color.MagentaString("[%s] ", key)
		fmt.Printf(k+format+"\n", args...)
	}
}

// InfoMaybeWithKeyf prints an info level message optionally with prefixed key.
func InfoMaybeWithKeyf(key string, showKey bool, format string, args ...interface{}) {
	if showKey {
		InfoWithKeyf(key, format, args...)
	} else {
		Infof(format, args...)
	}
}

// Debugf prints a debug level message.
func Debugf(format string, args ...interface{}) {
	if !Quiet && Verbose {
		color.HiBlack(format, args...)
	}
}

// Warnf prints a warn level message.
func Warnf(format string, args ...interface{}) {
	if !Quiet {
		k := color.YellowString("[warn] ")
		fmt.Printf(k+format+"\n", args...)
	}
}

// Error prints an error level message.
func Error(err error) {
	k := color.RedString("[error] ")
	fmt.Fprintf(os.Stderr, k+"%v\n", err)
}
