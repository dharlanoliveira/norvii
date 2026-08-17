import subprocess
import unittest
from pathlib import Path

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]


class PersistenceMakefileTest(unittest.TestCase):
    def test_bootstrap_delegates_to_managed_local_script(self) -> None:
        result = subprocess.run(
            ["make", "--no-print-directory", "--dry-run", "bootstrap"],
            cwd=REPOSITORY_ROOT,
            check=True,
            capture_output=True,
            text=True,
        )

        self.assertIn(
            "python infra/scripts/manage-local-environment.py start", result.stdout
        )

    def test_default_persistence_journey_runs_each_required_step_in_order(self) -> None:
        result = subprocess.run(
            ["make", "--no-print-directory", "MAKE=printf '%s\\n'", "persistence"],
            cwd=REPOSITORY_ROOT,
            check=True,
            capture_output=True,
            text=True,
        )

        self.assertEqual(
            ["persistence-up", "persistence-migrate", "persistence-verify"],
            result.stdout.splitlines(),
        )

    def test_local_lifecycle_delegates_to_the_environment_manager(self) -> None:
        for target, action in (
            ("local-start", "start"),
            ("local-status", "status"),
            ("local-stop", "stop"),
        ):
            with self.subTest(target=target):
                result = subprocess.run(
                    ["make", "--no-print-directory", "--dry-run", target],
                    cwd=REPOSITORY_ROOT,
                    check=True,
                    capture_output=True,
                    text=True,
                )

                self.assertIn(
                    f"python infra/scripts/manage-local-environment.py {action}",
                    result.stdout,
                )


if __name__ == "__main__":
    unittest.main()
