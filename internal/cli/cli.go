package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/devid-wp/orangepiCLI/internal/config"
)

const usage = `OrangeCTL manages ten universal process slots on Orange Pi.

Usage:
  orangectl init
  orangectl list
  orangectl validate [slot1..slot10]
  orangectl help
`

// exitUsageError is returned when the user invokes the CLI incorrectly:
// unknown command, too many arguments, or a disallowed slot name. It must
// be paired with reportUsageError so stderr carries the message and the
// usage hint.
const exitUsageError = 2

// exitOperationError is returned when a command fails because of an
// operational condition: configuration that cannot be loaded, filesystem
// permissions, and so on. It must be paired with reportOperationError.
const exitOperationError = 1

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return 0
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	case "list":
		if len(args) != 1 {
			return reportUsageError(stderr, "list does not accept arguments")
		}
		return list(stdout, stderr)
	case "init":
		if len(args) != 1 {
			return reportUsageError(stderr, "init does not accept arguments")
		}
		return initialize(stdout, stderr)
	case "validate":
		if len(args) > 2 {
			return reportUsageError(stderr, "validate accepts at most one slot")
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
			if err := config.RequireAllowed(name); err != nil {
				return reportUsageError(stderr, err.Error())
			}
		}
		return validate(stdout, name)
	default:
		return reportUsageError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
}

// reportOperationError prints an operational error to stderr and returns
// the operation-exit code. Every operational error message must flow
// through this helper so the "Error: " prefix and the stderr destination
// stay consistent.
func reportOperationError(stderr io.Writer, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(stderr, "Error: %s\n", err.Error())
	return exitOperationError
}

// reportOperationMessage is the same as reportOperationError but accepts a
// pre-formatted message instead of an error value. Use it when the error
// has been augmented with context (e.g. the slot name that failed to load).
func reportOperationMessage(stderr io.Writer, format string, args ...any) int {
	fmt.Fprintf(stderr, "Error: "+format+"\n", args...)
	return exitOperationError
}

// reportUsageError prints a usage error to stderr (followed by the usage
// hint on a new line) and returns the usage-exit code.
func reportUsageError(stderr io.Writer, message string) int {
	fmt.Fprintf(stderr, "Error: %s\n\n%s", message, usage)
	return exitUsageError
}

func initialize(stdout, stderr io.Writer) int {
	result, err := config.Initialize()
	if err != nil {
		return reportOperationError(stderr, err)
	}
	fmt.Fprintf(stdout, "Initialized OrangeCTL: %d created, %d kept\n", len(result.Created), len(result.Existing))
	fmt.Fprintf(stdout, "Config directory: %s\n", config.Directory())
	return 0
}

func list(stdout, stderr io.Writer) int {
	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tNAME\tENABLED\tSTATUS")
	failed := false
	for _, name := range config.AllowedSlots {
		slot, err := config.Load(name)
		if err != nil {
			failed = true
			fmt.Fprintf(w, "%s\t-\t-\terror\n", name)
			reportOperationMessage(stderr, "failed to load slot %q: %s", name, err.Error())
			continue
		}
		enabled, status := "no", "disabled"
		if slot.Enabled {
			enabled, status = "yes", "stopped"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, slot.DisplayName, enabled, status)
	}
	w.Flush()
	if failed {
		return exitOperationError
	}
	return 0
}

func validate(stdout io.Writer, requested string) int {
	targets := config.AllowedSlots
	if requested != "" {
		targets = []string{requested}
	}

	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tRESULT")
	failed := false
	for _, name := range targets {
		slot, validationErrors := config.LoadAndValidate(name)
		result := "OK"
		if len(validationErrors) > 0 {
			failed = true
			result = config.FormatErrors(validationErrors)
		} else if !slot.Enabled {
			result = "disabled"
		}
		fmt.Fprintf(w, "%s\t%s\n", name, result)
	}
	w.Flush()
	if failed {
		return exitOperationError
	}
	return 0
}