package ui

import (
	"errors"
	"fmt"
	"io"
	"strings"

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

// TableGroup is one row of a GroupedTable: one cell per leading label column and
// one or more lines stacked in the final column as a single multi-line cell.
type TableGroup struct {
	Labels []string
	Rows   []string
}

// GroupedTable prints caption then a boxed table to w whose columns are described
// by header. Each group is one row — its Labels fill the leading columns and its
// Rows stack in the final column — so the labels read as headings spanning the
// stacked lines.
func GroupedTable(w io.Writer, caption string, header []string, groups []TableGroup) {
	data := pterm.TableData{header}
	for _, group := range groups {
		data = append(data, append(append([]string{}, group.Labels...), strings.Join(group.Rows, "\n")))
	}

	fmt.Fprintf(w, "\n%s\n", caption)
	_ = pterm.DefaultTable.
		WithHasHeader().
		WithBoxed().
		WithRowSeparator("─").
		WithData(data).
		WithWriter(w).
		Render()
}
