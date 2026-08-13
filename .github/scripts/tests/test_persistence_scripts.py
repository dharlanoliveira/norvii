import os
import stat
import subprocess
import tempfile
import textwrap
import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
HEALTH_SCRIPT = REPOSITORY_ROOT / "infra" / "scripts" / "inspect-health.sh"


class PersistenceHealthScriptTest(unittest.TestCase):
    def test_reports_both_services_healthy_without_disclosing_environment(self) -> None:
        result = self.run_script(neo4j_health="healthy")

        self.assertEqual(0, result.returncode, result.stderr)
        self.assertIn("PostgreSQL is healthy.", result.stdout)
        self.assertIn("Neo4j is healthy.", result.stdout)
        self.assertNotIn("test-postgres-secret", result.stdout + result.stderr)
        self.assertNotIn("test-neo4j-secret", result.stdout + result.stderr)

    def test_unhealthy_service_returns_failure_with_bounded_diagnostic_command(self) -> None:
        result = self.run_script(neo4j_health="unhealthy")

        self.assertNotEqual(0, result.returncode)
        self.assertIn("Neo4j is unhealthy", result.stderr)
        self.assertIn("logs --tail 100 neo4j", result.stderr)
        self.assertNotIn("test-neo4j-secret", result.stdout + result.stderr)

    def run_script(self, *, neo4j_health: str) -> subprocess.CompletedProcess[str]:
        with tempfile.TemporaryDirectory() as temporary_directory:
            directory = Path(temporary_directory)
            environment_file = directory / ".env"
            environment_file.write_text(
                "NORVII_POSTGRES_PASSWORD=test-postgres-secret\n"
                "NORVII_NEO4J_PASSWORD=test-neo4j-secret\n",
                encoding="utf-8",
            )
            docker = directory / "docker"
            docker.write_text(
                textwrap.dedent(
                    f"""\
                    #!/usr/bin/env bash
                    set -eu
                    service="${{@: -1}}"
                    if [[ "$service" == "neo4j" ]]; then
                      printf '%s\\n' '{neo4j_health}'
                    else
                      printf '%s\\n' 'healthy'
                    fi
                    """
                ),
                encoding="utf-8",
            )
            docker.chmod(docker.stat().st_mode | stat.S_IXUSR)
            environment = os.environ | {"PATH": f"{directory}:{os.environ['PATH']}"}
            return subprocess.run(
                ["bash", str(HEALTH_SCRIPT), str(environment_file)],
                cwd=REPOSITORY_ROOT,
                env=environment,
                check=False,
                capture_output=True,
                text=True,
            )


if __name__ == "__main__":
    unittest.main()
