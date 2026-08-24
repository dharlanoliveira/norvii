#!/usr/bin/env python3
"""Manage the complete Norvii local development environment."""

from __future__ import annotations

import argparse
import json
import os
import re
import shlex
import shutil
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import TYPE_CHECKING, TextIO

if TYPE_CHECKING:
    from collections.abc import Iterable

PID_IDENTITY_FIELD_COUNT = 2
MAX_TCP_PORT = 65_535
HTTP_OK = 200
ENVIRONMENT_ASSIGNMENT = re.compile(r"^([A-Za-z_][A-Za-z0-9_]*)=(.*)$")
INITIAL_CORPORA = {
    "en": "10000000-0000-4000-8000-000000000002",
    "pt": "10000000-0000-4000-8000-000000000001",
}


class LocalEnvironmentError(RuntimeError):
    """Report an actionable local environment failure."""


class ComponentCommandError(LocalEnvironmentError):
    """Report a failed component command and its diagnostic log."""

    def __init__(self, component: str, log_path: Path, repository_root: Path) -> None:
        relative_log = log_path.relative_to(repository_root)
        super().__init__(f"{component} failed; inspect {relative_log}.")


@dataclass(frozen=True)
class RepositoryLayout:
    """Own validated paths for one Norvii checkout."""

    root: Path

    @property
    def environment_file(self) -> Path:
        """Return the ignored local environment file."""
        return self.root / "infra" / ".env"

    @property
    def compose_file(self) -> Path:
        """Return the canonical local Compose file."""
        return self.root / "infra" / "compose.yaml"

    @property
    def web_directory(self) -> Path:
        """Return the production web module directory."""
        return self.root / "apps" / "web"

    @property
    def api_directory(self) -> Path:
        """Return the Go API module directory."""
        return self.root / "apps" / "api"

    @property
    def ingestion_directory(self) -> Path:
        """Return the Python ingestion module directory."""
        return self.root / "apps" / "ingestion"

    @property
    def agent_directory(self) -> Path:
        """Return the Python LangGraph agent module directory."""
        return self.root / "apps" / "agent"

    @property
    def environment_runner(self) -> Path:
        """Return the non-evaluating environment wrapper."""
        return self.root / "infra" / "scripts" / "run-with-environment.py"

    @property
    def log_directory(self) -> Path:
        """Return the component log and lifecycle state directory."""
        return self.root / ".log"

    def log(self, component: str) -> Path:
        """Return the log file owned by a component."""
        return self.log_directory / f"{component}.log"

    def pid(self, component: str) -> Path:
        """Return the managed process identity file for a component."""
        return self.log_directory / f"{component}.pid"

    def ready_marker(self, component: str) -> Path:
        """Return the initialization marker for a one-shot component."""
        return self.log_directory / f"{component}.ready"

    def validate_root(self) -> None:
        """Reject broad or unrelated roots before creating runtime files."""
        required_identity_paths = (
            self.root / "Makefile",
            self.compose_file,
            self.web_directory / "package.json",
        )
        if self.root == Path(self.root.anchor) or any(
            not path.is_file() for path in required_identity_paths
        ):
            raise LocalEnvironmentError(f"{self.root} is not a Norvii repository root.")
        if self.log_directory.is_symlink() or (
            self.log_directory.exists() and not self.log_directory.is_dir()
        ):
            raise LocalEnvironmentError(".log must be a local directory, not a link or file.")

    def validate_environment(self) -> None:
        """Validate local configuration without exposing credentials."""
        if not self.environment_file.is_file():
            raise LocalEnvironmentError("Missing infra/.env.")
        environment = self.environment_file.read_text(encoding="utf-8")
        if "REPLACE_WITH_LOCAL_" in environment:
            raise LocalEnvironmentError(
                "infra/.env still contains password markers; replace both before startup."
            )

    def environment_values(self) -> dict[str, str]:
        """Read the same strict dotenv subset accepted by the process wrapper."""
        values: dict[str, str] = {}
        for line_number, raw_line in enumerate(
            self.environment_file.read_text(encoding="utf-8").splitlines(), start=1
        ):
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            match = ENVIRONMENT_ASSIGNMENT.fullmatch(line)
            if match is None:
                raise LocalEnvironmentError(
                    f"infra/.env has an invalid assignment on line {line_number}."
                )
            name, raw_value = match.groups()
            try:
                tokens = shlex.split(raw_value, comments=True, posix=True)
            except ValueError as error:
                raise LocalEnvironmentError(
                    f"infra/.env has invalid quoting on line {line_number}."
                ) from error
            if len(tokens) > 1:
                raise LocalEnvironmentError(
                    f"infra/.env has an unquoted value on line {line_number}."
                )
            values[name] = tokens[0] if tokens else ""
        return values


