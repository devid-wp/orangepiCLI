"""Loading and representing slot configuration files."""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

ALLOWED_SLOTS = tuple(f"slot{number}" for number in range(1, 11))
PROJECT_ROOT = Path(__file__).resolve().parent.parent


class ConfigError(ValueError):
    """A slot configuration cannot be loaded."""


@dataclass(frozen=True)
class SlotConfig:
    slot: str
    enabled: bool = False
    display_name: str = ""
    description: str = ""
    working_directory: str = ""
    start_command: str = ""
    stop_command: str = ""
    restart_command: str = ""
    log_file: str = ""
    use_sudo: bool = False
    auto_restart: bool = False
    environment: dict[str, str] = field(default_factory=dict)
    env_file: str = ""


def config_directory() -> Path:
    """Return config directory, optionally overridden for deployments/tests."""
    return Path(os.environ.get("ORANGECTL_CONFIG_DIR", PROJECT_ROOT / "configs")).expanduser()


def require_allowed_slot(slot: str) -> None:
    if slot not in ALLOWED_SLOTS:
        available = ", ".join(ALLOWED_SLOTS)
        raise ConfigError(f"slot {slot!r} does not exist; available slots: {available}")


def config_path(slot: str) -> Path:
    require_allowed_slot(slot)
    return config_directory() / f"{slot}.json"


def load_slot(slot: str) -> SlotConfig:
    path = config_path(slot)
    if not path.is_file():
        raise ConfigError(f"configuration file is missing: {path}")
    try:
        raw: Any = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as error:
        raise ConfigError(f"invalid JSON in {path.name}: {error.msg}") from error
    except OSError as error:
        raise ConfigError(f"cannot read {path}: {error}") from error

    if not isinstance(raw, dict):
        raise ConfigError(f"{path.name} must contain a JSON object")
    if raw.get("slot") != slot:
        raise ConfigError(f"field 'slot' in {path.name} must be {slot!r}")

    try:
        return SlotConfig(**raw)
    except TypeError as error:
        raise ConfigError(f"invalid fields in {path.name}: {error}") from error
