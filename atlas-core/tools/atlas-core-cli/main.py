from atlas_core_cli.menu import choose_action
from atlas_core_cli.confirm import require_confirmation
from atlas_core_cli.compose import compose_up, wait_for_readiness, destructive_cleanup
import argparse
import os


def main() -> int:
    parser = argparse.ArgumentParser(description="Atlas Core lifecycle CLI")
    parser.add_argument("action", nargs="?", choices=["start", "restart", "shutdown"])
    parser.add_argument("--yes", "--confirm", action="store_true", dest="yes")
    args = parser.parse_args()

    action = args.action or choose_action()
    enable_fusion = bool((os.environ.get("ATLAS_DATA_FUSION_STACK") or "").strip())
    if action in {"restart", "shutdown"}:
        require_confirmation(action, args.yes)
        destructive_cleanup()
        if action == "shutdown":
            return 0
    compose_up(enable_fusion=enable_fusion)
    wait_for_readiness()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
