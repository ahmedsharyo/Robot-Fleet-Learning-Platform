import pytest

def test_bc_mlp_output_shape_and_gradient():
    torch=pytest.importorskip("torch")
    from learning.models.mlp import mlp
    model=mlp([5,16,2]);x=torch.randn(4,5);y=model(x);assert y.shape==(4,2);loss=y.square().mean();loss.backward();assert model[0].weight.grad is not None
