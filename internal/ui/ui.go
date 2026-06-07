package ui

import (
	"errors"
	"io"

	"github.com/pterm/pterm"
)

// ErrSilent marks a failure whose message a command already printed itself.
// bootstrap recognizes it and exits non-zero without re-printing.
var ErrSilent = errors.New("failure already reported")

var success = pterm.PrefixPrinter{
	MessageStyle: pterm.NewStyle(pterm.FgGreen),
	Prefix:       pterm.Prefix{Style: pterm.NewStyle(pterm.FgGreen)},
}

var failure = pterm.PrefixPrinter{
	MessageStyle: pterm.NewStyle(pterm.FgRed),
	Prefix:       pterm.Prefix{Style: pterm.NewStyle(pterm.FgRed), Text: "❌"},
}

// Success prints msg to w as a green "<prefix> <msg>" line, where prefix is the
// operation's emoji, e.g. "🚀" or "🎉".
func Success(w io.Writer, prefix, msg string) {
	success.WithPrefix(pterm.Prefix{Style: pterm.NewStyle(pterm.FgGreen), Text: prefix}).WithWriter(w).Println(msg)
}

// Failure prints msg to w as a red "❌ <msg>" line.
func Failure(w io.Writer, msg string) { failure.WithWriter(w).Println(msg) }
