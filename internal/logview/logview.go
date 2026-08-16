// Package logview safely reads configured process log files.
package logview

import (
	"bytes"
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
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat log file: %w", err)
	}
	const blockSize int64 = 32 * 1024
	var data []byte
	offset := info.Size()
	breaks := 0
	for offset > 0 && breaks <= count {
		size := blockSize
		if offset < size {
			size = offset
		}
		offset -= size
		block := make([]byte, size)
		if _, err := file.ReadAt(block, offset); err != nil {
			return nil, fmt.Errorf("read log file: %w", err)
		}
		breaks += bytes.Count(block, []byte{'\n'})
		data = append(block, data...)
	}
	parts := bytes.Split(data, []byte{'\n'})
	if len(parts) > 0 && len(parts[len(parts)-1]) == 0 {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > count {
		parts = parts[len(parts)-count:]
	}
	lines := make([]string, len(parts))
	for i := range parts {
		lines[i] = string(parts[i])
	}
	return lines, nil
}
