"""Repeatable policy evaluation exporting JSON and CSV."""
from __future__ import annotations
import argparse, csv, json, statistics, time
from pathlib import Path
import numpy as np

def evaluate(policy, env_id: str, seeds=(101, 202, 303), max_steps=1000):
    import gymnasium as gym
    rows=[]
    for seed in seeds:
        env=gym.make(env_id);obs,_=env.reset(seed=seed);total=0.;steps=0;lat=[];term=trunc=False
        while steps<max_steps and not(term or trunc):
            t=time.perf_counter();action=policy(obs);lat.append(time.perf_counter()-t);obs,r,term,trunc,_=env.step(action);total+=float(r);steps+=1
        rows.append({"seed":seed,"return":total,"length":steps,"terminated":term,"truncated":trunc,"inference_latency_mean_ms":1000*statistics.mean(lat) if lat else 0.,"throughput_steps_s":steps/max(sum(lat),1e-9)});env.close()
    returns=[x["return"] for x in rows]
    return {"environment_id":env_id,"seeds":list(seeds),"episodes":rows,"summary":{"mean_return":statistics.mean(returns),"std_return":statistics.stdev(returns) if len(returns)>1 else 0.,"mean_length":statistics.mean(x["length"] for x in rows),"success_rate":None}}
def write_report(report, output):
    path=Path(output);path.parent.mkdir(parents=True,exist_ok=True);path.write_text(json.dumps(report,indent=2));csv_path=path.with_suffix(".csv")
    with csv_path.open("w",newline="") as f:w=csv.DictWriter(f,fieldnames=report["episodes"][0].keys());w.writeheader();w.writerows(report["episodes"])
if __name__=="__main__":
    p=argparse.ArgumentParser();p.add_argument("--env",default="Reacher-v5");p.add_argument("--output",default="./reports/evaluation.json");a=p.parse_args()
    # A random baseline is clearly identified. Learned checkpoint loading can pass a callable to evaluate().
    import gymnasium as gym
    space=gym.make(a.env).action_space;report=evaluate(lambda obs:space.sample(),a.env,seeds=(101,202,303));report["policy_id"]="random-baseline";write_report(report,a.output)
