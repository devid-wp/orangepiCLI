package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/devid-wp/orangepiCLI/internal/config"
)

const usage = `OrangeCTL manages ten universal process slots on Orange Pi.

Usage:
  orangectl [--json] init
  orangectl [--json] list
  orangectl [--json] validate [slot1..slot10]
  orangectl [--json] help
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

// globalOptions carries CLI-wide flags that can be applied to any
// command. They are extracted from the raw argument list before the
// dispatcher consults args[0].
type globalOptions struct {
	jsonOutput bool
}

// parseGlobalOptions walks the argument list and separates global
// flags from the positional command and its arguments. The cleaned
// slice preserves the original order of the remaining tokens. Unknown
// global flags return a usage-class error so the caller can route it
// through reportUsageError.
func parseGlobalOptions(args []string) (cleaned []string, opts globalOptions, err error) {
	for _, arg := range args {
		switch arg {
		case "--json", "-json":
			opts.jsonOutput = true
		case "-h", "--help", "help":
			cleaned = append(cleaned, arg)
		default:
			if len(arg) > 1 && arg[0] == '-' {
				return nil, opts, fmt.Errorf("unknown global flag %q", arg)
			}
			cleaned = append(cleaned, arg)
		}
	}
	return cleaned, opts, nil
}

// jsonInitResult is the --json shape of `orangectl init`.
type jsonInitResult struct {
	Created   []string `json:"created"`
	Existing  []string `json:"existing"`
	ConfigDir string   `json:"config_dir"`
}

// jsonListSlot is one element of the `slots` array in `orangectl list --json`.
type jsonListSlot struct {
	Slot        string `json:"slot"`
	DisplayName string `json:"display_name,omitempty"`
	Enabled     bool   `json:"enabled"`
	Status      string `json:"status"`
}

// jsonListError reports a slot that could not be loaded.
type jsonListError struct {
	Slot    string `json:"slot"`
	Message string `json:"message"`
}

// jsonListResult is the --json shape of `orangectl list`.
type jsonListResult struct {
	Slots  []jsonListSlot  `json:"slots"`
	Errors []jsonListError `json:"errors"`
}

// jsonValidateError mirrors config.ValidationError without exposing
// secrets. Only Field, Code and Message are forwarded.
type jsonValidateError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// jsonValidateSlot is one element of the `results` array in
// `orangectl validate --json`. Result is "OK", "disabled" or "errors";
// in the last case the Errors slice is populated.
type jsonValidateSlot struct {
	Slot   string              `json:"slot"`
	Result string              `json:"result"`
	Errors []jsonValidateError `json:"errors,omitempty"`
}

// jsonValidateResult is the --json shape of `orangectl validate`.
type jsonValidateResult struct {
	Results []jsonValidateSlot `json:"results"`
}

// jsonHelpCommand describes one entry in the help index.
type jsonHelpCommand struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

// jsonHelpResult is the --json shape of `orangectl help`.
type jsonHelpResult struct {
	Commands []jsonHelpCommand `json:"commands"`
	Usage    string            `json:"usage"`
}

// commandSummaries lists every CLI command with a short description.
// The order matches the order in runCommand.
var commandSummaries = []jsonHelpCommand{
	{Name: "init", Summary: "create missing configuration directories and slot files"},
	{Name: "list", Summary: "show all ten slots with enabled flag and basic status"},
	{Name: "validate", Summary: "validate one slot or all slots and report errors"},
	{Name: "help", Summary: "show usage information"},
}

func Run(args []string, stdout, stderr io.Writer) int {
	cleaned, opts, err := parseGlobalOptions(args)
	if err != nil {
		return reportUsageError(stderr, err.Error())
	}
	if len(cleaned) == 0 {
		// No command: behave like before (print usage), honour --json.
		if opts.jsonOutput {
			return writeJSON(stdout, jsonHelpResult{Commands: commandSummaries, Usage: usage})
		}
		fmt.Fprint(stdout, usage)
		return 0
	}

	switch cleaned[0] {
	case "help", "-h", "--help":
		return help(stdout, opts)
	case "list":
		if len(cleaned) != 1 {
			return reportUsageError(stderr, "list does not accept arguments")
		}
		return list(stdout, stderr, opts)
	case "init":
		if len(cleaned) != 1 {
			return reportUsageError(stderr, "init does not accept arguments")
		}
		return initialize(stdout, stderr, opts)
	case "validate":
		if len(cleaned) > 2 {
			return reportUsageError(stderr, "validate accepts at most one slot")
		}
		name := ""
		if len(cleaned) == 2 {
			name = cleaned[1]
			if err := config.RequireAllowed(name); err != nil {
				return reportUsageError(stderr, err.Error())
			}
		}
		return validate(stdout, name, opts)
	default:
		return reportUsageError(stderr, fmt.Sprintf("unknown command %q", cleaned[0]))
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

// writeJSON marshals value to stdout with a trailing newline. The encoder
// is configured with HTMLEscape disabled so output is safe for piping
// to tools like jq.
func writeJSON(stdout io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return exitOperationError
	}
	return 0
}

func help(stdout io.Writer, opts globalOptions) int {
	if opts.jsonOutput {
		return writeJSON(stdout, jsonHelpResult{Commands: commandSummaries, Usage: usage})
	}
	fmt.Fprint(stdout, usage)
	return 0
}

func initialize(stdout, stderr io.Writer, opts globalOptions) int {
	result, err := config.Initialize()
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if opts.jsonOutput {
		return writeJSON(stdout, jsonInitResult{
			Created:   result.Created,
			Existing:  result.Existing,
			ConfigDir: config.Directory(),
		})
	}
	fmt.Fprintf(stdout, "Initialized OrangeCTL: %d created, %d kept\n", len(result.Created), len(result.Existing))
	fmt.Fprintf(stdout, "Config directory: %s\n", config.Directory())
	return 0
}

func list(stdout, stderr io.Writer, opts globalOptions) int {
	if opts.jsonOutput {
		return listJSON(stdout, stderr)
	}
	return listText(stdout, stderr)
}

// listText prints the human-readable table and writes per-slot load
// failures to stderr.
func listText(stdout, stderr io.Writer) int {
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

// listJSON prints the structured representation. Slot-load failures are
// embedded in the JSON instead of being written to stderr so the stdout
// stream remains a single valid JSON document.
func listJSON(stdout, stderr io.Writer) int {
	result := jsonListResult{
		Slots:  make([]jsonListSlot, 0, len(config.AllowedSlots)),
		Errors: []jsonListError{},
	}
	failed := false
	for _, name := range config.AllowedSlots {
		slot, err := config.Load(name)
		if err != nil {
			failed = true
			result.Errors = append(result.Errors, jsonListError{Slot: name, Message: err.Error()})
			result.Slots = append(result.Slots, jsonListSlot{Slot: name, Status: "error"})
			continue
		}
		enabled, status := false, "disabled"
		if slot.Enabled {
			enabled, status = true, "stopped"
		}
		result.Slots = append(result.Slots, jsonListSlot{
			Slot:        name,
			DisplayName: slot.DisplayName,
			Enabled:     enabled,
			Status:      status,
		})
	}
	if writeJSON(stdout, result) != 0 {
		reportOperationMessage(stderr, "failed to encode list result as JSON")
		return exitOperationError
	}
	if failed {
		return exitOperationError
	}
	return 0
}

func validate(stdout io.Writer, requested string, opts globalOptions) int {
	targets := config.AllowedSlots
	if requested != "" {
		targets = []string{requested}
	}
	if opts.jsonOutput {
		return validateJSON(stdout, targets)
	}
	return validateText(stdout, targets)
}

func validateText(stdout io.Writer, targets []string) int {
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

func validateJSON(stdout io.Writer, targets []string) int {
	results := make([]jsonValidateSlot, 0, len(targets))
	failed := false
	for _, name := range targets {
		entry := jsonValidateSlot{Slot: name}
		slot, validationErrors := config.LoadAndValidate(name)
		switch {
		case len(validationErrors) > 0:
			failed = true
			entry.Result = "errors"
			entry.Errors = make([]jsonValidateError, 0, len(validationErrors))
			for _, validationError := range validationErrors {
				entry.Errors = append(entry.Errors, jsonValidateError{
					Field:   validationError.Field,
					Code:    string(validationError.Code),
					Message: validationError.Message,
				})
			}
		case !slot.Enabled:
			entry.Result = "disabled"
		default:
			entry.Result = "OK"
		}
		results = append(results, entry)
	}
	if writeJSON(stdout, jsonValidateResult{Results: results}) != 0 {
		return exitOperationError
	}
	if failed {
		return exitOperationError
	}
	return 0
}