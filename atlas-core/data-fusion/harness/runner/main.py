import json
import os
import time
import urllib.request
import urllib.error

BASE_URL = os.environ.get("ATLAS_CORE_BASE_URL", "http://atlas-core:8080")
STACK = os.environ.get("ATLAS_DATA_FUSION_STACK", "baseline")


def request(method: str, path: str, payload: dict | None = None) -> dict:
    data = None if payload is None else json.dumps(payload).encode()
    req = urllib.request.Request(f"{BASE_URL}{path}", data=data, method=method, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req) as response:
        return json.load(response)


def ensure_track() -> None:
    try:
        request("GET", "/entities/fusion-track-baseline")
        return
    except urllib.error.HTTPError as exc:
        if exc.code != 404:
            raise
    request(
        "POST",
        "/entities",
        {
            "entity_id": "fusion-track-baseline",
            "type": "track",
            "json": {"components": {"fusion_summary": {"observed_at": time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()), "algorithm": STACK}}, "extra": {"created_by": "atlas-data-fusion"}},
        },
    )


def main() -> None:
    while True:
        try:
            request("GET", "/queries/full")
            ensure_track()
            print(json.dumps({"service": "atlas-data-fusion", "event": "fusion.tick", "message": "baseline stack wrote track"}))
        except Exception as exc:
            print(json.dumps({"service": "atlas-data-fusion", "event": "fusion.error", "message": str(exc)}))
        time.sleep(10)


if __name__ == "__main__":
    main()
