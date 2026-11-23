"""Episode schema shared by collectors, validators, and learning jobs."""
from __future__ import annotations

from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from typing import Any
import hashlib
import json
import numpy as np

SCHEMA_VERSION = "1.0"

@dataclass
class Episode:
    episode_id: str
    environment_id: str
    environment_version: str
    observations: list[list[float]]
    actions: list[list[float]]
    rewards: list[float]
    terminated: list[bool]
    truncated: list[bool]
    episode_return: float
    episode_length: int
    seed: int
    policy_id: str
    worker_id: str
    collected_at: str
    config_hash: str
    schema_version: str = SCHEMA_VERSION

    @classmethod
    def create(cls, **kwargs: Any) -> "Episode":
        kwargs.setdefault("collected_at", datetime.now(timezone.utc).isoformat())
        kwargs.setdefault("schema_version", SCHEMA_VERSION)
        item = cls(**kwargs)
        validate_episode(item)
        return item

    def content_hash(self) -> str:
        data = json.dumps(asdict(self), sort_keys=True, separators=(",", ":"), allow_nan=False)
        return hashlib.sha256(data.encode()).hexdigest()

def validate_episode(ep: Episode) -> None:
    required = (ep.episode_id, ep.environment_id, ep.environment_version, ep.worker_id)
    if not all(required):
        raise ValueError("episode_id, environment_id/version, and worker_id are required")
    n = len(ep.actions)
    if n == 0 or ep.episode_length != n:
        raise ValueError("episode_length must equal non-empty action count")
    if not (len(ep.observations) == n + 1 and len(ep.rewards) == n and len(ep.terminated) == n and len(ep.truncated) == n):
        raise ValueError("alignment requires observations=T+1 and action/reward/end flags=T")
    obs_dim = len(ep.observations[0]); act_dim = len(ep.actions[0])
    if not obs_dim or not act_dim or any(len(x) != obs_dim for x in ep.observations) or any(len(x) != act_dim for x in ep.actions):
        raise ValueError("observation and action dimensions must be non-empty and constant")
    values = [v for row in (*ep.observations, *ep.actions) for v in row] + ep.rewards + [ep.episode_return]
    if not np.isfinite(np.asarray(values, dtype=np.float64)).all():
        raise ValueError("episode contains non-finite numerical values")
    if not (ep.terminated[-1] or ep.truncated[-1]):
        raise ValueError("last transition must be terminated or truncated")
    if any(ep.terminated[i] or ep.truncated[i] for i in range(n - 1)):
        raise ValueError("episode has transitions after an ending flag")
    if not np.isclose(sum(ep.rewards), ep.episode_return, rtol=1e-5, atol=1e-6):
        raise ValueError("episode_return does not match rewards")
