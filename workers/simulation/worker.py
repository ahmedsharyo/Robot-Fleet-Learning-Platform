"""Local polling worker for collection jobs; MuJoCo is an optional runtime dependency."""
from __future__ import annotations
import argparse, hashlib, json, logging, os, platform, random, socket, time, uuid
from pathlib import Path
from datasets.schema.episode import Episode
from datasets.storage.parquet import publish_dataset
from workers.common.client import FleetClient

logging.basicConfig(level=os.environ.get("LOG_LEVEL","INFO"),format='{"level":"%(levelname)s","message":"%(message)s"}')
def collect(config: dict, worker_id: str) -> dict:
    try: import gymnasium as gym
    except ImportError as exc: raise RuntimeError("simulation requires Gymnasium and MuJoCo; run make setup-python") from exc
    env_id=config.get("environment_id","Reacher-v5"); episodes=[]; root=Path(config.get("artifact_dir","./artifacts")); seed=int(config.get("seed",1)); count=int(config.get("episodes",1)); horizon=int(config.get("max_steps",200)); cfg_hash=hashlib.sha256(json.dumps(config,sort_keys=True).encode()).hexdigest()
    for idx in range(count):
        env=gym.make(env_id); obs,info=env.reset(seed=seed+idx); observations=[obs.tolist()]; actions=[]; rewards=[]; terminated=[]; truncated=[]; total=0.0
        for _ in range(horizon):
            # Explicitly a baseline, never mislabeled as an expert.
            action=env.action_space.sample(); nxt,reward,term,trunc,info=env.step(action); actions.append(action.tolist());observations.append(nxt.tolist());rewards.append(float(reward));terminated.append(bool(term));truncated.append(bool(trunc));total+=float(reward)
            if term or trunc: break
        if not terminated[-1] and not truncated[-1]: truncated[-1]=True
        episodes.append(Episode.create(episode_id=f"{worker_id}-{uuid.uuid4().hex}",environment_id=env_id,environment_version=getattr(gym,"__version__","unknown"),observations=observations,actions=actions,rewards=rewards,terminated=terminated,truncated=truncated,episode_return=total,episode_length=len(actions),seed=seed+idx,policy_id="random-baseline",worker_id=worker_id,config_hash=cfg_hash));env.close()
    path=publish_dataset(root,config.get("dataset_id",f"dataset-{uuid.uuid4().hex[:10]}"),config.get("dataset_version","v1"),episodes,{"job_id":config.get("job_id"),"controller":"random-baseline","worker_id":worker_id})
    return {"dataset_path":str(path),"episode_count":len(episodes),"label":"baseline"}
def main() -> None:
    ap=argparse.ArgumentParser();ap.add_argument("--worker-id",default=f"{socket.gethostname()}-{os.getpid()}");ap.add_argument("--once",action="store_true");args=ap.parse_args();client=FleetClient();caps=["COLLECT_EPISODES","VALIDATE_DATASET","TRAIN_BC","TRAIN_PPO","EVALUATE_POLICY","RUN_BENCHMARK"];client.register(args.worker_id,caps);logging.info("worker registered: %s",args.worker_id)
    while True:
        try:
            client.heartbeat(args.worker_id);job=client.call("/v1/workers/request",{"worker_id":args.worker_id,"capabilities":caps})
            if not job: time.sleep(1);continue
            lease={"job_id":job["job_id"],"worker_id":args.worker_id,"lease_token":job["lease_token"]};client.call("/v1/jobs/start",lease)
            try:
                result=collect(job.get("config",{}),args.worker_id) if job["type"]=="COLLECT_EPISODES" else {"accepted":True,"job_type":job["type"],"note":"handler not installed"}
                arts=[]
                if result.get("dataset_path"):
                    manifest=Path(result["dataset_path"])/"manifest.json";digest=hashlib.sha256(manifest.read_bytes()).hexdigest();arts=[{"name":"dataset_manifest","uri":str(manifest),"sha256":digest}]
                client.call("/v1/jobs/complete",{**lease,"artifacts":arts,"result":result})
            except Exception as exc:
                logging.exception("job failed");client.call("/v1/jobs/fail",{**lease,"error":str(exc)})
            if args.once:return
        except (OSError,RuntimeError) as exc: logging.warning("worker loop: %s",exc);time.sleep(2)
if __name__=="__main__":main()
