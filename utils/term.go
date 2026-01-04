package utils

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

var (
	IsNonStdOutTerminal = !terminal.IsTerminal(int(os.Stdout.Fd()))
	IsNonStdErrTerminal = !terminal.IsTerminal(int(os.Stderr.Fd()))
	IsNonTerminal       = IsNonStdOutTerminal || IsNonStdErrTerminal
)

const (
	TermReset  = "\033[0m"
	TermBold   = "\033[1m"
	TermUnder  = "\033[4m"
	TermRed    = "\033[31m"
	TermGreen  = "\033[32m"
	TermYellow = "\033[33m"
	TermBlue   = "\033[34m"
)

type StdErrOut struct {
	stdFile *os.File
}

var Stdout = StdErrOut{
	stdFile: os.Stdout,
}
var Stderr = StdErrOut{
	stdFile: os.Stderr,
}

func (s *StdErrOut) Print(message string) {
	_, _ = fmt.Fprint(s.stdFile, message)
}

func (s *StdErrOut) Println(message string) {
	_, _ = fmt.Fprintln(s.stdFile, message)
}

func (s *StdErrOut) Fprintf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(s.stdFile, format, args...)
}

func Bold(str string, color bool) string {
	return WithColor(str, TermBold, color)
}

func Under(str string, color bool) string {
	return WithColor(str, TermUnder, color)
}

func Red(str string, color bool) string {
	return WithColor(str, TermRed, color)
}

func Blue(str string, color bool) string {
	return WithColor(str, TermBlue, color)
}

func Green(str string, color bool) string {
	return WithColor(str, TermGreen, color)
}

func Yellow(str string, color bool) string {
	return WithColor(str, TermYellow, color)
}

func WBold(str string) string {
	return WithColor(str, TermBold, true)
}
func WUnder(str string) string {
	return WithColor(str, TermUnder, true)
}
func WRed(str string) string {
	return WithColor(str, TermRed, true)
}

func WBlue(str string) string {
	return WithColor(str, TermBlue, true)
}
func WGreen(str string) string {
	return WithColor(str, TermGreen, true)
}

func WYellow(str string) string {
	return WithColor(str, TermYellow, true)
}

func WithColor(str, termColor string, color bool) string {
	if !color {
		return str
	}
	if IsNonTerminal {
		return str
	}
	return fmt.Sprintf("%s%s%s", termColor, str, TermReset)
}
