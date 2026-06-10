package ui

import (
	"errors"
	"fmt"
	"io"
)

var ErrSilent = errors.New("failure already reported")

func Success(w io.Writer, prefix, msg string) {
	fmt.Fprintf(w, "%s %s\n", prefix, msg)
}

func Failure(w io.Writer, prefix, msg string) {
	fmt.Fprintf(w, "%s %s\n", prefix, msg)
}
