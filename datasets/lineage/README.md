# Artifact lineage

The orchestration job ID is recorded in each dataset manifest's provenance. Training metadata records the immutable dataset path and checkpoint digest. Evaluation JSON records the policy identifier, seeds, and metrics. Artifacts use immutable dataset IDs/versions and content hashes; the local shared filesystem is the transport. A future object-store backend can implement the same URI/hash references.
