"""Typer command-line interface."""

from __future__ import annotations

from typing import Optional

import typer
from rich.console import Console
from rich.table import Table

from .config_loader import ALLOWED_SLOTS, ConfigError, load_slot, require_allowed_slot
from .validators import validate_slot

app = typer.Typer(help="Manage ten universal OrangeCTL slots.", no_args_is_help=True)
console = Console()


@app.command("list")
def list_slots() -> None:
    """Show all configured slots."""
    table = Table("SLOT", "NAME", "ENABLED", "STATUS")
    failed = False
    for slot in ALLOWED_SLOTS:
        try:
            config = load_slot(slot)
            enabled = "yes" if config.enabled else "no"
            status = "stopped" if config.enabled else "disabled"
            table.add_row(slot, config.display_name, enabled, status)
        except ConfigError:
            failed = True
            table.add_row(slot, "-", "-", "config error", style="red")
    console.print(table)
    if failed:
        raise typer.Exit(1)


@app.command()
def validate(slot: Optional[str] = typer.Argument(None, help="slot1 through slot10")) -> None:
    """Validate one slot or every slot configuration."""
    targets = (slot,) if slot else ALLOWED_SLOTS
    if slot:
        try:
            require_allowed_slot(slot)
        except ConfigError as error:
            console.print(f"[red]Error:[/red] {error}")
            raise typer.Exit(2) from error

    failed = False
    table = Table("SLOT", "RESULT")
    for target in targets:
        config, errors = validate_slot(target)
        if errors:
            failed = True
            table.add_row(target, "; ".join(errors), style="red")
        elif config is not None and not config.enabled:
            table.add_row(target, "disabled", style="yellow")
        else:
            table.add_row(target, "OK", style="green")
    console.print(table)
    if failed:
        raise typer.Exit(1)
