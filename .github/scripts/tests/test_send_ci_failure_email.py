from __future__ import annotations

import sys
import unittest
from email.message import EmailMessage
from pathlib import Path

SCRIPT_DIRECTORY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPT_DIRECTORY))

from send_ci_failure_email import (  # noqa: E402
    EmailNotificationError,
    FailureEmailConfig,
    FailureEmailNotification,
)


class RecordingSmtpClient:
    def __init__(self) -> None:
        self.login_credentials: tuple[str, str] | None = None
        self.message: EmailMessage | None = None

    def __enter__(self) -> RecordingSmtpClient:
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def login(self, user: str, password: str) -> None:
        self.login_credentials = (user, password)

    def send_message(self, message: EmailMessage) -> None:
        self.message = message


class RecordingSmtpFactory:
    def __init__(self) -> None:
        self.client = RecordingSmtpClient()
        self.connection: tuple[str, int, int] | None = None

    def __call__(self, host: str, port: int, timeout: int) -> RecordingSmtpClient:
        self.connection = (host, port, timeout)
        return self.client


class FailureEmailConfigTests(unittest.TestCase):
    def test_requires_recipient(self) -> None:
        environment = self._environment()
        environment.pop("CI_FAILURE_EMAIL")

        with self.assertRaisesRegex(EmailNotificationError, "CI_FAILURE_EMAIL"):
            FailureEmailConfig.from_environment(environment)

    def _environment(self) -> dict[str, str]:
        return {
            "SMTP_USERNAME": "sender@example.com",
            "SMTP_PASSWORD": "password",
            "CI_FAILURE_EMAIL": "recipient@example.com",
            "GITHUB_REPOSITORY": "dharlanoliveira/norvii",
            "CI_WORKFLOW": "CI",
            "GIT_BRANCH": "main",
            "GIT_COMMIT": "0123456789abcdef",
            "CI_RUN_URL": "https://github.com/dharlanoliveira/norvii/actions/runs/1",
        }


class FailureEmailNotificationTests(unittest.TestCase):
    def test_sends_failure_context_over_gmail_smtp(self) -> None:
        factory = RecordingSmtpFactory()
        config = FailureEmailConfig.from_environment(self._environment())

        FailureEmailNotification(config, factory).send()

        self.assertEqual(factory.connection, ("smtp.gmail.com", 465, 30))
        self.assertEqual(
            factory.client.login_credentials, ("sender@example.com", "password")
        )
        message = factory.client.message
        self.assertIsNotNone(message)
        assert message is not None
        self.assertEqual(message["To"], "recipient@example.com")
        self.assertIn("0123456789ab", message.get_content())
        self.assertNotIn("password", message.as_string())

    def _environment(self) -> dict[str, str]:
        return {
            "SMTP_USERNAME": "sender@example.com",
            "SMTP_PASSWORD": "password",
            "CI_FAILURE_EMAIL": "recipient@example.com",
            "GITHUB_REPOSITORY": "dharlanoliveira/norvii",
            "CI_WORKFLOW": "CI",
            "GIT_BRANCH": "main",
            "GIT_COMMIT": "0123456789abcdef",
            "CI_RUN_URL": "https://github.com/dharlanoliveira/norvii/actions/runs/1",
        }


if __name__ == "__main__":
    unittest.main()
