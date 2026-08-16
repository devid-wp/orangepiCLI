// Package sysinfo collects portable system facts through injectable sources.
package sysinfo

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Temperature returns Celsius from a Linux thermal zone, or "unavailable".
func Temperature(source Source) (string, error) {
	data, err := source.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return "unavailable", nil
	}
	milli, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return "unavailable", nil
	}
	return fmt.Sprintf("%.1f°C", milli/1000), nil
}

type Source interface {
	Hostname() (string, error)
	ReadFile(string) ([]byte, error)
	Now() time.Time
}
type osSource struct{}

func (osSource) Hostname() (string, error)            { return os.Hostname() }
func (osSource) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (osSource) Now() time.Time                       { return time.Now() }

type Info struct {
	Hostname      string `json:"hostname"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

func Collect(source Source) (Info, error) {
	host, err := source.Hostname()
	if err != nil {
		return Info{}, fmt.Errorf("read hostname: %w", err)
	}
	data, err := source.ReadFile("/proc/uptime")
	if err != nil {
		return Info{}, fmt.Errorf("read uptime: %w", err)
	}
	var seconds float64
	if _, err := fmt.Sscan(string(data), &seconds); err != nil {
		return Info{}, fmt.Errorf("parse uptime: %w", err)
	}
	return Info{Hostname: host, UptimeSeconds: int64(seconds)}, nil
}
func Default() (Info, error) { return Collect(osSource{}) }
