from __future__ import annotations

import os
import smtplib
import sys
from dataclasses import dataclass
from email.message import EmailMessage
from typing import Callable, Mapping, Protocol, cast


class EmailNotificationError(RuntimeError):
    """Raised when a workflow failure notification cannot be sent."""


class SmtpClient(Protocol):
    def __enter__(self) -> SmtpClient: ...

    def __exit__(self, *args: object) -> None: ...

    def login(self, user: str, password: str) -> object: ...

    def send_message(self, message: EmailMessage) -> object: ...


SmtpFactory = Callable[..., SmtpClient]


@dataclass(frozen=True)
class FailureEmailConfig:
    smtp_username: str
    smtp_password: str
    recipient: str
    repository: str
    workflow: str
    branch: str
    commit: str
    run_url: str

    @classmethod
    def from_environment(cls, environment: Mapping[str, str]) -> FailureEmailConfig:
        names = (
            "SMTP_USERNAME",
            "SMTP_PASSWORD",
            "CI_FAILURE_EMAIL",
            "GITHUB_REPOSITORY",
            "CI_WORKFLOW",
            "GIT_BRANCH",
            "GIT_COMMIT",
            "CI_RUN_URL",
        )
        missing = [name for name in names if not environment.get(name, "").strip()]
        if missing:
            raise EmailNotificationError(
                f"Missing required environment values: {', '.join(missing)}"
            )

        return cls(
            smtp_username=environment["SMTP_USERNAME"],
            smtp_password=environment["SMTP_PASSWORD"],
            recipient=environment["CI_FAILURE_EMAIL"],
            repository=environment["GITHUB_REPOSITORY"],
            workflow=environment["CI_WORKFLOW"],
            branch=environment["GIT_BRANCH"],
            commit=environment["GIT_COMMIT"],
            run_url=environment["CI_RUN_URL"],
        )


class FailureEmailNotification:
    def __init__(
        self,
        config: FailureEmailConfig,
        smtp_factory: SmtpFactory | None = None,
    ) -> None:
        self._config = config
        self._smtp_factory = smtp_factory or self._open_smtp

    def send(self) -> None:
        message = self._create_message()
        try:
            with self._smtp_factory("smtp.gmail.com", 465, timeout=30) as smtp:
                smtp.login(self._config.smtp_username, self._config.smtp_password)
                smtp.send_message(message)
        except (OSError, smtplib.SMTPException) as error:
            raise EmailNotificationError(
                f"Unable to send CI failure email: {error}"
            ) from error

        print(f"CI failure notification sent to {self._config.recipient}.")

    def _create_message(self) -> EmailMessage:
        short_commit = self._config.commit[:12]
        message = EmailMessage()
        message["Subject"] = f"[Norvii CI] Failure on {self._config.branch}"
        message["From"] = self._config.smtp_username
        message["To"] = self._config.recipient
        message.set_content(
            "\n".join(
                (
                    "The Norvii continuous integration workflow failed.",
                    "",
                    f"Repository: {self._config.repository}",
                    f"Workflow: {self._config.workflow}",
                    f"Branch: {self._config.branch}",
                    f"Commit: {short_commit}",
                    f"Run: {self._config.run_url}",
                )
            )
        )
        return message

    @staticmethod
    def _open_smtp(host: str, port: int, timeout: int) -> SmtpClient:
        return cast(SmtpClient, smtplib.SMTP_SSL(host, port, timeout=timeout))


class FailureEmailApplication:
    def run(self) -> int:
        try:
            config = FailureEmailConfig.from_environment(os.environ)
            FailureEmailNotification(config).send()
        except EmailNotificationError as error:
            print(f"error: {error}", file=sys.stderr)
            return 1
        return 0


if __name__ == "__main__":
    raise SystemExit(FailureEmailApplication().run())
