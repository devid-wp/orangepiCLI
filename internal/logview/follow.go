package logview

import (
	"context"
	"io"
	"os"
	"time"
)

// Follow writes bytes appended to path until ctx is cancelled. It reopens the
// file when rotation changes its identity and rewinds when truncation shrinks it.
func Follow(ctx context.Context, path string, out io.Writer) error {
	file, err := Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	offset := info.Size()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			current, err := os.Stat(path)
			if err != nil {
				continue
			}
			if !os.SameFile(info, current) {
				file.Close()
				file, err = Open(path)
				if err != nil {
					continue
				}
				info, _ = file.Stat()
				offset = 0
			}
			if current.Size() < offset {
				offset = 0
			}
			if current.Size() > offset {
				section := io.NewSectionReader(file, offset, current.Size()-offset)
				if _, err := io.Copy(out, section); err != nil {
					return err
				}
				offset = current.Size()
			}
			info = current
		}
	}
}
