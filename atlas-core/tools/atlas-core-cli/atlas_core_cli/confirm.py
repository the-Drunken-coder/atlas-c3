def require_confirmation(action: str, already_confirmed: bool) -> None:
    if already_confirmed:
        return
    typed = input(f"{action} is destructive. Type YES to continue: ").strip()
    if typed != "YES":
        raise SystemExit("Confirmation failed.")
