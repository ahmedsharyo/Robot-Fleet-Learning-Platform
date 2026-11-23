# Reproducibility

Set explicit environment, dataset, and training seeds. Keep dataset versions immutable and retain manifest/checkpoint SHA-256 values. The BC split is by episode, not timestep. PPO collects rollouts in the environment during its training run. Checkpoints are atomically renamed and contain model/optimizer state, progress, and random-number generator state. Exact bitwise reproducibility is not promised across CPU architectures, operating systems, or library versions.
