"""Entry point for the Atlas Core lifecycle CLI.

The CLI wraps ``docker compose`` for local development: it starts, restarts,
or fully tears down the Atlas Core stack and then waits for the readiness
endpoint before returning. ``restart`` and ``shutdown`` are destructive
(they remove containers, images, and named volumes) and require a typed
``YES`` confirmation unless ``--yes``/``--confirm`` is supplied.
"""

from atlas_core_cli.menu import choose_action
from atlas_core_cli.confirm import require_confirmation
from atlas_core_cli.compose import compose_up, wait_for_readiness, destructive_cleanup
import argparse
import os


def main() -> int:
    """Parse arguments, dispatch the chosen lifecycle action, and return an exit code.

    If no action is supplied on the command line, the user is prompted via
    :func:`atlas_core_cli.menu.choose_action`. Destructive actions
    (``restart`` and ``shutdown``) require explicit confirmation. The data
    fusion harness profile is enabled iff the ``ATLAS_DATA_FUSION_STACK``
    environment variable is set to a non-empty value.

    Returns:
        Exit code suitable for ``sys.exit`` / ``SystemExit``.
    """
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
