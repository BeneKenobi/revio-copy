package ui

import "github.com/fatih/color"

// Green prints a message in green.
func Green(format string, a ...interface{}) {
	color.New(color.FgGreen).PrintfFunc()(format, a...)
}

// Yellow prints a message in yellow.
func Yellow(format string, a ...interface{}) {
	color.New(color.FgYellow).PrintfFunc()(format, a...)
}

// Red prints a message in red.
func Red(format string, a ...interface{}) {
	color.New(color.FgRed).PrintfFunc()(format, a...)
}

// Bold prints a message in bold.
func Bold(format string, a ...interface{}) {
	color.New(color.Bold).PrintfFunc()(format, a...)
}

// Italic prints a message in italic.
func Italic(format string, a ...interface{}) {
	color.New(color.Italic).PrintfFunc()(format, a...)
}
