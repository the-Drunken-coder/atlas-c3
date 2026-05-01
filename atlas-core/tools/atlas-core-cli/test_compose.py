"""Regression tests for Atlas Core lifecycle compose helpers."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import types
import unittest
from unittest import mock


MODULE_PATH = Path(__file__).resolve().parent / "atlas_core_cli" / "compose.py"
_MODULE_COUNTER = 0


def import_compose_module():
    """Import the compose helper module under a unique temporary name."""
    global _MODULE_COUNTER
    _MODULE_COUNTER += 1
    spec = importlib.util.spec_from_file_location(f"atlas_core_cli_compose_test_{_MODULE_COUNTER}", MODULE_PATH)
    if spec is None or spec.loader is None:
        raise AssertionError("failed to load compose helper module")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ComposeHelperTests(unittest.TestCase):
    """Tests for compose readiness-port resolution."""

    def test_readiness_host_port_uses_compose_mapping(self) -> None:
        """Prefer Docker's resolved host port over environment defaults."""
        module = import_compose_module()
        with (
            mock.patch.object(module, "run", return_value=types.SimpleNamespace(returncode=0, stdout="0.0.0.0:49123\n", stderr="")),
            mock.patch.object(module.os, "environ", {"ATLAS_CORE_HOST_PORT": "8080"}),
        ):
            self.assertEqual(module.readiness_host_port(), 49123)

    def test_readiness_host_port_falls_back_to_dotenv(self) -> None:
        """Use ROOT/.env when the shell environment does not export the host port."""
        module = import_compose_module()
        env_path = module.ROOT / ".env"
        original = None
        if env_path.exists():
            original = env_path.read_text(encoding="utf-8")
        try:
            env_path.write_text("ATLAS_CORE_HOST_PORT=19090\n", encoding="utf-8")
            with (
                mock.patch.object(module, "run", return_value=types.SimpleNamespace(returncode=1, stdout="", stderr="boom")),
                mock.patch.object(module.os, "environ", {}),
            ):
                self.assertEqual(module.readiness_host_port(), 19090)
        finally:
            if original is None:
                env_path.unlink(missing_ok=True)
            else:
                env_path.write_text(original, encoding="utf-8")

    def test_parse_host_port_rejects_invalid_values(self) -> None:
        """Reject non-numeric host port overrides."""
        module = import_compose_module()
        with self.assertRaisesRegex(SystemExit, "ATLAS_CORE_HOST_PORT"):
            module.parse_host_port("not-a-port")


if __name__ == "__main__":
    unittest.main()
