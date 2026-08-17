import os
import socket
import stat
import subprocess
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
MANAGER = REPOSITORY_ROOT / "infra" / "scripts" / "manage-local-environment.py"


class LocalEnvironmentManagerTest(unittest.TestCase):
    def test_start_status_and_stop_manage_components_with_isolated_logs(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            environment = self.prepare_fixture(root)

            start_result = self.run_manager(root, environment, "start")
            try:
                self.assertEqual(0, start_result.returncode, start_result.stderr)
                self.assertIn("Norvii is ready", start_result.stdout)
                first_process_identities = {
                    path.name: path.read_text(encoding="utf-8")
                    for path in sorted((root / ".log").glob("*.pid"))
                }

                repeated_start = self.run_manager(root, environment, "start")
                self.assertEqual(0, repeated_start.returncode, repeated_start.stderr)
                self.assertEqual(
                    first_process_identities,
                    {
                        path.name: path.read_text(encoding="utf-8")
                        for path in sorted((root / ".log").glob("*.pid"))
                    },
                )

                status_result = self.run_manager(root, environment, "status")
                self.assertEqual(0, status_result.returncode, status_result.stderr)
                self.assertIn("PostgreSQL is healthy", status_result.stdout)
                self.assertIn("Web is running", status_result.stdout)
                self.assertIn("API initialization is complete", status_result.stdout)
                self.assertIn(
                    "Ingestion initialization is complete", status_result.stdout
                )

                pid_files = sorted((root / ".log").glob("*.pid"))
                initial_identities = [path.read_text() for path in pid_files]
                repeated_result = self.run_manager(root, environment, "start")
                self.assertEqual(0, repeated_result.returncode, repeated_result.stderr)
                self.assertEqual(
                    initial_identities, [path.read_text() for path in pid_files]
                )

                failing_environment = environment | {
                    "FAIL_TARGET": "persistence-migrate"
                }
                failed_restart = self.run_manager(root, failing_environment, "start")
                self.assertNotEqual(0, failed_restart.returncode)
                self.assertEqual(
                    initial_identities, [path.read_text() for path in pid_files]
                )

                logs = root / ".log"
                self.assertIn("persistence-up", (logs / "bootstrap.log").read_text())
                self.assertIn("persistence-migrate", (logs / "api.log").read_text())
                self.assertIn("persistence-verify-api", (logs / "api.log").read_text())
                self.assertIn(
                    "persistence-verify-ingestion",
                    (logs / "ingestion.log").read_text(),
                )
                self.assertIn(
                    "npm dependencies installed", (logs / "web.log").read_text()
                )
                self.assertIn(
                    "postgres container log", (logs / "postgres.log").read_text()
                )
                self.assertIn("neo4j container log", (logs / "neo4j.log").read_text())
                self.assertEqual(
                    1,
                    (root / "command-trace.log")
                    .read_text(encoding="utf-8")
                    .count("npm-ci"),
                )
            finally:
                stop_result = self.run_manager(root, environment, "stop")

            self.assertEqual(0, stop_result.returncode, stop_result.stderr)
            self.assertIn("Norvii is stopped", stop_result.stdout)
            self.assertFalse(list((root / ".log").glob("*.pid")))
            self.assertIn("persistence-stop", (root / "command-trace.log").read_text())

    def test_start_failure_names_the_component_log(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)
            environment = self.prepare_fixture(root)
            environment["FAIL_TARGET"] = "persistence-migrate"

            result = self.run_manager(root, environment, "start")

            self.assertNotEqual(0, result.returncode)
            self.assertIn(".log/api.log", result.stderr)
            self.assertTrue((root / ".log" / "api.log").is_file())

    def test_rejects_non_repository_root_without_creating_runtime_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary_directory:
            root = Path(temporary_directory)

            result = self.run_manager(root, os.environ.copy(), "start")

            self.assertNotEqual(0, result.returncode)
            self.assertIn("not a Norvii repository root", result.stderr)
            self.assertFalse((root / ".log").exists())

    def run_manager(
        self, root: Path, environment: dict[str, str], action: str
    ) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(MANAGER), action, "--repository-root", str(root)],
            cwd=REPOSITORY_ROOT,
            env=environment,
            check=False,
            capture_output=True,
            text=True,
            timeout=30,
        )

    def prepare_fixture(self, root: Path) -> dict[str, str]:
        (root / "infra").mkdir()
        (root / "apps" / "web").mkdir(parents=True)
        (root / "infra" / ".env").write_text(
            "TEST_ENVIRONMENT=true\n", encoding="utf-8"
        )
        (root / "infra" / "compose.yaml").write_text("services: {}\n", encoding="utf-8")
        (root / "Makefile").write_text("fixture:\n\t@true\n", encoding="utf-8")
        (root / "apps" / "web" / "package.json").write_text("{}\n", encoding="utf-8")

        binary_directory = root / "bin"
        binary_directory.mkdir()
        self.write_executable(
            binary_directory / "make",
            """
            target="${@: -1}"
            printf '%s\n' "$target" >> "$COMMAND_TRACE"
            printf '%s\n' "$target"
            if [[ "${FAIL_TARGET:-}" == "$target" ]]; then
              exit 1
            fi
            if [[ "$target" == "persistence-health" ]]; then
              printf '%s\n' 'PostgreSQL is healthy.' 'Neo4j is healthy.'
            fi
            """,
        )
        self.write_executable(
            binary_directory / "npm",
            """
            if [[ "$*" == *" ci"* ]]; then
              printf '%s\n' 'npm-ci' >> "$COMMAND_TRACE"
              printf '%s\n' 'npm dependencies installed'
              exit 0
            fi
            port="5173"
            previous=""
            for argument in "$@"; do
              if [[ "$previous" == "--port" ]]; then
                port="$argument"
              fi
              previous="$argument"
            done
            exec python3 -m http.server "$port" --bind 127.0.0.1
            """,
        )
        self.write_executable(
            binary_directory / "docker",
            """
            service="${@: -1}"
            printf '%s container log\n' "$service"
            while true; do sleep 1; done
            """,
        )

        port = self.available_port()
        return os.environ | {
            "COMMAND_TRACE": str(root / "command-trace.log"),
            "NORVII_WEB_PORT": str(port),
            "PATH": f"{binary_directory}:{os.environ['PATH']}",
        }

    @staticmethod
    def write_executable(path: Path, body: str) -> None:
        path.write_text(
            f"#!/usr/bin/env bash\nset -euo pipefail\n{textwrap.dedent(body).strip()}\n",
            encoding="utf-8",
        )
        path.chmod(path.stat().st_mode | stat.S_IXUSR)

    @staticmethod
    def available_port() -> int:
        with socket.socket() as server:
            server.bind(("127.0.0.1", 0))
            return int(server.getsockname()[1])


if __name__ == "__main__":
    unittest.main()
