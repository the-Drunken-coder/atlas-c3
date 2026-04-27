"""Regression tests for the Atlas Data Fusion harness runner."""

from __future__ import annotations

import importlib.util
import os
from pathlib import Path
import unittest
import urllib.error
from unittest import mock


MODULE_PATH = Path(__file__).with_name("main.py")
_MODULE_COUNTER = 0


def load_module(base_url: str | None):
    """Load the runner module with a temporary ATLAS_CORE_BASE_URL value."""
    global _MODULE_COUNTER
    _MODULE_COUNTER += 1
    patched_env = {} if base_url is None else {"ATLAS_CORE_BASE_URL": base_url}
    with mock.patch.dict(os.environ, patched_env, clear=False):
        if base_url is None:
            os.environ.pop("ATLAS_CORE_BASE_URL", None)
        spec = importlib.util.spec_from_file_location(f"atlas_data_fusion_runner_test_{_MODULE_COUNTER}", MODULE_PATH)
        if spec is None or spec.loader is None:
            raise AssertionError("failed to load harness runner module")
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module


class HarnessRunnerTests(unittest.TestCase):
    """Tests for the Atlas Data Fusion harness runner module."""

    def test_validate_base_url_rejects_unsafe_or_missing_schemes(self) -> None:
        """Reject base URLs that are not absolute HTTP(S) URLs."""
        for base_url in ("file:///tmp/atlas-core", "atlas-core:8080"):
            with self.subTest(base_url=base_url):
                with self.assertRaisesRegex(ValueError, "ATLAS_CORE_BASE_URL"):
                    load_module(base_url)

    def test_validate_base_url_rejects_query_fragment_and_userinfo(self) -> None:
        """Reject malformed base URLs that are unsafe to use as request prefixes."""
        for base_url in (
            "https://atlas-core.example/internal?x=1",
            "https://atlas-core.example/internal#fragment",
            "http://user:pass@atlas-core.example/internal",
        ):
            with self.subTest(base_url=base_url):
                with self.assertRaisesRegex(ValueError, "ATLAS_CORE_BASE_URL"):
                    load_module(base_url)

    def test_validate_base_url_accepts_http_and_https(self) -> None:
        """Normalize accepted HTTP(S) base URLs."""
        module = load_module("https://atlas-core.example/internal/")
        self.assertEqual(module.BASE_URL, "https://atlas-core.example/internal")

    def test_ensure_track_treats_conflict_as_already_exists(self) -> None:
        """Treat a concurrent create conflict as an already-existing track."""
        module = load_module("http://atlas-core:8080")
        with mock.patch.object(
            module,
            "request",
            side_effect=[
                urllib.error.HTTPError(f"{module.BASE_URL}/entities/fusion-track-baseline", 404, "not found", None, None),
                urllib.error.HTTPError(f"{module.BASE_URL}/entities", 409, "conflict", None, None),
            ],
        ):
            self.assertFalse(module.ensure_track())

    def test_ensure_track_propagates_non_409_errors(self) -> None:
        """Propagate create errors other than HTTP 409."""
        module = load_module("http://atlas-core:8080")
        with mock.patch.object(
            module,
            "request",
            side_effect=[
                urllib.error.HTTPError(f"{module.BASE_URL}/entities/fusion-track-baseline", 404, "not found", None, None),
                urllib.error.HTTPError(f"{module.BASE_URL}/entities", 500, "boom", None, None),
            ],
        ):
            with self.assertRaises(urllib.error.HTTPError) as ctx:
                module.ensure_track()
        self.assertEqual(ctx.exception.code, 500)


if __name__ == "__main__":
    unittest.main()
