"""Atlas Data Fusion baseline harness.

Runs as a long-lived sidecar against Atlas Core. Each tick: fetches the
full query state (sanity check) and ensures a baseline fusion track entity
exists. The harness is intentionally minimal — it exercises the Atlas Core
HTTP surface without performing real fusion logic.

The ``ATLAS_CORE_BASE_URL`` and ``ATLAS_DATA_FUSION_STACK`` environment
variables select the target service and the labelled algorithm. All HTTP
requests are bounded by :data:`REQUEST_TIMEOUT_SECONDS` so a stalled core
cannot hang the harness indefinitely.
"""

import http.client
import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request

STACK = os.environ.get("ATLAS_DATA_FUSION_STACK", "baseline")
REQUEST_TIMEOUT_SECONDS = 10


def validate_base_url(value: str) -> str:
    """Validate and normalize the Atlas Core base URL."""
    parsed = urllib.parse.urlparse(value)
    if parsed.scheme not in {"http", "https"} or not parsed.netloc:
        raise ValueError("ATLAS_CORE_BASE_URL must be an absolute http(s) URL")
    if parsed.query or parsed.fragment:
        raise ValueError("ATLAS_CORE_BASE_URL must not include a query string or fragment")
    if parsed.username is not None or parsed.password is not None:
        raise ValueError("ATLAS_CORE_BASE_URL must not include user credentials")
    normalized = f"{parsed.scheme}://{parsed.netloc}{parsed.path}".rstrip("/")
    return normalized


BASE_URL = validate_base_url(os.environ.get("ATLAS_CORE_BASE_URL", "http://atlas-core:8080"))


def request(method: str, path: str, payload: dict | None = None) -> dict:
    """Send a JSON request to Atlas Core and decode the JSON response.

    Args:
        method: HTTP method (``GET``, ``POST``, etc.).
        path: Request path joined to :data:`BASE_URL`.
        payload: Optional JSON-serializable body. Sent with
            ``Content-Type: application/json`` when present.

    Returns:
        The decoded JSON response body as a Python dict.

    Raises:
        urllib.error.HTTPError: For non-2xx responses.
        urllib.error.URLError: For transport failures (including timeout).
    """
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(f"{BASE_URL}{path}", data=data, method=method, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=REQUEST_TIMEOUT_SECONDS) as response:
        return json.load(response)


def ensure_track() -> bool:
    """Ensure the baseline fusion track entity exists in Atlas Core.

    Returns:
        ``True`` when this call had to create the entity (404 on GET, then
        a successful POST). ``False`` when the entity already existed, or
        when a concurrent create returns HTTP 409. Used by the caller to log
        accurate ``created`` vs ``verified`` messages.

    Raises:
        urllib.error.HTTPError: Any non-404 GET error, or any non-409 POST
            error, is re-raised to the tick loop so it can log the failure.
    """
    try:
        request("GET", "/entities/fusion-track-baseline")
        return False
    except urllib.error.HTTPError as exc:
        if exc.code != 404:
            raise
    try:
        request(
            "POST",
            "/entities",
            {
                "entity_id": "fusion-track-baseline",
                "type": "track",
                "json": {"components": {"fusion_summary": {"observed_at": time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()), "algorithm": STACK}}, "extra": {"created_by": "atlas-data-fusion"}},
            },
        )
    except urllib.error.HTTPError as exc:
        if exc.code == 409:
            return False
        raise
    return True


def main() -> None:
    """Run the harness tick loop forever.

    Each iteration performs a full-query GET, ensures the baseline track,
    and emits a single JSON line describing the outcome. Network, HTTP,
    OS, and JSON decode failures are caught per-tick and logged as
    ``fusion.error`` events; other exceptions still propagate. The loop
    sleeps 10 seconds between ticks regardless of success.
    """
    while True:
        try:
            request("GET", "/queries/full")
            created = ensure_track()
            message = "baseline stack created track" if created else "baseline stack verified track"
            print(json.dumps({"service": "atlas-data-fusion", "event": "fusion.tick", "message": message}))
        except (
            urllib.error.URLError,
            http.client.HTTPException,
            # Keep genuine request timeouts in the per-tick error path, but let
            # unrelated local OSErrors still propagate.
            TimeoutError,
            UnicodeDecodeError,
            json.JSONDecodeError,
        ) as exc:
            print(json.dumps({"service": "atlas-data-fusion", "event": "fusion.error", "message": str(exc)}))
        time.sleep(10)


if __name__ == "__main__":
    main()
