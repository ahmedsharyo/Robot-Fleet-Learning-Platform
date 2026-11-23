"""Compact actor and critic networks for continuous control."""
from __future__ import annotations
from typing import Sequence
try:
    import torch
    from torch import nn
except ImportError as exc:
    raise RuntimeError("learning modules require PyTorch; run make setup-python") from exc

def mlp(sizes: Sequence[int], activation=nn.Tanh, output_activation=nn.Identity) -> nn.Sequential:
    layers=[]
    for i,(a,b) in enumerate(zip(sizes[:-1],sizes[1:])):
        layers.extend((nn.Linear(a,b),output_activation() if i==len(sizes)-2 else activation()))
    return nn.Sequential(*layers)

class GaussianActorCritic(nn.Module):
    def __init__(self,obs_dim:int,action_dim:int,hidden:Sequence[int]=(128,128)):
        super().__init__();self.actor=mlp([obs_dim,*hidden,action_dim]);self.log_std=nn.Parameter(torch.full((action_dim,),-0.5));self.critic=mlp([obs_dim,*hidden,1])
    def distribution(self,obs):return torch.distributions.Normal(self.actor(obs),self.log_std.exp().expand_as(self.actor(obs)))
    def act(self,obs,deterministic:bool=False):
        dist=self.distribution(obs);action=dist.mean if deterministic else dist.sample();return action,dist.log_prob(action).sum(-1),self.critic(obs).squeeze(-1)
