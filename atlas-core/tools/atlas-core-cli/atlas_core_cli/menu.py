def choose_action() -> str:
    actions = ["start", "restart", "shutdown"]
    print("Select Atlas Core action:")
    for index, action in enumerate(actions, start=1):
        print(f"  {index}. {action}")
    while True:
        selection = input("Enter selection [1-3]: ").strip()
        if selection in {"1", "2", "3"}:
            return actions[int(selection) - 1]
        print("Invalid selection.")
