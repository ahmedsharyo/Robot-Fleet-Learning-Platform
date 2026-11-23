import pytest
from datasets.schema.episode import Episode, validate_episode
from datasets.validation.validate import validate_batch

def episode(id="e1"):
    return Episode.create(episode_id=id,environment_id="Reacher-v5",environment_version="1",observations=[[0.,1.],[1.,2.]],actions=[[.1]],rewards=[1.],terminated=[False],truncated=[True],episode_return=1.,episode_length=1,seed=3,policy_id="baseline",worker_id="w",config_hash="abc")

def test_episode_alignment_and_return():
    ep=episode();validate_episode(ep);assert ep.episode_length==1

def test_invalid_alignment_rejected():
    ep=episode();ep.observations=ep.observations[:1]
    with pytest.raises(ValueError):validate_episode(ep)

def test_duplicate_ids_rejected():
    good,errors=validate_batch([episode(),episode()]);assert len(good)==1 and errors[0]["code"]=="INVALID_EPISODE"
