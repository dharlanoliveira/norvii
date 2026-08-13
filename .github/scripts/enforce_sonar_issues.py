from __future__ import annotations

import json
import os
import sys
from dataclasses import dataclass
from typing import Mapping, Protocol, cast
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


class SonarPolicyError(RuntimeError):
    """Raised when the Sonar issue policy cannot be evaluated."""


class HttpResponse(Protocol):
    def __enter__(self) -> HttpResponse: ...

    def __exit__(self, *args: object) -> None: ...

    def read(self) -> bytes: ...


class HttpOpener(Protocol):
    def __call__(self, request: Request, timeout: float) -> HttpResponse: ...


@dataclass(frozen=True)
class SonarIssuePolicyConfig:
    host_url: str
    project_key: str
    token: str
    pull_request: str | None = None

    @classmethod
    def from_environment(cls, environment: Mapping[str, str]) -> SonarIssuePolicyConfig:
        required = ("SONAR_HOST_URL", "SONAR_PROJECT_KEY", "SONAR_TOKEN")
        missing = [name for name in required if not environment.get(name, "").strip()]
        if missing:
            raise SonarPolicyError(
                f"Missing required environment values: {', '.join(missing)}"
            )

        pull_request = environment.get("SONAR_PULL_REQUEST", "").strip() or None
        return cls(
            host_url=environment["SONAR_HOST_URL"].rstrip("/"),
            project_key=environment["SONAR_PROJECT_KEY"].strip(),
            token=environment["SONAR_TOKEN"],
            pull_request=pull_request,
        )


class SonarIssuePolicy:
    def __init__(
        self, config: SonarIssuePolicyConfig, opener: HttpOpener | None = None
    ) -> None:
        self._config = config
        self._opener = opener or self._open_url

    def assert_no_open_issues(self) -> None:
        issue_count = self._open_issue_count()
        if issue_count > 0:
            raise SonarPolicyError(
                f"SonarQube Cloud reports {issue_count} open issue(s) for "
                f"{self._config.project_key}. Resolve every issue before merging."
            )

        print(
            f"SonarQube Cloud reports zero open issues for {self._config.project_key}."
        )

    def _open_issue_count(self) -> int:
        query = {
            "componentKeys": self._config.project_key,
            "resolved": "false",
            "ps": "1",
        }
        if self._config.pull_request:
            query["pullRequest"] = self._config.pull_request

        request = Request(
            f"{self._config.host_url}/api/issues/search?{urlencode(query)}",
            headers={
                "Accept": "application/json",
                "Authorization": f"Bearer {self._config.token}",
            },
        )

        try:
            with self._opener(request, timeout=30.0) as response:
                payload = json.loads(response.read())
        except (HTTPError, URLError, TimeoutError, json.JSONDecodeError) as error:
            raise SonarPolicyError(
                f"Unable to retrieve SonarQube Cloud issues: {error}"
            ) from error

        total = payload.get("total")
        if not isinstance(total, int):
            raise SonarPolicyError("SonarQube Cloud returned an invalid issue count.")
        return total

    @staticmethod
    def _open_url(request: Request, timeout: float) -> HttpResponse:
        return cast(HttpResponse, urlopen(request, timeout=timeout))


class SonarIssuePolicyApplication:
    def run(self) -> int:
        try:
            config = SonarIssuePolicyConfig.from_environment(os.environ)
            SonarIssuePolicy(config).assert_no_open_issues()
        except SonarPolicyError as error:
            print(f"error: {error}", file=sys.stderr)
            return 1
        return 0


if __name__ == "__main__":
    raise SystemExit(SonarIssuePolicyApplication().run())
