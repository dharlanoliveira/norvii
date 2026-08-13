#!/usr/bin/env python3
"""Run a command with a validated dotenv file without evaluating shell code."""

from __future__ import annotations

import os
import re
import shlex
import sys
from pathlib import Path


ASSIGNMENT_PATTERN = re.compile(r"^([A-Za-z_][A-Za-z0-9_]*)=(.*)$")


class EnvironmentFileError(ValueError):
    """Indicate that an environment file cannot be safely parsed."""


class EnvironmentFile:
    """Parse a strict dotenv subset without interpolation or code evaluation."""

    def __init__(self, path: Path) -> None:
        self._path = path

    def read(self) -> dict[str, str]:
        """Return validated assignments while preserving quoted literal characters."""
        try:
            lines = self._path.read_text(encoding="utf-8").splitlines()
        except OSError as error:
            raise EnvironmentFileError(f"Environment file {self._path} could not be read.") from error

        assignments: dict[str, str] = {}
        for line_number, raw_line in enumerate(lines, start=1):
            line = raw_line.strip()
            if not line or line.startswith("#"):
                continue
            match = ASSIGNMENT_PATTERN.fullmatch(line)
            if match is None:
                raise EnvironmentFileError(
                    f"Environment file {self._path} has an invalid assignment on line {line_number}."
                )
            name, raw_value = match.groups()
            assignments[name] = self._parse_value(raw_value, line_number)
        return assignments

    def _parse_value(self, raw_value: str, line_number: int) -> str:
        try:
            tokens = shlex.split(raw_value, comments=True, posix=True)
        except ValueError as error:
            raise EnvironmentFileError(
                f"Environment file {self._path} has invalid quoting on line {line_number}."
            ) from error
        if len(tokens) > 1:
            raise EnvironmentFileError(
                f"Environment file {self._path} has an unquoted value on line {line_number}."
            )
        return tokens[0] if tokens else ""


class EnvironmentCommandRunner:
    """Replace the current process with a command using parsed environment values."""

    def __init__(self, environment_file: EnvironmentFile) -> None:
        self._environment_file = environment_file

    def run(self, command: list[str]) -> None:
        """Execute a nonempty command with inherited and file-provided environment."""
        if not command:
            raise EnvironmentFileError("A command is required after the environment file.")
        environment = os.environ | self._environment_file.read()
        os.execvpe(command[0], command, environment)


def main(arguments: list[str]) -> int:
    """Validate CLI input and execute the requested command."""
    if len(arguments) < 2:
        print("Usage: run-with-environment.py ENV_FILE COMMAND [ARGUMENT ...]", file=sys.stderr)
        return 2
    try:
        EnvironmentCommandRunner(EnvironmentFile(Path(arguments[0]))).run(arguments[1:])
    except EnvironmentFileError as error:
        print(error, file=sys.stderr)
        return 2
    except OSError as error:
        print(f"Command could not be started: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
