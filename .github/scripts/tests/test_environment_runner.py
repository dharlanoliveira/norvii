import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
RUNNER = REPOSITORY_ROOT / "infra" / "scripts" / "run-with-environment.py"


class EnvironmentRunnerTest(unittest.TestCase):
    def test_runs_command_with_literal_quoted_values(self) -> None:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8") as environment_file:
            environment_file.write(
                "PLAIN_VALUE=plain\n"
                "QUOTED_VALUE='space dollar-$ slash-/ hash-#'\n"
                "# ignored comment\n"
            )
            environment_file.flush()

            result = subprocess.run(
                [
                    sys.executable,
                    str(RUNNER),
                    environment_file.name,
                    sys.executable,
                    "-c",
                    "import os; print(os.environ['PLAIN_VALUE']); print(os.environ['QUOTED_VALUE'])",
                ],
                cwd=REPOSITORY_ROOT,
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual("plain\nspace dollar-$ slash-/ hash-#\n", result.stdout)

    def test_rejects_invalid_assignment_without_running_command(self) -> None:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8") as environment_file:
            environment_file.write("NOT AN ASSIGNMENT\n")
            environment_file.flush()

            result = subprocess.run(
                [sys.executable, str(RUNNER), environment_file.name, sys.executable, "-c", "print('ran')"],
                cwd=REPOSITORY_ROOT,
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertNotEqual(0, result.returncode)
        self.assertNotIn("ran", result.stdout)
        self.assertIn("line 1", result.stderr)


if __name__ == "__main__":
    unittest.main()
