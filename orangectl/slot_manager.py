"""High-level slot operations (extended in process-management stages)."""

from .config_loader import ALLOWED_SLOTS, SlotConfig, load_slot


def all_slots() -> list[SlotConfig]:
    return [load_slot(slot) for slot in ALLOWED_SLOTS]
