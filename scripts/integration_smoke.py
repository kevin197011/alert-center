#!/usr/bin/env python3
import os
import sys
import subprocess

ROOT = os.path.dirname(os.path.abspath(__file__))
FAST = os.path.join(ROOT, "integration_smoke_fast.py")
SLOW = os.path.join(ROOT, "integration_smoke_slow.py")

MODE = "all"
if len(sys.argv) > 1:
    MODE = sys.argv[1]

cmds = []
if MODE in ("all", "fast"):
    cmds.append([sys.executable, FAST])
if MODE in ("all", "slow"):
    cmds.append([sys.executable, SLOW])

if not cmds:
    print("Usage: integration_smoke.py [all|fast|slow]")
    sys.exit(1)

for cmd in cmds:
    print(f"Running: {' '.join(cmd)}")
    proc = subprocess.run(cmd)
    if proc.returncode != 0:
        sys.exit(proc.returncode)

print("All integration smoke tests passed")
