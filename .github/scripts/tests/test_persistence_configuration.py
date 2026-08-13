import json
import subprocess
import tempfile
import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
COMPOSE_FILE = REPOSITORY_ROOT / "infra" / "compose.yaml"


class PersistenceConfigurationTest(unittest.TestCase):
    def test_default_environment_contains_exactly_the_required_services(self) -> None:
        configuration = self.render_configuration()

        self.assertEqual({"postgres", "neo4j"}, set(configuration["services"]))

    def test_images_are_pinned_by_release_and_multi_architecture_digest(self) -> None:
        services = self.render_configuration()["services"]

        self.assertEqual(
            "pgvector/pgvector:0.8.6-pg18-trixie@sha256:1963bc48febf543433baa1ce3edcc6cc08154de722e22495f86681cc9a849026",
            services["postgres"]["image"],
        )
        self.assertEqual(
            "neo4j:2026.06.0-community-trixie@sha256:42fd5b9ead4dd4211f6f91bd831c358e4e2117367d04633fbf88682ca4792b30",
            services["neo4j"]["image"],
        )

    def test_each_service_has_authenticated_health_and_bounded_resources(self) -> None:
        services = self.render_configuration()["services"]

        postgres_health = services["postgres"]["healthcheck"]["test"]
        neo4j_health = services["neo4j"]["healthcheck"]["test"]
        self.assertIn("psql", " ".join(postgres_health))
        self.assertIn("SELECT 1", " ".join(postgres_health))
        self.assertIn("cypher-shell", " ".join(neo4j_health))
        self.assertIn("RETURN 1", " ".join(neo4j_health))
        self.assertEqual(536_870_912, int(services["postgres"]["mem_limit"]))
        self.assertEqual(1_610_612_736, int(services["neo4j"]["mem_limit"]))

    def test_each_store_owns_a_distinct_named_data_volume(self) -> None:
        configuration = self.render_configuration()

        self.assertEqual(
            {"norvii_postgres_data", "norvii_neo4j_data"},
            set(configuration["volumes"]),
        )
        self.assertEqual(
            ["norvii_postgres_data"],
            [mount["source"] for mount in configuration["services"]["postgres"]["volumes"]],
        )
        self.assertEqual(
            ["norvii_neo4j_data"],
            [mount["source"] for mount in configuration["services"]["neo4j"]["volumes"]],
        )
        self.assertEqual(["/logs:size=64m"], configuration["services"]["neo4j"]["tmpfs"])

    def render_configuration(self) -> dict[str, object]:
        with tempfile.NamedTemporaryFile(mode="w", encoding="utf-8") as environment_file:
            environment_file.write(
                "\n".join(
                    [
                        "NORVII_POSTGRES_PORT=5432",
                        "NORVII_POSTGRES_DATABASE=norvii",
                        "NORVII_POSTGRES_USER=norvii",
                        "NORVII_POSTGRES_PASSWORD=integration-postgres-secret",
                        "NORVII_NEO4J_HTTP_PORT=7474",
                        "NORVII_NEO4J_BOLT_PORT=7687",
                        "NORVII_NEO4J_USER=neo4j",
                        "NORVII_NEO4J_PASSWORD=integration-neo4j-secret",
                    ]
                )
            )
            environment_file.flush()
            result = subprocess.run(
                [
                    "docker",
                    "compose",
                    "--env-file",
                    environment_file.name,
                    "-f",
                    str(COMPOSE_FILE),
                    "config",
                    "--format",
                    "json",
                ],
                check=False,
                capture_output=True,
                text=True,
            )

        self.assertEqual(0, result.returncode, result.stderr)
        return json.loads(result.stdout)


if __name__ == "__main__":
    unittest.main()
