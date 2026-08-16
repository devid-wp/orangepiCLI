package process

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
)

// OpenLog opens path for append without accepting a directory or symbolic
// link. A missing log is created with mode 0600, and an existing regular file
// is restricted to that mode before it is handed to a child process.
func OpenLog(path string) (*os.File, error) {
	if path == "" {
		return nil, fmt.Errorf("log_file is required")
	}
	info, err := os.Lstat(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create log_file: %w", err)
		}
		return file, nil
	case err != nil:
		return nil, fmt.Errorf("inspect log_file: %w", err)
	case info.Mode()&os.ModeSymlink != 0:
		return nil, fmt.Errorf("log_file must not be a symbolic link")
	case !info.Mode().IsRegular():
		return nil, fmt.Errorf("log_file must be a regular file")
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log_file: %w", err)
	}
	if err := file.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		_ = file.Close()
		return nil, fmt.Errorf("secure log_file: %w", err)
	}
	return file, nil
}

// AttachLogs opens a secure append-only log and routes both child streams to
// it. The caller closes the returned closer after the child has exited.
func AttachLogs(command *Command, path string) (io.Closer, error) {
	if command == nil {
		return nil, fmt.Errorf("attach log_file: command is nil")
	}
	file, err := OpenLog(path)
	if err != nil {
		return nil, err
	}
	command.Stdout = file
	command.Stderr = file
	return file, nil
}
