"""Immutable episode partitions and manifests backed by Parquet."""
from __future__ import annotations
from dataclasses import asdict
from datetime import datetime, timezone
import hashlib, json, os, tempfile
from pathlib import Path
from typing import Iterable, Iterator
from datasets.schema.episode import Episode
from datasets.validation.validate import validate_batch

def _atomic_json(path: Path, payload: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp = tempfile.mkstemp(prefix=".manifest-", dir=path.parent)
    try:
        with os.fdopen(fd, "w") as f: json.dump(payload, f, sort_keys=True, indent=2); f.flush(); os.fsync(f.fileno())
        os.replace(tmp, path)
    except BaseException:
        try: os.unlink(tmp)
        except FileNotFoundError: pass
        raise

def publish_dataset(root: str | Path, dataset_id: str, version: str, episodes: Iterable[Episode], provenance: dict) -> Path:
    try: import pyarrow as pa; import pyarrow.parquet as pq
    except ImportError as exc: raise RuntimeError("Parquet support requires pyarrow; run make setup-python") from exc
    root = Path(root); dest = root / dataset_id / version
    if dest.exists(): raise FileExistsError(f"immutable dataset already exists: {dest}")
    valid, rejects = validate_batch(episodes)
    if rejects: raise ValueError(json.dumps(rejects, sort_keys=True))
    if not valid: raise ValueError("cannot publish an empty dataset")
    dest.parent.mkdir(parents=True, exist_ok=True)
    staging = dest.parent / f".{version}.staging-{os.getpid()}"
    staging.mkdir()
    try:
        refs=[]
        for start in range(0,len(valid),256):
            batch=valid[start:start+256]; table=pa.Table.from_pylist([asdict(x) for x in batch]); name=f"episodes-{start//256:05d}.parquet"; path=staging/name
            pq.write_table(table,path,compression="zstd"); digest=hashlib.sha256(path.read_bytes()).hexdigest(); refs.append({"path":name,"sha256":digest,"episode_count":len(batch)})
        manifest={"dataset_id":dataset_id,"version":version,"schema_version":valid[0].schema_version,"environment_id":valid[0].environment_id,"environment_version":valid[0].environment_version,"episode_count":len(valid),"episodes":refs,"provenance":provenance,"created_at":datetime.now(timezone.utc).isoformat()}
        _atomic_json(staging/"manifest.json",manifest)
        os.rename(staging,dest)
    except BaseException:
        import shutil; shutil.rmtree(staging,ignore_errors=True); raise
    return dest

def read_episodes(dataset_path: str | Path, batch_size: int = 128) -> Iterator[list[dict]]:
    import pyarrow.parquet as pq
    base=Path(dataset_path); manifest=json.loads((base/"manifest.json").read_text())
    for ref in manifest["episodes"]:
        path=base/ref["path"]
        if hashlib.sha256(path.read_bytes()).hexdigest()!=ref["sha256"]: raise ValueError(f"hash mismatch: {path}")
        yield from pq.ParquetFile(path).iter_batches(batch_size=batch_size)
