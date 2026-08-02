package cli

import (
	"fmt"
	"io"
	"strings"
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
			return usageError(stderr, "list does not accept arguments")
		}
		return list(stdout)
	case "init":
		if len(args) != 1 {
			return usageError(stderr, "init does not accept arguments")
		}
		return initialize(stdout, stderr)
	case "validate":
		if len(args) > 2 {
			return usageError(stderr, "validate accepts at most one slot")
		}
		name := ""
		if len(args) == 2 {
			name = args[1]
			if err := config.RequireAllowed(name); err != nil {
				fmt.Fprintf(stderr, "Error: %v\n", err)
				return 2
			}
		return validate(stdout, name)
	default:
		return usageError(stderr, fmt.Sprintf("unknown command %q", args[0]))
	}
}

func initialize(stdout, stderr io.Writer) int {
	result, err := config.Initialize()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Initialized OrangeCTL: %d created, %d kept\n", len(result.Created), len(result.Existing))
	fmt.Fprintf(stdout, "Config directory: %s\n", config.Directory())
	return 0
}

func usageError(stderr io.Writer, message string) int {
	fmt.Fprintf(stderr, "Error: %s\n\n%s", message, usage)
	return 2
}

func list(stdout io.Writer) int {
	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tNAME\tENABLED\tSTATUS")
	failed := false
	for _, name := range config.AllowedSlots {
		slot, err := config.Load(name)
		if err != nil {
			failed = true
			fmt.Fprintf(w, "%s\t-\t-\tconfig error\n", name)
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
		return 1
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
		slot, errors := config.LoadAndValidate(name)
		result := "OK"
		if len(errors) > 0 {
			failed = true
			result = strings.Join(errors, "; ")
		} else if !slot.Enabled {
			result = "disabled"
		}
		fmt.Fprintf(w, "%s\t%s\n", name, result)
	}
	w.Flush()
	if failed {
		return 1
	}
	return 0
}
