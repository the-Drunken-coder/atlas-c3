"""Destructive-action confirmation prompt for the Atlas Core CLI."""


def require_confirmation(action: str, already_confirmed: bool) -> None:
    """Block until the user types ``YES`` to confirm a destructive action.

    Exits the process with a non-zero ``SystemExit`` if the user declines,
    if stdin is closed (e.g. CI / piped input), or if the prompt is
    interrupted with Ctrl-C. Callers can bypass the prompt entirely by
    passing ``already_confirmed=True`` (typically driven by ``--yes``).

    Args:
        action: Human-readable name of the action being confirmed (for
            display in the prompt).
        already_confirmed: When true, returns immediately without prompting.

    Raises:
        SystemExit: If the user does not type ``YES`` exactly, or if the
            confirmation prompt cannot be presented.
    """
    if already_confirmed:
        return
    try:
        typed = input(f"{action} is destructive. Type YES to continue: ").strip()
    except EOFError as err:
        raise SystemExit("Confirmation failed: no input available (use --yes to confirm non-interactively).") from err
    except KeyboardInterrupt as err:
        raise SystemExit("Confirmation aborted.") from err
    if typed != "YES":
        raise SystemExit("Confirmation failed.")
