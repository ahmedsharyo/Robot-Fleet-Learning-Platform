"""Small JSON control-plane client; trajectory artifacts stay on shared storage."""
from __future__ import annotations
import json, os, time, urllib.error, urllib.request
from typing import Any

class FleetClient:
    def __init__(self, base_url: str | None = None, timeout: float = 5.0):
        self.base_url=(base_url or os.environ.get("FLEET_URL","http://127.0.0.1:8080")).rstrip("/")
        self.timeout=timeout
    def call(self,path: str,payload: dict | None=None) -> Any:
        data=None if payload is None else json.dumps(payload).encode()
        req=urllib.request.Request(self.base_url+path,data=data,headers={"Content-Type":"application/json"},method="GET" if data is None else "POST")
        try:
            with urllib.request.urlopen(req,timeout=self.timeout) as r:
                if r.status==204:return None
                return json.loads(r.read())
        except urllib.error.HTTPError as exc:
            body=exc.read().decode(errors="replace")
            if exc.code==409: raise RuntimeError(f"conflict: {body}") from exc
            raise RuntimeError(f"control plane HTTP {exc.code}: {body}") from exc
    def register(self,worker_id: str,capabilities: list[str]) -> None:
        self.call("/v1/workers/register",{"worker_id":worker_id,"capabilities":capabilities,"cpu_millis":1000,"memory_bytes":0})
    def heartbeat(self,worker_id: str) -> None:self.call("/v1/workers/heartbeat",{"worker_id":worker_id})
