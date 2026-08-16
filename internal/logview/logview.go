// Package logview safely reads configured process log files.
package logview

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

var ErrUnsafeLogFile = errors.New("unsafe log file")

// Open accepts only a regular file, never a directory or symbolic link.
func Open(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect log file: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: log must be a regular non-symlink file", ErrUnsafeLogFile)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	return file, nil
}

// LastLines returns the requested suffix of a log. The initial implementation
// favours correctness; large-file reverse block reading is added separately.
func LastLines(path string, count int) ([]string, error) {
	if count <= 0 {
		return nil, fmt.Errorf("line count must be positive")
	}
	file, err := Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	lines := make([]string, 0, count)
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 64*1024)
	scanner.Buffer(buffer, 1024*1024)
	for scanner.Scan() {
		if len(lines) == count {
			copy(lines, lines[1:])
			lines = lines[:count-1]
		}
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read log file: %w", err)
	}
	return lines, nil
}
