"""Docker Compose helpers for the Atlas Core lifecycle CLI.

These functions are intentionally thin wrappers around ``docker``/``docker
compose`` so the CLI can be inspected and reasoned about without invoking
the runtime. ``ROOT`` resolves to the ``atlas-core/`` directory (where
``docker-compose.yml`` lives) regardless of where the CLI is invoked from.
"""

from __future__ import annotations

import json
import os
import subprocess
import time
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
PROJECT = "atlas-core"
LABEL = f"atlas.core.project={PROJECT}"


def run(*args: str, check: bool = True) -> subprocess.CompletedProcess[str]:
    """Run a subprocess inside :data:`ROOT` and capture its stdout/stderr.

    Args:
        *args: Argv vector for the subprocess. The first element is the
            program; remaining elements are passed as separate args (no
            shell interpolation).
        check: If true, a non-zero exit status raises
            :class:`subprocess.CalledProcessError`.

    Returns:
        The completed :class:`subprocess.CompletedProcess` with text I/O.
    """
    return subprocess.run(args, cwd=ROOT, check=check, text=True, capture_output=True)


def compose_up(enable_fusion: bool) -> None:
    """Start the Atlas Core docker-compose project, optionally with the fusion profile.

    Always builds images (``--build``) and runs detached (``-d``).

    Args:
        enable_fusion: When true, activates the ``fusion`` profile so the
            data-fusion harness service is also started.
    """
    command = ["docker", "compose", "-p", PROJECT]
    if enable_fusion:
        command.extend(["--profile", "fusion"])
    command.extend(["up", "-d", "--build"])
    print("Starting Atlas Core compose project...")
    subprocess.run(command, cwd=ROOT, check=True)


def wait_for_readiness(timeout_seconds: int = 120) -> None:
    """Poll the Atlas Core readiness endpoint until it reports ``ready``.

    Each ``urlopen`` call is bounded by the remaining time budget so a hung
    connect/read cannot exceed the overall ``timeout_seconds`` deadline.
    Raises :class:`SystemExit` if the service is not ready in time.

    Args:
        timeout_seconds: Total wall-clock budget for the readiness probe.
    """
    url = f"http://localhost:{os.environ.get('ATLAS_CORE_PORT', '8080')}/readiness"
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        attempt_timeout = max(1.0, deadline - time.time())
        try:
            with urllib.request.urlopen(url, timeout=attempt_timeout) as response:
                payload = json.load(response)
                if response.status == 200 and payload.get("status") == "ready":
                    print("Atlas Core is ready.")
                    return
        except Exception:
            pass
        time.sleep(2)
    raise SystemExit("Atlas Core did not become ready in time.")


def destructive_cleanup() -> None:
    """Remove all Atlas Core containers, images, and volumes labeled with the project tag.

    Iterates over the three resource kinds, queries Docker for IDs/names
    matching ``atlas.core.project=atlas-core``, and force-removes each one.
    Intended to back the ``restart`` and ``shutdown`` CLI actions.
    """
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
