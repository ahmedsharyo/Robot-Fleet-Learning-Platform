"""Scheduling measurement with a worker returning immediately after each lease."""
from __future__ import annotations
import argparse,json,os,statistics,time,urllib.request,uuid

def post(base,path,data):
    req=urllib.request.Request(base+path,data=json.dumps(data).encode(),headers={"Content-Type":"application/json"})
    with urllib.request.urlopen(req,timeout=10) as r:return json.load(r) if r.status!=204 else None
def main():
    p=argparse.ArgumentParser();p.add_argument("--jobs",type=int,default=30);p.add_argument("--output",default="reports/benchmark.json");a=p.parse_args();base=os.environ.get("FLEET_URL","http://127.0.0.1:8080");wid="benchmark-"+uuid.uuid4().hex[:8];post(base,"/v1/workers/register",{"worker_id":wid,"capabilities":["RUN_BENCHMARK"],"cpu_millis":1000,"memory_bytes":0});submit=[]
    for _ in range(a.jobs):
        start=time.perf_counter();j=post(base,"/v1/jobs",{"idempotency_key":uuid.uuid4().hex,"type":"RUN_BENCHMARK","config":{},"max_retries":0});submit.append(time.perf_counter()-start)
    rows=[]
    for _ in range(a.jobs):
        j=post(base,"/v1/workers/request",{"worker_id":wid,"capabilities":["RUN_BENCHMARK"]});post(base,"/v1/jobs/start",{"job_id":j["job_id"],"worker_id":wid,"lease_token":j["lease_token"]});t=time.perf_counter();post(base,"/v1/jobs/complete",{"job_id":j["job_id"],"worker_id":wid,"lease_token":j["lease_token"],"artifacts":[],"result":{}});rows.append(time.perf_counter()-t)
    result={"measurement":"scheduling_overhead","jobs":a.jobs,"repetitions":1,"submission_latency_s":submit,"completion_ack_latency_s":rows,"mean_submission_latency_s":statistics.mean(submit),"mean_completion_ack_latency_s":statistics.mean(rows),"note":"Single run; repeat at least three times for reportable statistics."};from pathlib import Path;path=Path(a.output);path.parent.mkdir(parents=True,exist_ok=True);path.write_text(json.dumps(result,indent=2));print(json.dumps(result,indent=2))
if __name__=="__main__":main()
