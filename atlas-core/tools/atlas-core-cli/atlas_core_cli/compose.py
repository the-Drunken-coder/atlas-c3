"""Docker Compose helpers for the Atlas Core lifecycle CLI.

These functions are intentionally thin wrappers around ``docker``/``docker
compose`` so the CLI can be inspected and reasoned about without invoking
the runtime. ``ROOT`` resolves to the ``atlas-core/`` directory (where
``docker-compose.yml`` lives) regardless of where the CLI is invoked from.
"""

from __future__ import annotations

import json
import os
import re
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


def parse_host_port(raw_port: str) -> int:
    """Validate a host port string and return it as an integer."""
    try:
        port = int(raw_port)
    except ValueError as exc:
        raise SystemExit(f"ATLAS_CORE_HOST_PORT must be a number, got {raw_port!r}") from exc
    if not (1 <= port <= 65535):
        raise SystemExit(f"ATLAS_CORE_HOST_PORT must be 1-65535, got {port}")
    return port


def dotenv_host_port() -> str | None:
    """Read ``ATLAS_CORE_HOST_PORT`` from ``ROOT/.env`` when present."""
    env_path = ROOT / ".env"
    if not env_path.is_file():
        return None
    for raw_line in env_path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue
        key, sep, value = line.partition("=")
        if sep and key.strip() == "ATLAS_CORE_HOST_PORT":
            return value.strip().strip("\"'")
    return None


def resolved_compose_host_port() -> int | None:
    """Return the published Atlas Core host port reported by Docker Compose."""
    result = run("docker", "compose", "-p", PROJECT, "port", "atlas-core", "8080", check=False)
    if result.returncode != 0:
        return None
    for raw_line in result.stdout.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        match = re.search(r":(\d+)$", line)
        if match:
            return parse_host_port(match.group(1))
    return None


def readiness_host_port() -> int:
    """Resolve the host port used for readiness polling."""
    compose_port = resolved_compose_host_port()
    if compose_port is not None:
        return compose_port
    raw_port = (os.environ.get("ATLAS_CORE_HOST_PORT") or "").strip()
    if raw_port:
        return parse_host_port(raw_port)
    dotenv_port = dotenv_host_port()
    if dotenv_port:
        return parse_host_port(dotenv_port)
    return 8080


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
    port = readiness_host_port()
    url = f"http://localhost:{port}/readiness"
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
