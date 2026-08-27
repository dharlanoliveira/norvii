"""Deterministic lifecycle coverage for the managed evaluation worker registration."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).with_name("manage-local-environment.py")
SPECIFICATION = importlib.util.spec_from_file_location("manage_local_environment", SCRIPT_PATH)
if SPECIFICATION is None or SPECIFICATION.loader is None:
    raise RuntimeError("Could not load the local environment manager module.")
MODULE = importlib.util.module_from_spec(SPECIFICATION)
sys.modules[SPECIFICATION.name] = MODULE
SPECIFICATION.loader.exec_module(MODULE)


class EvaluationWorkerLifecycleTest(unittest.TestCase):
    """Verify worker lifecycle registration without starting local services."""

    def test_worker_uses_persistent_log_and_readiness_marker(self) -> None:
        repository_root = Path(__file__).parents[2]
        layout = MODULE.RepositoryLayout(repository_root)
        manager = MODULE.LocalEnvironmentManager(layout)

        worker = manager._processes["evaluation-worker"]

        self.assertEqual(layout.log("evaluation-worker"), repository_root / ".log" / "evaluation-worker.log")
        self.assertEqual(layout.ready_marker("evaluation-worker"), repository_root / ".log" / "evaluation-worker.ready")
        self.assertIn("./cmd/evaluation-worker", worker._command)
        self.assertIn("--ready-file", worker._command)
        self.assertIn(str(layout.ready_marker("evaluation-worker")), worker._command)

    def test_restart_is_a_supported_lifecycle_action(self) -> None:
        options = MODULE.parse_arguments(["restart", "--repository-root", "/tmp/norvii"])

        self.assertEqual(options.action, "restart")
        self.assertEqual(options.repository_root, Path("/tmp/norvii"))


if __name__ == "__main__":
    unittest.main()
