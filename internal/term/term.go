// Package term centralises the decision of whether ANSI colour escapes
// should be emitted to a writer.
//
// Rules implemented (in order of priority):
//
//  1. JSON output is always colour-free.
//  2. The explicit --no-color flag disables colour.
//  3. The NO_COLOR environment variable disables colour when set to any
//     non-empty value (https://no-color.org/).
//  4. TERM=dumb disables colour.
//  5. A non-TTY stdout disables colour (so redirected output stays
//     machine-readable).
//
// The package has no global state and no side effects; tests inject an
// environment getter and a "force TTY" switch so they can run without a
// real terminal.
package term

import (
	"io"
	"os"

	"golang.org/x/term"
)

// Options carries the inputs that influence the colour decision.
type Options struct {
	// NoColorFlag mirrors --no-color on the command line.
	NoColorFlag bool
	// JSONOutput mirrors --json. When true, ANSI escapes are never emitted.
	JSONOutput bool
	// ForceTTY bypasses the isTerminal check. It exists for tests that
	// want to exercise the "yes, this is a terminal" branch without an
	// actual TTY. Production code must leave it false.
	ForceTTY bool
	// Getenv is used to read NO_COLOR and TERM. nil means use os.Getenv.
	Getenv func(string) string
}

// ShouldColor reports whether stdout should receive ANSI colour escapes
// under the given options.
func ShouldColor(stdout io.Writer, opts Options) bool {
	if opts.JSONOutput {
		return false
	}
	if opts.NoColorFlag {
		return false
	}
	get := opts.Getenv
	if get == nil {
		get = defaultGetenv
	}
	if v, ok := lookupEnv(get, "NO_COLOR"); ok && v != "" {
		return false
	}
	if v, ok := lookupEnv(get, "TERM"); ok && v == "dumb" {
		return false
	}
	if opts.ForceTTY {
		return true
	}
	return isTerminalWriter(stdout)
}

func lookupEnv(get func(string) string, key string) (string, bool) {
	v := get(key)
	if v == "" {
		return "", false
	}
	return v, true
}

func defaultGetenv(key string) string {
	return os.Getenv(key)
}

// ANSI escape sequences. We only use the eight-colour set because the
// project targets Orange Pi terminals and SSH sessions, where true-colour
// support is patchy.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
)

// Paint wraps text in an ANSI escape pair if enabled is true. Otherwise
// it returns text unchanged. The function is safe to call with an empty
// text or an unknown code; the wrapped result is what the caller prints.
func Paint(enabled bool, code, text string) string {
	if !enabled || text == "" {
		return text
	}
	return code + text + ansiReset
}

// Bold is a convenience wrapper around Paint using the bold escape.
func Bold(enabled bool, text string) string { return Paint(enabled, ansiBold, text) }

// Dim returns text in dim grey when enabled is true.
func Dim(enabled bool, text string) string { return Paint(enabled, ansiDim, text) }

// Red returns text in red when enabled is true.
func Red(enabled bool, text string) string { return Paint(enabled, ansiRed, text) }

// Green returns text in green when enabled is true.
func Green(enabled bool, text string) string { return Paint(enabled, ansiGreen, text) }

// Yellow returns text in yellow when enabled is true.
func Yellow(enabled bool, text string) string { return Paint(enabled, ansiYellow, text) }

// isTerminalWriter reports whether w is backed by a file descriptor that
// refers to a terminal. Writers that do not expose Fd() (e.g.
// bytes.Buffer used in tests) are reported as non-terminal.
func isTerminalWriter(w io.Writer) bool {
	type fdWriter interface{ Fd() uintptr }
	fd, ok := w.(fdWriter)
	if !ok {
		return false
	}
	return term.IsTerminal(int(fd.Fd()))
}