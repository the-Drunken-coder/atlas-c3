from __future__ import annotations

import json
import os
import subprocess
import time
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
PROJECT = "atlas-core"
LABEL = f"atlas.core.project={PROJECT}"


def run(*args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(args, cwd=ROOT, check=check, text=True, capture_output=True)


def compose_up(enable_fusion: bool) -> None:
    command = ["docker", "compose", "-p", PROJECT]
    if enable_fusion:
        command.extend(["--profile", "fusion"])
    command.extend(["up", "-d", "--build"])
    print("Starting Atlas Core compose project...")
    subprocess.run(command, cwd=ROOT, check=True)


def wait_for_readiness(timeout_seconds: int = 120) -> None:
    url = f"http://localhost:{os.environ.get('ATLAS_CORE_PORT', '8080')}/readiness"
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(url) as response:
                payload = json.load(response)
                if response.status == 200 and payload.get("status") == "ready":
                    print("Atlas Core is ready.")
                    return
        except Exception:
            pass
        time.sleep(2)
    raise SystemExit("Atlas Core did not become ready in time.")


def destructive_cleanup() -> None:
    print("Removing Atlas Core project resources...")
    for resource, args in [
        ("container", ["docker", "ps", "-a", "--filter", f"label={LABEL}", "--format", "{{.ID}}"]),
        ("image", ["docker", "images", "--filter", f"label={LABEL}", "--format", "{{.ID}}"]) ,
        ("volume", ["docker", "volume", "ls", "--filter", f"label={LABEL}", "--format", "{{.Name}}"]) ,
    ]:
        result = run(*args)
        ids = [line for line in result.stdout.splitlines() if line.strip()]
        for item in ids:
            if resource == "container":
                subprocess.run(["docker", "rm", "-f", item], check=True)
            elif resource == "image":
                subprocess.run(["docker", "rmi", "-f", item], check=True)
            elif resource == "volume":
                subprocess.run(["docker", "volume", "rm", "-f", item], check=True)