class ComponentLogger:
    """Append lifecycle output to component-owned files."""

    COMPONENTS = ("bootstrap", "api", "agent", "ingestion", "web", "postgres", "neo4j")

    def __init__(self, layout: RepositoryLayout) -> None:
        self._layout = layout

    def initialize(self) -> None:
        """Create the runtime directory and every component log."""
        self._layout.log_directory.mkdir(parents=True, exist_ok=True)
        for component in self.COMPONENTS:
            log_path = self._layout.log(component)
            if log_path.is_symlink():
                raise LocalEnvironmentError(f"Refusing linked component log .log/{log_path.name}.")
            log_path.touch(exist_ok=True)

    def open(self, component: str) -> TextIO:
        """Open a component log and append a timestamped lifecycle boundary."""
        log = self._layout.log(component).open(mode="a", encoding="utf-8")
        timestamp = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())
        log.write(f"\n[{timestamp}] Norvii local environment\n")
        log.flush()
        return log


class CommandRunner:
    """Run foreground component commands with isolated logs."""

    def __init__(self, layout: RepositoryLayout, logger: ComponentLogger) -> None:
        self._layout = layout
        self._logger = logger

    def run(self, component: str, command: list[str]) -> None:
        """Run an internal command and raise with its component log on failure."""
        with self._logger.open(component) as log:
            completed = subprocess.run(  # noqa: S603 - repository-owned argv only
                command,
                cwd=self._layout.root,
                check=False,
                stdout=log,
                stderr=subprocess.STDOUT,
                env=os.environ,
                text=True,
            )
        if completed.returncode != 0:
            raise ComponentCommandError(
                component.capitalize(), self._layout.log(component), self._layout.root
            )

    def capture(self, component: str, command: list[str]) -> str:
        """Run an internal command while returning and logging its standard output."""
        completed = subprocess.run(  # noqa: S603 - repository-owned argv only
            command,
            cwd=self._layout.root,
            check=False,
            capture_output=True,
            env=os.environ,
            text=True,
        )
        with self._logger.open(component) as log:
            log.write(completed.stdout)
            log.write(completed.stderr)
        if completed.returncode != 0:
            raise ComponentCommandError(
                component.capitalize(), self._layout.log(component), self._layout.root
            )
        return completed.stdout


class ManagedProcess:
    """Start and safely stop a process group recorded with its kernel start time."""

    def __init__(
        self,
        component: str,
        command: list[str],
        expected_command_terms: tuple[str, ...],
        layout: RepositoryLayout,
        logger: ComponentLogger,
    ) -> None:
        self._component = component
        self._command = command
        self._expected_command_terms = expected_command_terms
        self._layout = layout
        self._logger = logger

    def start(self) -> bool:
        """Start the process group unless its recorded identity is still active."""
        if self.is_running():
            return False
        with self._logger.open(self._component) as log:
            process = subprocess.Popen(  # noqa: S603 - repository-owned argv only
                self._command,
                cwd=self._layout.root,
                stdin=subprocess.DEVNULL,
                stdout=log,
                stderr=subprocess.STDOUT,
                env=os.environ,
                start_new_session=True,
                text=True,
            )
        time.sleep(0.1)
        if process.poll() is not None:
            raise ComponentCommandError(
                self._component.capitalize(),
                self._layout.log(self._component),
                self._layout.root,
            )
        _, start_time = self._read_process_stat(process.pid)
        self._layout.pid(self._component).write_text(
            f"{process.pid} {start_time}\n", encoding="utf-8"
        )
        return True

    def is_running(self) -> bool:
        """Return whether the recorded kernel process identity is still active."""
        identity = self._read_identity()
        if identity is None:
            return False
        process_id, recorded_start_time = identity
        try:
            _, current_start_time = self._read_process_stat(process_id)
        except (FileNotFoundError, ProcessLookupError):
            return False
        return current_start_time == recorded_start_time

    def stop(self) -> None:
        """Terminate only the exact process group started by Norvii."""
        identity = self._read_identity()
        if identity is None:
            return
        process_id, recorded_start_time = identity
        try:
            _, current_start_time = self._read_process_stat(process_id)
        except (FileNotFoundError, ProcessLookupError):
            self._layout.pid(self._component).unlink(missing_ok=True)
            return
        if current_start_time != recorded_start_time or os.getpgid(process_id) != process_id:
            raise LocalEnvironmentError(
                f"Refusing to stop unverified {self._component} PID {process_id}."
            )

        os.killpg(process_id, signal.SIGTERM)
        for _ in range(50):
            try:
                state, _ = self._read_process_stat(process_id)
            except FileNotFoundError:
                self._layout.pid(self._component).unlink(missing_ok=True)
                return
            if state == "Z":
                self._layout.pid(self._component).unlink(missing_ok=True)
                return
            time.sleep(0.1)
        raise LocalEnvironmentError(
            f"{self._component.capitalize()} did not stop; inspect its process before retrying."
        )

    def _read_identity(self) -> tuple[int, int] | None:
        try:
            values = self._layout.pid(self._component).read_text(encoding="utf-8").split()
        except FileNotFoundError:
            return None
        if len(values) == 1 and values[0].isdigit():
            process_id = int(values[0])
            self._validate_legacy_process(process_id)
            _, start_time = self._read_process_stat(process_id)
            return process_id, start_time
        if len(values) != PID_IDENTITY_FIELD_COUNT or not all(value.isdigit() for value in values):
            raise LocalEnvironmentError(f"Invalid {self._component} PID file.")
        return int(values[0]), int(values[1])

    def _validate_legacy_process(self, process_id: int) -> None:
        command_line = (
            Path(f"/proc/{process_id}/cmdline")
            .read_bytes()
            .replace(b"\0", b" ")
            .decode("utf-8", errors="replace")
        )
        if os.getpgid(process_id) != process_id or not all(
            term in command_line for term in self._expected_command_terms
        ):
            raise LocalEnvironmentError(
                f"Refusing unverified legacy {self._component} PID {process_id}."
            )

    @staticmethod
    def _read_process_stat(process_id: int) -> tuple[str, int]:
        stat = Path(f"/proc/{process_id}/stat").read_text(encoding="utf-8")
        fields_after_name = stat[stat.rfind(")") + 2 :].split()
        return fields_after_name[0], int(fields_after_name[19])


