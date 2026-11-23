# Data model

An episode stores observations before the first action and after every action (`T+1`), plus `T` actions, rewards, terminated flags, and truncated flags. Return must equal the reward sum. The final transition must end the episode, with no transitions after an ending flag. Random-policy output is explicitly named `random-baseline`.

Parquet partitions hold episode rows. An immutable `manifest.json` records IDs, schema/environment versions, content hashes, counts, provenance, and timestamp. Readers verify each partition hash and stream row batches. A training run should retain manifest path/hash; evaluation should retain checkpoint hash and unseen seeds.
