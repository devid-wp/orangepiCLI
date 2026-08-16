//go:build !linux

package process

import "fmt"

type unsupportedProcReader struct{}

// NewProcReader returns a reader that reports /proc as unavailable on
// platforms that do not provide Linux procfs.
func NewProcReader() ProcReader { return unsupportedProcReader{} }

func (unsupportedProcReader) ReadProcess(pid int) (ProcProcess, error) {
	return ProcProcess{}, fmt.Errorf("read process %d: /proc is unavailable on this platform", pid)
}