class LocalEnvironmentManager:
    """Coordinate persistence, initialization, web, status, and shutdown."""

    def __init__(self, layout: RepositoryLayout) -> None:
        self._layout = layout
        self._logger = ComponentLogger(layout)
        self._runner = CommandRunner(layout, self._logger)
        web_port = os.environ.get("NORVII_WEB_PORT", "5173")
        if not web_port.isdigit() or not 1 <= int(web_port) <= MAX_TCP_PORT:
            raise LocalEnvironmentError("NORVII_WEB_PORT must be a valid TCP port.")
        self._web_url = f"http://127.0.0.1:{web_port}"
        environment = layout.environment_values() if layout.environment_file.is_file() else {}
        api_port = environment.get("NORVII_API_PORT", "8080")
        if not api_port.isdigit() or not 1 <= int(api_port) <= MAX_TCP_PORT:
            raise LocalEnvironmentError("NORVII_API_PORT must be a valid TCP port.")
        self._api_health_url = f"http://127.0.0.1:{api_port}/healthz"
        self._api_base_url = f"http://127.0.0.1:{api_port}/api/v1"
        agent_port = environment.get("NORVII_AGENT_PORT", "8090")
        if not agent_port.isdigit() or not 1 <= int(agent_port) <= MAX_TCP_PORT:
            raise LocalEnvironmentError("NORVII_AGENT_PORT must be a valid TCP port.")
        self._agent_health_url = f"http://127.0.0.1:{agent_port}/healthz"
        initial_timeout = os.environ.get(
            "NORVII_INITIAL_INGESTION_TIMEOUT_SECONDS",
            environment.get("NORVII_INITIAL_INGESTION_TIMEOUT_SECONDS", "90"),
        )
        if not initial_timeout.isdigit() or int(initial_timeout) <= 0:
            raise LocalEnvironmentError(
                "NORVII_INITIAL_INGESTION_TIMEOUT_SECONDS must be a positive integer."
            )
        self._initial_ingestion_timeout = int(initial_timeout)
        compose = [
            "docker",
            "compose",
            "--env-file",
            str(layout.environment_file),
            "-f",
            str(layout.compose_file),
        ]
        self._processes = {
            "postgres": ManagedProcess(
                "postgres",
                [
                    *compose,
                    "logs",
                    "--follow",
                    "--tail",
                    "200",
                    "--no-color",
                    "postgres",
                ],
                ("docker", "compose", "postgres"),
                layout,
                self._logger,
            ),
            "neo4j": ManagedProcess(
                "neo4j",
                [*compose, "logs", "--follow", "--tail", "200", "--no-color", "neo4j"],
                ("docker", "compose", "neo4j"),
                layout,
                self._logger,
            ),
            "api": ManagedProcess(
                "api",
                [
                    sys.executable,
                    str(layout.environment_runner),
                    str(layout.environment_file),
                    "go",
                    "-C",
                    str(layout.api_directory),
                    "run",
                    "./cmd/server",
                ],
                ("go", "cmd/server"),
                layout,
                self._logger,
            ),
            "agent": ManagedProcess(
                "agent",
                [
                    sys.executable,
                    str(layout.environment_runner),
                    str(layout.environment_file),
                    "uv",
                    "run",
                    "--directory",
                    str(layout.agent_directory),
                    "norvii-agent",
                ],
                ("uv", "norvii-agent"),
                layout,
                self._logger,
            ),
            "ingestion": ManagedProcess(
                "ingestion",
                [
                    sys.executable,
                    str(layout.environment_runner),
                    str(layout.environment_file),
                    "uv",
                    "run",
                    "--directory",
                    str(layout.ingestion_directory),
                    "norvii-ingestion-worker",
                ],
                ("uv", "norvii-ingestion-worker"),
                layout,
                self._logger,
            ),
            "web": ManagedProcess(
                "web",
                [
                    "npm",
                    "--prefix",
                    str(layout.web_directory),
                    "run",
                    "dev",
                    "--",
                    "--host",
                    "127.0.0.1",
                    "--port",
                    web_port,
                    "--strictPort",
                ],
                ("npm", "dev"),
                layout,
                self._logger,
            ),
        }

    def start(self) -> None:
        """Start and verify every currently executable local component."""
        self._prepare(required_commands=("docker", "go", "make", "npm", "uv"))
        started_components: list[str] = []
        try:
            self._runner.run("bootstrap", self._make("persistence-up"))
            self._runner.run("api", self._make("persistence-migrate"))
            self._runner.run("api", self._make("persistence-verify-api"))
            self._runner.run("ingestion", self._make("persistence-verify-ingestion"))
            if not self._processes["web"].is_running():
                self._runner.run("web", ["npm", "--prefix", str(self._layout.web_directory), "ci"])
            started_components.extend(
                component
                for component in ("postgres", "neo4j", "agent", "api", "ingestion", "web")
                if self._processes[component].start()
            )
            self._wait_for_agent()
            self._wait_for_api()
            self._wait_for_web()
            initial_states = self._wait_for_initial_sources()
            snapshot_status = self._initialize_initial_snapshots(initial_states)
        except LocalEnvironmentError:
            self._stop_managed_processes(reversed(started_components))
            raise
        states = ", ".join(
            f"{language}={initial_states[language]}" for language in sorted(initial_states)
        )
        print(
            f"Norvii is ready\nWeb: {self._web_url}\n"
            f"Initial sources: {states}\nSnapshots: {snapshot_status}\n"
            f"Logs: {self._layout.log_directory}"
        )

    def status(self) -> None:
        """Report persistence health and managed component lifecycle state."""
        self._prepare(required_commands=("make",))
        health = self._runner.capture("bootstrap", self._make("persistence-health"))
        print(health, end="")
        print("Web is running." if self._processes["web"].is_running() else "Web is stopped.")
        for component in ("api", "agent", "ingestion"):
            state = "running" if self._processes[component].is_running() else "stopped"
            label = {"api": "API", "agent": "Agent", "ingestion": "Ingestion"}[component]
            print(f"{label} is {state}.")

    def stop(self) -> None:
        """Stop managed processes and persistence without deleting stored data."""
        self._layout.validate_root()
        self._logger.initialize()
        self._stop_managed_processes(("web", "ingestion", "api", "agent", "neo4j", "postgres"))
        if self._layout.environment_file.exists():
            self._runner.run("bootstrap", self._make("persistence-stop"))
        for component in ("api", "agent", "ingestion"):
            self._layout.ready_marker(component).unlink(missing_ok=True)
        print("Norvii is stopped.")

    def _prepare(self, *, required_commands: tuple[str, ...]) -> None:
        self._layout.validate_root()
        self._logger.initialize()
        self._layout.validate_environment()
        for command in required_commands:
            if shutil.which(command) is None:
                raise LocalEnvironmentError(f"Required command '{command}' is unavailable.")

    def _make(self, target: str) -> list[str]:
        return ["make", "-C", str(self._layout.root), target]

    def _wait_for_web(self) -> None:
        timeout = int(os.environ.get("NORVII_WEB_START_TIMEOUT_SECONDS", "30"))
        if timeout <= 0:
            raise LocalEnvironmentError(
                "NORVII_WEB_START_TIMEOUT_SECONDS must be a positive integer."
            )
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if not self._processes["web"].is_running():
                raise ComponentCommandError("Web", self._layout.log("web"), self._layout.root)
            try:
                with urllib.request.urlopen(self._web_url, timeout=1):  # noqa: S310
                    return
            except (OSError, urllib.error.URLError):
                time.sleep(0.2)
        raise ComponentCommandError("Web", self._layout.log("web"), self._layout.root)

    def _wait_for_api(self) -> None:
        """Wait for the API health contract before allowing web traffic."""
        timeout = int(os.environ.get("NORVII_API_START_TIMEOUT_SECONDS", "30"))
        if timeout <= 0:
            raise LocalEnvironmentError(
                "NORVII_API_START_TIMEOUT_SECONDS must be a positive integer."
            )
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if not self._processes["api"].is_running():
                raise ComponentCommandError("API", self._layout.log("api"), self._layout.root)
            try:
                with urllib.request.urlopen(self._api_health_url, timeout=1) as response:  # noqa: S310
                    if response.status == HTTP_OK:
                        return
            except (OSError, urllib.error.URLError):
                time.sleep(0.2)
        raise ComponentCommandError("API", self._layout.log("api"), self._layout.root)

    def _wait_for_agent(self) -> None:
        """Wait for the internal LangGraph agent health contract."""
        timeout = int(os.environ.get("NORVII_AGENT_START_TIMEOUT_SECONDS", "30"))
        if timeout <= 0:
            raise LocalEnvironmentError(
                "NORVII_AGENT_START_TIMEOUT_SECONDS must be a positive integer."
            )
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if not self._processes["agent"].is_running():
                raise ComponentCommandError("Agent", self._layout.log("agent"), self._layout.root)
            try:
                with urllib.request.urlopen(self._agent_health_url, timeout=1) as response:  # noqa: S310
                    if response.status == HTTP_OK:
                        return
            except (OSError, urllib.error.URLError):
                time.sleep(0.2)
        raise ComponentCommandError("Agent", self._layout.log("agent"), self._layout.root)

    def _wait_for_initial_sources(self) -> dict[str, str]:
        """Wait until each stable seed source reaches a bounded terminal state."""
        deadline = time.monotonic() + self._initial_ingestion_timeout
        while time.monotonic() < deadline:
            states: dict[str, str] = {}
            try:
                for language, corpus_id in INITIAL_CORPORA.items():
                    url = f"{self._api_base_url}/corpora/{corpus_id}/sources"
                    with urllib.request.urlopen(url, timeout=2) as response:  # noqa: S310
                        payload = json.load(response)
                    if not isinstance(payload, list) or len(payload) != 1:
                        states[language] = "pending"
                        continue
                    source = payload[0]
                    status = source.get("processingStatus") if isinstance(source, dict) else None
                    states[language] = status if isinstance(status, str) else "pending"
            except (OSError, TypeError, ValueError, urllib.error.URLError):
                time.sleep(0.25)
                continue
            if all(state in {"ready", "failed"} for state in states.values()):
                return states
            time.sleep(0.25)
        raise ComponentCommandError("Ingestion", self._layout.log("ingestion"), self._layout.root)

    def _initialize_initial_snapshots(self, states: dict[str, str]) -> str:
        """Create initial releases only after both seeded sources are retrieval-ready."""
        if not all(state == "ready" for state in states.values()):
            return "pending (one or more initial sources are not ready)"
        self._runner.run("api", self._make("persistence-initialize-snapshots"))
        return "ready"

    def _stop_managed_processes(self, components: Iterable[str]) -> None:
        for component in components:
            self._processes[component].stop()


def parse_arguments(arguments: list[str]) -> argparse.Namespace:
    """Parse a lifecycle action and optional repository root."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("action", choices=("start", "status", "stop"))
    parser.add_argument("--repository-root", type=Path, default=Path(__file__).parents[2])
    return parser.parse_args(arguments)


def main(arguments: list[str]) -> int:
    """Execute the requested local lifecycle action."""
    options = parse_arguments(arguments)
    try:
        manager = LocalEnvironmentManager(RepositoryLayout(options.repository_root.resolve()))
        getattr(manager, options.action)()
    except (LocalEnvironmentError, ValueError) as error:
        print(f"Norvii local environment failed: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
