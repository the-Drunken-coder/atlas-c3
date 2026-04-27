"""Interactive action selection for the Atlas Core CLI.

Used when the user invokes the CLI without an explicit ``action`` argument.
"""


def choose_action() -> str:
    """Prompt the user to pick one of ``start``, ``restart``, or ``shutdown``.

    Loops until the user enters a valid numeric selection. Blocks on stdin;
    callers that run non-interactively should pass the action via argv
    instead of relying on this prompt.

    Returns:
        The chosen action name.
    """
    actions = ["start", "restart", "shutdown"]
    print("Select Atlas Core action:")
    for index, action in enumerate(actions, start=1):
        print(f"  {index}. {action}")
    while True:
        try:
            selection = input("Enter selection [1-3]: ").strip()
        except EOFError:
            print()
            return "shutdown"
        if selection in {"1", "2", "3"}:
            return actions[int(selection) - 1]
        print("Invalid selection.")
