import unittest
from pathlib import Path


REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = REPOSITORY_ROOT / ".github" / "workflows" / "ci.yml"


class SonarMonorepoConfigurationTest(unittest.TestCase):
    def test_workflow_maps_each_production_module_to_its_project_key(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")

        expected_entries = {
            "apps/web": "SONAR_WEB_PROJECT_KEY",
            "apps/api": "SONAR_API_PROJECT_KEY",
            "apps/ingestion": "SONAR_INGESTION_PROJECT_KEY",
        }
        for module_path, project_key_variable in expected_entries.items():
            with self.subTest(module_path=module_path):
                self.assertIn(f"project_path: {module_path}", workflow)
                self.assertIn(
                    f"project_key: ${{{{ vars.{project_key_variable} }}}}", workflow
                )

        self.assertIn("projectBaseDir: ${{ matrix.project_path }}", workflow)
        self.assertIn('make -C "${{ matrix.project_path }}" coverage', workflow)
        self.assertIn("name: sonar-coverage-${{ matrix.project_name }}", workflow)
        self.assertNotIn("vars.SONAR_PROJECT_KEY", workflow)

    def test_each_production_module_owns_a_sonar_configuration(self) -> None:
        expected_properties = {
            "apps/web": ("sonar.sources=src", "sonar.javascript.lcov.reportPaths="),
            "apps/api": ("sonar.sources=.", "sonar.go.coverage.reportPaths="),
            "apps/ingestion": (
                "sonar.sources=src",
                "sonar.python.coverage.reportPaths=",
            ),
        }
        for module_path, required_properties in expected_properties.items():
            with self.subTest(module_path=module_path):
                properties_path = REPOSITORY_ROOT / module_path / "sonar-project.properties"
                properties = properties_path.read_text(encoding="utf-8")
                for required_property in required_properties:
                    self.assertIn(required_property, properties)
                self.assertIn("sonar.tests=", properties)

        self.assertFalse((REPOSITORY_ROOT / "sonar-project.properties").exists())

    def test_web_coverage_generates_the_lcov_report_consumed_by_sonar(self) -> None:
        vitest_config = (REPOSITORY_ROOT / "apps/web/vitest.config.ts").read_text(
            encoding="utf-8"
        )

        self.assertIn('reporter: ["text", "lcov"]', vitest_config)


if __name__ == "__main__":
    unittest.main()
