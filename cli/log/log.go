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

// Print an info level message
func Info(format string, args ...interface{}) {
	if !Quiet {
		fmt.Printf(format+"\n", args...)
	}
}

// Print an info level message with prefixed key
func InfoWithKey(key string, format string, args ...interface{}) {
	if !Quiet {
		k := color.MagentaString("[%s] ", key)
		fmt.Printf(k+format+"\n", args...)
	}
}

// Print an info level message optionally with prefixed key
func InfoMaybeWithKey(key string, showKey bool, format string, args ...interface{}) {
	if showKey {
		InfoWithKey(key, format, args...)
	} else {
		Info(format, args...)
	}
}

// Print a debug level message
func Debug(format string, args ...interface{}) {
	if !Quiet && Verbose {
		color.HiBlack(format, args...)
	}
}

// Print a warn level message
func Warn(format string, args ...interface{}) {
	if !Quiet {
		k := color.YellowString("[warn] ")
		fmt.Printf(k+format+"\n", args...)
	}
}

// Print an error level message
func Error(err error) {
	k := color.RedString("[error] ")
	fmt.Fprintf(os.Stderr, k+"%v\n", err)
}
