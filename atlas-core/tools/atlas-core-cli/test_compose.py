"""Regression tests for Atlas Core lifecycle compose helpers."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import tempfile
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
        with tempfile.TemporaryDirectory() as temp_dir:
            env_path = Path(temp_dir) / ".env"
            env_path.write_text("ATLAS_CORE_HOST_PORT=19090\n", encoding="utf-8")
            with (
                mock.patch.object(module, "ROOT", Path(temp_dir)),
                mock.patch.object(module, "run", return_value=types.SimpleNamespace(returncode=1, stdout="", stderr="boom")),
                mock.patch.object(module.os, "environ", {}),
            ):
                self.assertEqual(module.readiness_host_port(), 19090)

    def test_parse_host_port_rejects_invalid_values(self) -> None:
        """Reject non-numeric host port overrides."""
        module = import_compose_module()
        with self.assertRaisesRegex(SystemExit, "ATLAS_CORE_HOST_PORT"):
            module.parse_host_port("not-a-port")

    def test_wait_for_readiness_reports_last_error(self) -> None:
        """Include the last readiness failure in the timeout message."""
        module = import_compose_module()
        monotonic_values = iter([0.0, 0.0, 3.1, 3.1])
        with (
            mock.patch.object(module, "readiness_host_port", return_value=8080),
            mock.patch.object(module.urllib.request, "urlopen", side_effect=module.urllib.error.URLError("boom")),
            mock.patch.object(module.time, "monotonic", side_effect=lambda: next(monotonic_values)),
            mock.patch.object(module.time, "sleep"),
        ):
            with self.assertRaisesRegex(SystemExit, "boom"):
                module.wait_for_readiness(timeout_seconds=3)

    def test_destructive_cleanup_deduplicates_resource_ids(self) -> None:
        """Avoid repeating docker removals when list commands return duplicate IDs."""
        module = import_compose_module()
        with (
            mock.patch.object(
                module,
                "run",
                side_effect=[
                    types.SimpleNamespace(returncode=0, stdout="container-1\n", stderr=""),
                    types.SimpleNamespace(returncode=0, stdout="image-1\nimage-1\nimage-2\n", stderr=""),
                    types.SimpleNamespace(returncode=0, stdout="volume-1\n", stderr=""),
                ],
            ),
            mock.patch.object(module.subprocess, "run") as subprocess_run,
        ):
            module.destructive_cleanup()
        removals = [call.args[0] for call in subprocess_run.call_args_list]
        self.assertIn(["docker", "rm", "-f", "container-1"], removals)
        self.assertEqual(removals.count(["docker", "rmi", "-f", "image-1"]), 1)
        self.assertEqual(removals.count(["docker", "rmi", "-f", "image-2"]), 1)
        self.assertIn(["docker", "volume", "rm", "-f", "volume-1"], removals)


if __name__ == "__main__":
    unittest.main()
