"""Batch validation with structured per-episode rejection records."""
from __future__ import annotations
from dataclasses import asdict
from typing import Iterable
from datasets.schema.episode import Episode, validate_episode

def validate_batch(episodes: Iterable[Episode]) -> tuple[list[Episode], list[dict[str, str]]]:
    good: list[Episode] = []; errors: list[dict[str, str]] = []; seen: set[str] = set()
    for ep in episodes:
        try:
            validate_episode(ep)
            if ep.episode_id in seen:
                raise ValueError("duplicate episode_id")
            seen.add(ep.episode_id); good.append(ep)
        except (TypeError, ValueError) as exc:
            errors.append({"episode_id": getattr(ep, "episode_id", "<unknown>"), "code": "INVALID_EPISODE", "message": str(exc)})
    return good, errors
