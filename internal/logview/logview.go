// Package logview safely reads configured process log files.
package logview

import (
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
