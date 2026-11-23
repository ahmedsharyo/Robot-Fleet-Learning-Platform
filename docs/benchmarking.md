# Benchmarking

Do not publish invented measurements. Record machine model, logical CPU count, Python/Go versions, dependency versions, environment ID/version, seeds, warm-up, run count, and raw results. Use at least three independent repetitions and report individual values, mean, standard deviation, and an uncertainty interval. For worker scaling, compare native and containerized runs separately and calculate `E_N = T_1 / (N * T_N)`. Avoid more workers than available CPU capacity.

Implemented harness: `benchmarks/scheduling/run.py`, which records per-request submission and completion acknowledgement latency. The full scalability, backpressure, failure injection, and BC/PPO comparison harnesses are still outstanding. There are no measured values in this repository yet.
