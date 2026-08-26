import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
RESET_SCRIPT = REPOSITORY_ROOT / "infra" / "scripts" / "reset-local-data.sh"


class PersistenceResetScriptTest(unittest.TestCase):
    def test_refuses_reset_without_exact_confirmation_before_calling_docker(self) -> None:
        result, docker_calls = self.run_script(confirmation="wrong-confirmation")

        self.assertNotEqual(0, result.returncode)
        self.assertIn("explicit confirmation", result.stderr)
        self.assertEqual("", docker_calls)

    def test_removes_only_exact_owned_volumes_after_confirmation(self) -> None:
        result, docker_calls = self.run_script(confirmation="reset-norvii-data")

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn(
            "volume rm norvii_postgres_data norvii_neo4j_data",
            docker_calls,
        )
        self.assertIn("cannot be recovered", result.stdout)

    def test_refuses_unexpected_project_volume(self) -> None:
        result, docker_calls = self.run_script(
            confirmation="reset-norvii-data",
            project_volumes="norvii_postgres_data\nnorvii_neo4j_data\nunexpected_volume",
        )

        self.assertNotEqual(0, result.returncode)
        self.assertIn("unexpected persistence volumes", result.stderr)
        self.assertNotIn("volume rm", docker_calls)

    def test_refuses_reset_without_assertion_preflight_before_calling_docker(self) -> None:
        result, docker_calls = self.run_script(
            confirmation="reset-norvii-data", preflight="missing"
        )

        self.assertNotEqual(0, result.returncode)
        self.assertIn("normative assertion preflight", result.stderr)
        self.assertEqual("", docker_calls)

    def run_script(
        self,
        *,
        confirmation: str,
        preflight: str = "passed",
        project_volumes: str = "norvii_postgres_data\nnorvii_neo4j_data",
    ) -> tuple[subprocess.CompletedProcess[str], str]:
        with tempfile.TemporaryDirectory() as temporary_directory:
            directory = Path(temporary_directory)
            calls_file = directory / "docker-calls"
            environment_file = directory / ".env"
            environment_file.write_text("LOCAL_TEST_VALUE=present\n", encoding="utf-8")
            docker = directory / "docker"
            docker.write_text(
                textwrap.dedent(
                    f"""\
                    #!/usr/bin/env bash
                    set -eu
                    printf '%s\\n' "$*" >> "{calls_file}"
                    if [[ "$*" == *"compose"*"config --format json"* ]]; then
                      printf '%s\\n' '{{"name":"norvii"}}'
                    elif [[ "$*" == *"volume ls"* ]]; then
                      printf '%s\\n' '{project_volumes}'
                    elif [[ "$*" == *"volume inspect"*"norvii_postgres_data"* ]]; then
                      printf '%s\\n' 'norvii norvii_postgres_data'
                    elif [[ "$*" == *"volume inspect"*"norvii_neo4j_data"* ]]; then
                      printf '%s\\n' 'norvii norvii_neo4j_data'
                    fi
                    """
                ),
                encoding="utf-8",
            )
            docker.chmod(docker.stat().st_mode | stat.S_IXUSR)
            environment = os.environ | {
                "PATH": f"{directory}:{os.environ['PATH']}",
                "NORVII_ASSERTION_RESET_PREFLIGHT": preflight,
            }
            result = subprocess.run(
                ["bash", str(RESET_SCRIPT), str(environment_file), confirmation],
                cwd=REPOSITORY_ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )
            calls = calls_file.read_text(encoding="utf-8") if calls_file.exists() else ""
        return result, calls


if __name__ == "__main__":
    unittest.main()
