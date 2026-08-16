# OrangeCTL

OrangeCTL is a Go CLI for managing up to ten user-configured processes on an
Orange Pi. The current implementation provides the `init`, `list`, `validate`,
and `version` commands. It has no runtime dependencies and builds into a single
executable.

## Requirements

- Go 1.22 or newer for building
- Linux on the target Orange Pi

## Build and test

```bash
go test ./...
go build -o orangectl ./cmd/orangectl
```

Install for the current user:

```bash
mkdir -p ~/.local/bin
go build -o ~/.local/bin/orangectl ./cmd/orangectl
```

Cross-compile from another machine:

```bash
# 64-bit Orange Pi OS
GOOS=linux GOARCH=arm64 go build -o dist/orangectl-linux-arm64 ./cmd/orangectl

# 32-bit ARMv7 Orange Pi OS
GOOS=linux GOARCH=arm GOARM=7 go build -o dist/orangectl-linux-armv7 ./cmd/orangectl
```

## Use

Initialize the application and create ten empty configuration slots:

```bash
orangectl init
```

By default, OrangeCTL follows the XDG directory layout:

- configurations: `$XDG_CONFIG_HOME/orangectl` or `~/.config/orangectl`;
- state: `$XDG_STATE_HOME/orangectl` or `~/.local/state/orangectl`;
- process state: `<state directory>/pids`;
- CLI logs: `<state directory>/logs`.

For local development, point OrangeCTL at the directories in this repository:

```bash
export ORANGECTL_CONFIG_DIR=./configs
export ORANGECTL_STATE_DIR=./state
orangectl list
orangectl validate
orangectl validate slot1
orangectl version
```

Explicit `ORANGECTL_CONFIG_DIR` and `ORANGECTL_STATE_DIR` values take precedence
over XDG variables. Running `orangectl init` is idempotent: existing slot files
are kept unchanged.

Edit one of the generated slot files, set `enabled` to `true`, and provide its
working directory and start command.

`orangectl version` reports the version, source commit, and build date. Embed
release metadata during a build with:

```bash
go build -ldflags "-X github.com/devid-wp/orangepiCLI/internal/buildinfo.Version=v1.0.0 -X github.com/devid-wp/orangepiCLI/internal/buildinfo.Commit=$(git rev-parse --short HEAD) -X github.com/devid-wp/orangepiCLI/internal/buildinfo.BuildDate=$(git show -s --format=%cI HEAD)" -o orangectl ./cmd/orangectl
```
