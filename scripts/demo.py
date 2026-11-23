"""Start with `make up`, then this script submits a small baseline collection job."""
import json, os, time, urllib.error, urllib.request

def get_json(url):
    with urllib.request.urlopen(url,timeout=5) as r:return json.load(r)

def main():
    base=os.environ.get("FLEET_URL","http://127.0.0.1:8080");artifact_dir=os.environ.get("FLEET_ARTIFACTS","./artifacts");data={"idempotency_key":f"demo-{int(time.time())}","type":"COLLECT_EPISODES","priority":1,"max_retries":2,"config":{"environment_id":"Reacher-v5","episodes":2,"seed":42,"artifact_dir":artifact_dir,"dataset_id":"demo-dataset","dataset_version":f"v{int(time.time())}"}};req=urllib.request.Request(base+"/v1/jobs",data=json.dumps(data).encode(),headers={"Content-Type":"application/json"});
    with urllib.request.urlopen(req,timeout=10) as r:job=json.load(r)
    status_url=base+"/v1/jobs/"+job["job_id"];deadline=time.monotonic()+300
    while time.monotonic()<deadline:
        current=get_json(status_url)
        if current["status"] in {"SUCCEEDED","FAILED","CANCELLED"}:
            print(json.dumps(current,indent=2))
            if current["status"]!="SUCCEEDED":raise SystemExit(1)
            return
        time.sleep(1)
    raise TimeoutError(f"demo job did not finish within 300 seconds: {status_url}")
if __name__=="__main__":main()
