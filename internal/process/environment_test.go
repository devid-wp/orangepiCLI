package process

import (
	"errors"
	"strings"
	"testing"
)

type fakeEnvironmentSource struct {
	environment []string
	contents    []byte
	err         error
}

func (source fakeEnvironmentSource) Environ() []string { return source.environment }

func (source fakeEnvironmentSource) ReadEnvFile(string) ([]byte, error) {
	return source.contents, source.err
}

func TestBuildEnvironmentAppliesSourcesInOrder(t *testing.T) {
	source := fakeEnvironmentSource{
		environment: []string{"PATH=/usr/bin", "SHARED=host", "HOST_ONLY=yes"},
		contents:    []byte("# slot variables\nSHARED=file\nFILE_ONLY=present\n"),
	}
	got, err := BuildEnvironmentWith(source, "slot.env", map[string]string{
		"SHARED":      "config",
		"CONFIG_ONLY": "present",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"CONFIG_ONLY=present",
		"FILE_ONLY=present",
		"HOST_ONLY=yes",
		"PATH=/usr/bin",
		"SHARED=config",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("BuildEnvironmentWith() = %v, want %v", got, want)
	}
}

func TestBuildEnvironmentAcceptsEmptyValueAndNoEnvFile(t *testing.T) {
	source := fakeEnvironmentSource{environment: []string{"KEEP=one"}}
	got, err := BuildEnvironmentWith(source, "", map[string]string{"EMPTY": ""})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "EMPTY=,KEEP=one" {
		t.Fatalf("BuildEnvironmentWith() = %v", got)
	}
}

func TestBuildEnvironmentDoesNotExposeSecretsInErrors(t *testing.T) {
	const secret = "do-not-leak-this-value"
	tests := []struct {
		name        string
		source      fakeEnvironmentSource
		environment map[string]string
	}{
		{
			name:        "read failure",
			source:      fakeEnvironmentSource{err: errors.New("cannot read env file")},
			environment: map[string]string{"TOKEN": secret},
		},
		{
			name:        "invalid file entry",
			source:      fakeEnvironmentSource{contents: []byte("TOKEN=" + secret + "\nnot valid\n")},
			environment: map[string]string{},
		},
		{
			name:        "invalid config key",
			source:      fakeEnvironmentSource{},
			environment: map[string]string{"INVALID=" + secret: secret},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := BuildEnvironmentWith(test.source, "slot.env", test.environment)
			if err == nil {
				t.Fatal("BuildEnvironmentWith() returned no error")
			}
			if strings.Contains(err.Error(), secret) {
				t.Fatalf("error exposed secret: %q", err)
			}
		})
	}
}

func TestBuildEnvironmentRejectsInvalidEnvFileEntry(t *testing.T) {
	_, err := BuildEnvironmentWith(fakeEnvironmentSource{contents: []byte("INVALID KEY=value\n")}, "slot.env", nil)
	if err == nil || err.Error() != "env_file contains an invalid entry" {
		t.Fatalf("BuildEnvironmentWith() error = %v", err)
	}
}
