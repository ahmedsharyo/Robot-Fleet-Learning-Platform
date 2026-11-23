"""Episode-level split behavior cloning trainer with atomic checkpoints."""
from __future__ import annotations
import argparse, hashlib, json, os, random, tempfile
from pathlib import Path
import numpy as np
import torch
from torch import nn
from learning.models.mlp import mlp
from datasets.storage.parquet import read_episodes

def load_data(path:Path):
    obs=[];acts=[]
    for batch in read_episodes(path):
        for ep in batch.to_pylist(): obs.extend(ep["observations"][:-1]);acts.extend(ep["actions"])
    return np.asarray(obs,dtype=np.float32),np.asarray(acts,dtype=np.float32)
def train(dataset:str,output:str,epochs:int=100,seed:int=1,hidden=(128,128)):
    random.seed(seed);np.random.seed(seed);torch.manual_seed(seed);X,Y=load_data(Path(dataset));episodes=[]
    # Re-load episode boundaries to prevent timestep leakage in the split.
    for batch in read_episodes(dataset): episodes.extend(batch.to_pylist())
    order=np.random.permutation(len(episodes));cut=max(1,int(.8*len(order)));train_ids=set(order[:cut]);trX=[];trY=[];vaX=[];vaY=[]
    for i,ep in enumerate(episodes):
        (trX if i in train_ids else vaX).extend(ep["observations"][:-1]);(trY if i in train_ids else vaY).extend(ep["actions"])
    if not vaX: vaX,vaY=trX[-1:],trY[-1:]
    tx=torch.tensor(trX);ty=torch.tensor(trY);vx=torch.tensor(vaX);vy=torch.tensor(vaY);mean=tx.mean(0);std=tx.std(0).clamp_min(1e-6);model=mlp([tx.shape[1],*hidden,ty.shape[1]]);opt=torch.optim.Adam(model.parameters(),lr=3e-4);best=float("inf");patience=12;stale=0;history=[]
    for epoch in range(epochs):
        model.train();perm=torch.randperm(len(tx));
        for ix in perm.split(256):
            pred=model((tx[ix]-mean)/std);loss=nn.functional.mse_loss(pred,ty[ix]);opt.zero_grad();loss.backward();nn.utils.clip_grad_norm_(model.parameters(),1.0);opt.step()
        model.eval()
        with torch.no_grad(): val=float(nn.functional.mse_loss(model((vx-mean)/std),vy))
        history.append({"epoch":epoch+1,"validation_mse":val})
        if val<best:best=val;stale=0;state={"model":model.state_dict(),"optimizer":opt.state_dict(),"progress":{"epoch":epoch+1,"best_validation_mse":best},"normalization":{"mean":mean,"std":std},"rng":{"torch":torch.get_rng_state(),"numpy":np.random.get_state(),"python":random.getstate()}}
        else:stale+=1
        if stale>=patience:break
    path=Path(output);path.parent.mkdir(parents=True,exist_ok=True);fd,tmp=tempfile.mkstemp(dir=path.parent,prefix=".checkpoint-");os.close(fd)
    try:torch.save(state,tmp);os.replace(tmp,path)
    finally:
        if os.path.exists(tmp):os.unlink(tmp)
    digest=hashlib.sha256(path.read_bytes()).hexdigest();meta={"algorithm":"BC","dataset_path":str(dataset),"seed":seed,"device":"cpu","model":"MLP","torch_version":torch.__version__,"checkpoint_sha256":digest,"metrics":history};path.with_suffix(path.suffix+".json").write_text(json.dumps(meta,indent=2));return meta
if __name__=="__main__":
    ap=argparse.ArgumentParser();ap.add_argument("dataset");ap.add_argument("output");ap.add_argument("--epochs",type=int,default=100);ap.add_argument("--seed",type=int,default=1);a=ap.parse_args();print(json.dumps(train(a.dataset,a.output,a.epochs,a.seed),indent=2))
