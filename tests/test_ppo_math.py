import numpy as np
import pytest

def test_gae_true_terminal_masks_bootstrap():
    from learning.ppo.train import compute_gae
    adv,ret=compute_gae([1.],[2.],[True],[False],[100.],gamma=.9,lam=.95)
    assert ret[0]==pytest.approx(1.)

def test_gae_truncation_bootstraps():
    from learning.ppo.train import compute_gae
    adv,ret=compute_gae([1.],[2.],[False],[True],[3.],gamma=.9,lam=.95)
    assert ret[0]==pytest.approx(1.+.9*3.)
