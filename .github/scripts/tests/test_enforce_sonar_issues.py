from __future__ import annotations

import json
import sys
import unittest
from pathlib import Path
from urllib.request import Request

SCRIPT_DIRECTORY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPT_DIRECTORY))

from enforce_sonar_issues import (  # noqa: E402
    SonarIssuePolicy,
    SonarIssuePolicyConfig,
    SonarPolicyError,
)


class FakeHttpResponse:
    def __init__(self, payload: dict[str, object]) -> None:
        self._payload = payload

    def __enter__(self) -> FakeHttpResponse:
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def read(self) -> bytes:
        return json.dumps(self._payload).encode("utf-8")


class RecordingHttpOpener:
    def __init__(self, payload: dict[str, object]) -> None:
        self._payload = payload
        self.request: Request | None = None

    def __call__(self, request: Request, timeout: float) -> FakeHttpResponse:
        self.request = request
        self.timeout = timeout
        return FakeHttpResponse(self._payload)


class SonarIssuePolicyConfigTests(unittest.TestCase):
    def test_requires_every_credential(self) -> None:
        with self.assertRaisesRegex(SonarPolicyError, "SONAR_TOKEN"):
            SonarIssuePolicyConfig.from_environment(
                {
                    "SONAR_HOST_URL": "https://sonarcloud.io",
                    "SONAR_PROJECT_KEY": "norvii",
                }
            )


class SonarIssuePolicyTests(unittest.TestCase):
    def test_accepts_zero_open_issues(self) -> None:
        opener = RecordingHttpOpener({"total": 0})
        policy = SonarIssuePolicy(self._config(), opener)

        policy.assert_no_open_issues()

    def test_rejects_any_open_issue(self) -> None:
        opener = RecordingHttpOpener({"total": 1})
        policy = SonarIssuePolicy(self._config(), opener)

        with self.assertRaisesRegex(SonarPolicyError, "1 open issue"):
            policy.assert_no_open_issues()

    def test_scopes_query_to_pull_request(self) -> None:
        opener = RecordingHttpOpener({"total": 0})
        config = self._config(pull_request="42")

        SonarIssuePolicy(config, opener).assert_no_open_issues()

        request = opener.request
        self.assertIsNotNone(request)
        assert request is not None
        self.assertIn("pullRequest=42", request.full_url)
        self.assertEqual(request.get_header("Authorization"), "Bearer token")

    def _config(self, pull_request: str | None = None) -> SonarIssuePolicyConfig:
        return SonarIssuePolicyConfig(
            host_url="https://sonarcloud.io",
            project_key="dharlanoliveira_norvii",
            token="token",
            pull_request=pull_request,
        )


if __name__ == "__main__":
    unittest.main()
