package term

import (
	"bytes"
	"testing"
)

func TestShouldColorJSONOutputAlwaysFalse(t *testing.T) {
	// Even with ForceTTY and clean env, --json must not emit colour.
	opts := Options{JSONOutput: true, ForceTTY: true, Getenv: func(string) string { return "" }}
	if ShouldColor(&bytes.Buffer{}, opts) {
		t.Fatal("ShouldColor() = true, want false for JSON output")
	}
}

func TestShouldColorNoColorExplicit(t *testing.T) {
	opts := Options{NoColorFlag: true, ForceTTY: true, Getenv: func(string) string { return "" }}
	if ShouldColor(&bytes.Buffer{}, opts) {
		t.Fatal("ShouldColor() = true, want false when --no-color is set")
	}
}

func TestShouldColorEnvNO_COLOR(t *testing.T) {
	env := map[string]string{"NO_COLOR": "1"}
	get := func(key string) string { return env[key] }

	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "set to non-empty", value: "1", want: false},
		{name: "set to whitespace", value: " ", want: false},
		{name: "unset", value: "", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env["NO_COLOR"] = tc.value
			opts := Options{ForceTTY: true, Getenv: get}
			if got := ShouldColor(&bytes.Buffer{}, opts); got != tc.want {
				t.Fatalf("ShouldColor() = %v, want %v (NO_COLOR=%q)", got, tc.want, tc.value)
			}
		})
	}
}

func TestShouldColorTERMDumb(t *testing.T) {
	get := func(key string) string {
		if key == "TERM" {
			return "dumb"
		}
		return ""
	}
	opts := Options{ForceTTY: true, Getenv: get}
	if ShouldColor(&bytes.Buffer{}, opts) {
		t.Fatal("ShouldColor() = true, want false when TERM=dumb")
	}
}

func TestShouldColorNonTTY(t *testing.T) {
	// bytes.Buffer has no Fd(), so it is reported as non-terminal.
	opts := Options{Getenv: func(string) string { return "" }}
	if ShouldColor(&bytes.Buffer{}, opts) {
		t.Fatal("ShouldColor() = true for non-TTY writer")
	}
}

func TestShouldColorTTYDefault(t *testing.T) {
	opts := Options{ForceTTY: true, Getenv: func(string) string { return "" }}
	if !ShouldColor(&bytes.Buffer{}, opts) {
		t.Fatal("ShouldColor() = false, want true when ForceTTY is on and env is clean")
	}
}

func TestPaintDisabledReturnsPlainText(t *testing.T) {
	if got := Paint(false, "\033[31m", "red"); got != "red" {
		t.Fatalf("Paint(false, ...) = %q, want %q", got, "red")
	}
}

func TestPaintEnabledWrapsInEscape(t *testing.T) {
	got := Paint(true, "\033[31m", "red")
	want := "\033[31mred\033[0m"
	if got != want {
		t.Fatalf("Paint(true, ...) = %q, want %q", got, want)
	}
}

func TestPaintEmptyTextBypassesWrapping(t *testing.T) {
	if got := Paint(true, "\033[31m", ""); got != "" {
		t.Fatalf("Paint(true, ..., \"\") = %q, want empty string", got)
	}
}
