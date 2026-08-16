//go:build linux

package process

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const linuxClockTicksPerSecond = 100

// LinuxProcReader reads identity facts from a Linux procfs mount. Root is
// configurable for tests; an empty Root means /proc.
type LinuxProcReader struct {
	Root string
}

// NewProcReader returns the Linux /proc implementation on Linux hosts.
func NewProcReader() ProcReader { return LinuxProcReader{} }

func (reader LinuxProcReader) ReadProcess(pid int) (ProcProcess, error) {
	if pid <= 0 {
		return ProcProcess{}, fmt.Errorf("%w: invalid PID", ErrProcessNotFound)
	}
	root := reader.Root
	if root == "" {
		root = "/proc"
	}
	directory := filepath.Join(root, strconv.Itoa(pid))
	info, err := os.Stat(directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ProcProcess{}, fmt.Errorf("%w: %d", ErrProcessNotFound, pid)
		}
		return ProcProcess{}, fmt.Errorf("inspect /proc process: %w", err)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return ProcProcess{}, fmt.Errorf("inspect /proc process: unavailable owner")
	}
	startTicks, err := readStartTicks(filepath.Join(directory, "stat"))
	if err != nil {
		return ProcProcess{}, err
	}
	bootTime, err := readBootTime(filepath.Join(root, "stat"))
	if err != nil {
		return ProcProcess{}, err
	}
	command, err := readCommand(filepath.Join(directory, "cmdline"))
	if err != nil {
		return ProcProcess{}, err
	}
	workingDirectory, err := os.Readlink(filepath.Join(directory, "cwd"))
	if err != nil {
		return ProcProcess{}, fmt.Errorf("read process working directory: %w", err)
	}
	wholeSeconds := startTicks / linuxClockTicksPerSecond
	partialTicks := startTicks % linuxClockTicksPerSecond
	startedAt := time.Unix(bootTime, 0).
		Add(time.Duration(wholeSeconds) * time.Second).
		Add(time.Duration(partialTicks) * time.Second / linuxClockTicksPerSecond)
	return ProcProcess{
		PID:              pid,
		StartedAt:        startedAt.UTC(),
		Identity:         "linux-start-ticks:" + strconv.FormatUint(startTicks, 10),
		Command:          command,
		WorkingDirectory: workingDirectory,
		UserID:           strconv.FormatUint(uint64(stat.Uid), 10),
	}, nil
}

func readStartTicks(path string) (uint64, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read process start time: %w", err)
	}
	closeParen := strings.LastIndex(string(contents), ")")
	if closeParen < 0 {
		return 0, fmt.Errorf("read process start time: malformed proc stat")
	}
	fields := strings.Fields(string(contents)[closeParen+1:])
	// field 22 in proc(5) is starttime. fields begins at field 3 (state).
	const startTimeIndex = 19
	if len(fields) <= startTimeIndex {
		return 0, fmt.Errorf("read process start time: malformed proc stat")
	}
	ticks, err := strconv.ParseUint(fields[startTimeIndex], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("read process start time: malformed proc stat")
	}
	return ticks, nil
}

func readBootTime(path string) (int64, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read system boot time: %w", err)
	}
	for _, line := range strings.Split(string(contents), "\n") {
		value, found := strings.CutPrefix(line, "btime ")
		if !found {
			continue
		}
		bootTime, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("read system boot time: malformed proc stat")
		}
		return bootTime, nil
	}
	return 0, fmt.Errorf("read system boot time: btime is missing")
}

func readCommand(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read process command: %w", err)
	}
	parts := strings.Split(strings.TrimSuffix(string(contents), "\x00"), "\x00")
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("read process command: command line is empty")
	}
	return strings.Join(parts, " "), nil
}
