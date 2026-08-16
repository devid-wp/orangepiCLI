package process

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var environmentKeyPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// EnvironmentSource provides the system environment and secure env_file
// reading. It is an interface so merge rules can be tested without touching
// the host environment or filesystem.
type EnvironmentSource interface {
	Environ() []string
	ReadEnvFile(path string) ([]byte, error)
}

type osEnvironmentSource struct{}

func (osEnvironmentSource) Environ() []string { return os.Environ() }

func (osEnvironmentSource) ReadEnvFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("env_file must be a regular file")
	}
	return os.ReadFile(path)
}

// BuildEnvironment starts with os.Environ(), then overlays values in envFile,
// then overlays explicit environment values. This order lets a slot replace
// inherited values without exposing secret values in an error.
func BuildEnvironment(envFile string, environment map[string]string) ([]string, error) {
	return BuildEnvironmentWith(osEnvironmentSource{}, envFile, environment)
}

// BuildEnvironmentWith is BuildEnvironment with injectable system inputs.
func BuildEnvironmentWith(source EnvironmentSource, envFile string, environment map[string]string) ([]string, error) {
	values := make(map[string]string)
	for _, item := range source.Environ() {
		key, value, found := strings.Cut(item, "=")
		if found && environmentKeyPattern.MatchString(key) {
			values[key] = value
		}
	}

	if envFile != "" {
		contents, err := source.ReadEnvFile(envFile)
		if err != nil {
			// Do not include file data or environment values in this error.
			return nil, fmt.Errorf("read env_file: %w", err)
		}
		fileValues, err := parseEnvFile(contents)
		if err != nil {
			return nil, err
		}
		for key, value := range fileValues {
			values[key] = value
		}
	}

	for key, value := range environment {
		if !environmentKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("environment contains an invalid key")
		}
		values[key] = value
	}
	return flattenEnvironment(values), nil
}

func parseEnvFile(contents []byte) (map[string]string, error) {
	values := make(map[string]string)
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSuffix(line, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found || !environmentKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("env_file contains an invalid entry")
		}
		values[key] = value
	}
	return values, nil
}

func flattenEnvironment(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+values[key])
	}
	return result
}
