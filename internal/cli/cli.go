package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/devid-wp/orangepiCLI/internal/buildinfo"
	"github.com/devid-wp/orangepiCLI/internal/config"
	"github.com/devid-wp/orangepiCLI/internal/logview"
	"github.com/devid-wp/orangepiCLI/internal/process"
	"github.com/devid-wp/orangepiCLI/internal/term"
)

const usage = `OrangeCTL manages ten universal process slots on Orange Pi.

Usage:
  orangectl [--json] [--no-color] init
  orangectl [--json] [--no-color] list
  orangectl [--json] [--no-color] validate [slot1..slot10]
  orangectl [--json] [--no-color] start <slot1..slot10>
  orangectl [--json] [--no-color] status [slot1..slot10]
  orangectl [--json] [--no-color] [--timeout 10s] [--force] stop <slot1..slot10>
  orangectl [--json] [--no-color] restart <slot1..slot10>
  orangectl [--json] [--no-color] [--lines N] [--follow] logs <slot1..slot10>
  orangectl [--json] [--no-color] version
  orangectl [--json] [--no-color] help

Global flags:
  --json       Emit machine-readable JSON on stdout.
  --no-color   Disable ANSI colour escapes. Respects NO_COLOR and TERM=dumb.
  --timeout D  Stop timeout (default: 10s); valid only with stop.
  --force      Send SIGKILL after stop timeout; valid only with stop.
  --lines N    Number of log lines; valid only with logs.
  --follow     Continue streaming a log until Ctrl+C; valid only with logs.

Each command's exit code is 0 on success, 1 on operational failure, and 2
on usage errors (unknown command, bad arguments, unknown slot).
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
	jsonOutput  bool
	noColor     bool
	stopTimeout time.Duration
	timeoutSet  bool
	force       bool
	logLines    int
	linesSet    bool
	follow      bool
}

// parseGlobalOptions walks the argument list and separates global
// flags from the positional command and its arguments. The cleaned
// slice preserves the original order of the remaining tokens. Unknown
// global flags return a usage-class error so the caller can route it
// through reportUsageError.
func parseGlobalOptions(args []string) (cleaned []string, opts globalOptions, err error) {
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--json", "-json":
			opts.jsonOutput = true
		case "--no-color", "-no-color":
			opts.noColor = true
		case "--timeout":
			if index+1 >= len(args) {
				return nil, opts, fmt.Errorf("--timeout requires a duration")
			}
			index++
			duration, parseErr := time.ParseDuration(args[index])
			if parseErr != nil || duration <= 0 {
				return nil, opts, fmt.Errorf("--timeout must be a positive duration")
			}
			opts.stopTimeout, opts.timeoutSet = duration, true
		case "--force":
			opts.force = true
		case "--lines":
			if index+1 >= len(args) {
				return nil, opts, fmt.Errorf("--lines requires a positive integer")
			}
			index++
			count, parseErr := strconv.Atoi(args[index])
			if parseErr != nil || count <= 0 {
				return nil, opts, fmt.Errorf("--lines requires a positive integer")
			}
			opts.logLines, opts.linesSet = count, true
		case "--follow":
			opts.follow = true
		case "-h", "--help", "help":
			cleaned = append(cleaned, arg)
		default:
			if strings.HasPrefix(arg, "--timeout=") {
				duration, parseErr := time.ParseDuration(strings.TrimPrefix(arg, "--timeout="))
				if parseErr != nil || duration <= 0 {
					return nil, opts, fmt.Errorf("--timeout must be a positive duration")
				}
				opts.stopTimeout, opts.timeoutSet = duration, true
				continue
			}
			if strings.HasPrefix(arg, "--lines=") {
				count, parseErr := strconv.Atoi(strings.TrimPrefix(arg, "--lines="))
				if parseErr != nil || count <= 0 {
					return nil, opts, fmt.Errorf("--lines requires a positive integer")
				}
				opts.logLines, opts.linesSet = count, true
				continue
			}
			if len(arg) > 1 && arg[0] == '-' {
				return nil, opts, fmt.Errorf("unknown global flag %q", arg)
			}
			cleaned = append(cleaned, arg)
		}
	}
	return cleaned, opts, nil
}

// shouldColor centralises the term.Options construction so every command
// receives a consistent decision based on the global flags.
func (opts globalOptions) shouldColor(stdout io.Writer) bool {
	return term.ShouldColor(stdout, term.Options{
		NoColorFlag: opts.noColor,
		JSONOutput:  opts.jsonOutput,
	})
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

// jsonVersionResult is the stable --json shape of `orangectl version`.
type jsonVersionResult struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

type jsonStartResult struct {
	State process.ProcessState `json:"state"`
}

type jsonStatusResult struct {
	Results []process.SlotStatus `json:"results"`
}

// commandSummaries lists every CLI command with a short description.
// The order matches the order in runCommand.
var commandSummaries = []jsonHelpCommand{
	{Name: "init", Summary: "create missing configuration directories and slot files"},
	{Name: "list", Summary: "show all ten slots with enabled flag and basic status"},
	{Name: "validate", Summary: "validate one slot or all slots and report errors"},
	{Name: "start", Summary: "start an enabled, valid slot process"},
	{Name: "status", Summary: "show running, stopped, or stale process state"},
	{Name: "stop", Summary: "stop a verified slot process"},
	{Name: "restart", Summary: "stop then start a slot process"},
	{Name: "logs", Summary: "show the last 50 lines of a slot log"},
	{Name: "version", Summary: "show version, source revision, and build date"},
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
	if (opts.timeoutSet || opts.force) && cleaned[0] != "stop" {
		return reportUsageError(stderr, "--timeout and --force are valid only with stop")
	}
	if (opts.linesSet || opts.follow) && cleaned[0] != "logs" {
		return reportUsageError(stderr, "--lines is valid only with logs")
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
	case "start":
		if len(cleaned) != 2 {
			return reportUsageError(stderr, "start requires exactly one slot")
		}
		if err := config.RequireAllowed(cleaned[1]); err != nil {
			return reportUsageError(stderr, err.Error())
		}
		return start(stdout, stderr, cleaned[1], opts)
	case "status":
		if len(cleaned) > 2 {
			return reportUsageError(stderr, "status accepts at most one slot")
		}
		name := ""
		if len(cleaned) == 2 {
			name = cleaned[1]
			if err := config.RequireAllowed(name); err != nil {
				return reportUsageError(stderr, err.Error())
			}
		}
		return status(stdout, stderr, name, opts)
	case "stop":
		if len(cleaned) != 2 {
			return reportUsageError(stderr, "stop requires exactly one slot")
		}
		if err := config.RequireAllowed(cleaned[1]); err != nil {
			return reportUsageError(stderr, err.Error())
		}
		return stop(stdout, stderr, cleaned[1], opts)
	case "restart":
		if len(cleaned) != 2 {
			return reportUsageError(stderr, "restart requires exactly one slot")
		}
		if err := config.RequireAllowed(cleaned[1]); err != nil {
			return reportUsageError(stderr, err.Error())
		}
		return restart(stdout, stderr, cleaned[1], opts)
	case "logs":
		if len(cleaned) != 2 {
			return reportUsageError(stderr, "logs requires exactly one slot")
		}
		if err := config.RequireAllowed(cleaned[1]); err != nil {
			return reportUsageError(stderr, err.Error())
		}
		count := 50
		if opts.linesSet {
			count = opts.logLines
		}
		return logs(stdout, stderr, cleaned[1], opts, count)
	case "version":
		if len(cleaned) != 1 {
			return reportUsageError(stderr, "version does not accept arguments")
		}
		return version(stdout, opts)
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

// version prints build metadata. Release builds overwrite these defaults with
// -ldflags -X, while local builds retain meaningful development values.
func version(stdout io.Writer, opts globalOptions) int {
	result := jsonVersionResult{
		Version:   buildinfo.Version,
		Commit:    buildinfo.Commit,
		BuildDate: buildinfo.BuildDate,
	}
	if opts.jsonOutput {
		return writeJSON(stdout, result)
	}
	fmt.Fprintf(stdout, "Version: %s\nCommit: %s\nBuild date: %s\n", result.Version, result.Commit, result.BuildDate)
	return 0
}

func start(stdout, stderr io.Writer, name string, opts globalOptions) int {
	slot, err := config.Load(name)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	state, err := process.DefaultManager().Start(slot)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if opts.jsonOutput {
		return writeJSON(stdout, jsonStartResult{State: state})
	}
	fmt.Fprintf(stdout, "Started %s (PID %d)\n", state.Slot, state.PID)
	return 0
}

func status(stdout, stderr io.Writer, requested string, opts globalOptions) int {
	targets := config.AllowedSlots
	if requested != "" {
		targets = []string{requested}
	}
	manager := process.DefaultManager()
	results := make([]process.SlotStatus, 0, len(targets))
	failed := false
	for _, name := range targets {
		slot, err := config.Load(name)
		if err != nil {
			failed = true
			reportOperationMessage(stderr, "failed to load slot %q: %s", name, err)
			continue
		}
		result, err := manager.Status(slot)
		if err != nil {
			failed = true
			reportOperationMessage(stderr, "failed to inspect slot %q: %s", name, err)
			continue
		}
		results = append(results, result)
	}
	if opts.jsonOutput {
		if writeJSON(stdout, jsonStatusResult{Results: results}) != 0 {
			return exitOperationError
		}
	} else {
		for _, result := range results {
			if requested == "" {
				fmt.Fprintf(stdout, "%s\t%s", result.Slot, result.State)
			} else {
				fmt.Fprintf(stdout, "Slot: %s\nState: %s", result.Slot, result.State)
			}
			if result.PID != 0 {
				fmt.Fprintf(stdout, "\tPID: %d", result.PID)
			}
			fmt.Fprintln(stdout)
		}
	}
	if failed {
		return exitOperationError
	}
	return 0
}

func stop(stdout, stderr io.Writer, name string, opts globalOptions) int {
	slot, err := config.Load(name)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	manager := process.DefaultManager()
	if opts.timeoutSet {
		manager.StopTimeout = opts.stopTimeout
	}
	if opts.force {
		err = manager.ForceStop(slot)
	} else {
		err = manager.Stop(slot)
	}
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if opts.jsonOutput {
		return writeJSON(stdout, map[string]string{"slot": name, "result": "stopped"})
	}
	fmt.Fprintf(stdout, "Stopped %s\n", name)
	return 0
}

func restart(stdout, stderr io.Writer, name string, opts globalOptions) int {
	slot, err := config.Load(name)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	state, err := process.DefaultManager().Restart(slot)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if opts.jsonOutput {
		return writeJSON(stdout, jsonStartResult{State: state})
	}
	fmt.Fprintf(stdout, "Restarted %s (PID %d)\n", state.Slot, state.PID)
	return 0
}

func logs(stdout, stderr io.Writer, name string, opts globalOptions, count int) int {
	slot, err := config.Load(name)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if slot.LogFile == "" {
		return reportOperationError(stderr, fmt.Errorf("slot %q has no log_file", name))
	}
	lines, err := logview.LastLines(slot.LogFile, count)
	if err != nil {
		return reportOperationError(stderr, err)
	}
	if opts.jsonOutput {
		return writeJSON(stdout, map[string]any{"slot": name, "lines": lines})
	}
	for _, line := range lines {
		fmt.Fprintln(stdout, line)
	}
	if opts.follow {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if err := logview.Follow(ctx, slot.LogFile, stdout); err != nil {
			return reportOperationError(stderr, err)
		}
	}
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
	return listText(stdout, stderr, opts.shouldColor(stdout))
}

// listText prints the human-readable table and writes per-slot load
// failures to stderr.
func listText(stdout, stderr io.Writer, useColor bool) int {
	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "SLOT\tNAME\tENABLED\tSTATUS")
	failed := false
	for _, name := range config.AllowedSlots {
		slot, err := config.Load(name)
		if err != nil {
			failed = true
			fmt.Fprintf(w, "%s\t-\t-\t%s\n", name, term.Red(useColor, "error"))
			reportOperationMessage(stderr, "failed to load slot %q: %s", name, err.Error())
			continue
		}
		enabled, status := "no", "disabled"
		if slot.Enabled {
			enabled, status = "yes", "stopped"
		}
		enabled = term.Bold(useColor, enabled)
		status = paintStatus(useColor, status)
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
	return validateText(stdout, targets, opts.shouldColor(stdout))
}

func validateText(stdout io.Writer, targets []string, useColor bool) int {
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
		fmt.Fprintf(w, "%s\t%s\n", name, paintValidateResult(useColor, result, len(validationErrors) > 0))
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

// paintStatus colours the STATUS column of `list` based on the slot's
// lifecycle state. "stopped" gets green, "disabled" gets dim grey, and
// "error" is red. The mapping is intentionally narrow — only the words
// that already appear in the column are coloured.
func paintStatus(useColor bool, status string) string {
	if !useColor {
		return status
	}
	switch status {
	case "stopped":
		return term.Green(true, status)
	case "disabled":
		return term.Dim(true, status)
	default:
		return status
	}
}

// paintValidateResult colours the RESULT column of `validate`. A clean
// "OK" is green; "disabled" is dim; anything else (a validation error
// description) is printed in red. Errors always render the actual error
// text — colour is a layer on top, never a replacement.
func paintValidateResult(useColor bool, result string, hasErrors bool) string {
	if !useColor {
		return result
	}
	if hasErrors {
		return term.Red(true, result)
	}
	switch result {
	case "OK":
		return term.Green(true, result)
	case "disabled":
		return term.Dim(true, result)
	default:
		return result
	}
}
