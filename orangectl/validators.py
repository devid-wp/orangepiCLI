"""Validation rules for user-provided slot configuration."""

from __future__ import annotations

from pathlib import Path

from .config_loader import ConfigError, SlotConfig, load_slot


def validate_config(config: SlotConfig) -> list[str]:
    errors: list[str] = []
    if not isinstance(config.enabled, bool):
        errors.append("enabled must be true or false")
    if not isinstance(config.environment, dict) or not all(
        isinstance(key, str) and isinstance(value, str)
        for key, value in config.environment.items()
    ):
        errors.append("environment must contain string keys and values")

    if not config.enabled:
        return errors
    if not config.working_directory:
        errors.append("missing working_directory")
    elif not Path(config.working_directory).expanduser().is_dir():
        errors.append("invalid working_directory")
    if not config.start_command.strip():
        errors.append("missing start_command")
    if config.log_file:
        log_path = Path(config.log_file).expanduser()
        if not log_path.is_file():
            errors.append("log_file does not exist")
    if config.env_file and not Path(config.env_file).expanduser().is_file():
        errors.append("env_file does not exist")
    return errors


def validate_slot(slot: str) -> tuple[SlotConfig | None, list[str]]:
    try:
        config = load_slot(slot)
    except ConfigError as error:
        return None, [str(error)]
    return config, validate_config(config)
