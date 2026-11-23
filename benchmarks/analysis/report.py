import json,sys
from pathlib import Path
data=json.loads(Path(sys.argv[1]).read_text());print(json.dumps(data,indent=2));print("\nOnly measured fields are reported; no benchmark values are inferred.")
