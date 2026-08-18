from __future__ import annotations

import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable, Sequence


class RepositoryLanguageError(RuntimeError):
    """Raised when repository files cannot be discovered or validated."""


@dataclass(frozen=True)
class LanguageViolation:
    path: Path
    line_number: int
    reason: str


class RepositoryLanguagePolicy:
    _TEXT_SUFFIXES = frozenset(
        {
            ".css",
            ".go",
            ".graphql",
            ".html",
            ".js",
            ".json",
            ".jsx",
            ".md",
            ".properties",
            ".py",
            ".scss",
            ".sh",
            ".sql",
            ".toml",
            ".ts",
            ".tsx",
            ".txt",
            ".xml",
            ".yaml",
            ".yml",
        }
    )
    _TEXT_FILENAMES = frozenset(
        {"AGENTS.md", "Dockerfile", "LICENSE", "Makefile", "NOTICE", "README.md"}
    )
    _MANAGED_PREFIXES = (
        Path(".agents/skills/speckit-"),
        Path(".specify/extensions"),
        Path(".specify/scripts"),
        Path(".specify/templates"),
        Path(".specify/workflows"),
    )
    _PORTUGUESE_TERM_PARTS = (
        ("adicio", "nar"),
        ("aplica", "cao"),
        ("arqui", "vo"),
        ("atuali", "zar"),
        ("bus", "car"),
        ("cadas", "tro"),
        ("cance", "lar"),
        ("carre", "gando"),
        ("codi", "go"),
        ("confir", "mar"),
        ("cri", "ar"),
        ("documenta", "cao"),
        ("envi", "ar"),
        ("er", "ro"),
        ("fa", "lha"),
        ("idio", "ma"),
        ("in", "gles"),
        ("pergun", "ta"),
        ("portu", "gues"),
        ("proce", "sso"),
        ("remo", "ver"),
        ("reposi", "torio"),
        ("respos", "ta"),
        ("sal", "var"),
        ("sis", "tema"),
        ("usua", "rio"),
        ("usua", "rios"),
    )

    def __init__(self) -> None:
        terms = "|".join(
            re.escape("".join(parts)) for parts in self._PORTUGUESE_TERM_PARTS
        )
        self._portuguese_pattern = re.compile(rf"\b(?:{terms})\b", re.IGNORECASE)

    def validate(self, root: Path, paths: Iterable[Path]) -> list[LanguageViolation]:
        violations: list[LanguageViolation] = []
        for relative_path in sorted(set(paths)):
            if self._is_managed(relative_path):
                continue
            violations.extend(self._validate_path(relative_path))
            if not self._is_text(relative_path):
                continue
            violations.extend(self._validate_file(root, relative_path))
        return violations

    def _validate_path(self, path: Path) -> list[LanguageViolation]:
        path_text = path.as_posix()
        violations: list[LanguageViolation] = []
        if not path_text.isascii():
            violations.append(
                LanguageViolation(
                    path, 1, "file and directory names must use ASCII English"
                )
            )

        match = self._portuguese_pattern.search(path_text)
        if match:
            violations.append(
                LanguageViolation(
                    path,
                    1,
                    f"Portuguese path term '{match.group(0)}' is not allowed",
                )
            )
        return violations

    def _validate_file(
        self, root: Path, relative_path: Path
    ) -> list[LanguageViolation]:
        if self._is_content_exception(relative_path):
            return []

        try:
            content = (root / relative_path).read_text(encoding="utf-8")
        except UnicodeDecodeError:
            return [LanguageViolation(relative_path, 1, "file is not valid UTF-8 text")]
        except OSError as error:
            raise RepositoryLanguageError(
                f"Unable to read {relative_path.as_posix()}: {error}"
            ) from error

        violations: list[LanguageViolation] = []
        for line_number, line in enumerate(content.splitlines(), start=1):
            if not line.isascii():
                violations.append(
                    LanguageViolation(
                        relative_path,
                        line_number,
                        "non-ASCII text is outside an approved content path",
                    )
                )

            match = self._portuguese_pattern.search(line)
            if match:
                violations.append(
                    LanguageViolation(
                        relative_path,
                        line_number,
                        f"Portuguese term '{match.group(0)}' is not allowed here",
                    )
                )
        return violations

    def _is_managed(self, path: Path) -> bool:
        return any(self._has_prefix(path, prefix) for prefix in self._MANAGED_PREFIXES)

    def _is_text(self, path: Path) -> bool:
        return (
            path.name in self._TEXT_FILENAMES
            or path.suffix.lower() in self._TEXT_SUFFIXES
        )

    def _is_content_exception(self, path: Path) -> bool:
        parts = path.parts
        if parts and parts[0] == "corpora":
            return True
        if len(parts) >= 2 and parts[:2] == ("data", "corpora"):
            return True
        if "legal-content" in parts and "fixtures" in parts:
            return True

        for marker in ("i18n", "locales"):
            if marker not in parts:
                continue
            marker_index = parts.index(marker)
            locale_parts = parts[marker_index + 1 :]
            if locale_parts and locale_parts[0].lower().startswith("pt"):
                return True
        return False

    @staticmethod
    def _has_prefix(path: Path, prefix: Path) -> bool:
        path_parts = path.parts
        prefix_parts = prefix.parts
        if prefix.name.endswith("-"):
            parent_parts = prefix_parts[:-1]
            return (
                path_parts[: len(parent_parts)] == parent_parts
                and len(path_parts) > len(parent_parts)
                and path_parts[len(parent_parts)].startswith(prefix.name)
            )
        return path_parts[: len(prefix_parts)] == prefix_parts


class GitRepositoryFileDiscovery:
    def discover(self, root: Path) -> list[Path]:
        try:
            result = subprocess.run(
                ("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z"),
                cwd=root,
                check=True,
                capture_output=True,
            )
        except (OSError, subprocess.CalledProcessError) as error:
            raise RepositoryLanguageError(
                "Unable to discover project-owned files with Git."
            ) from error

        paths = [Path(item.decode("utf-8")) for item in result.stdout.split(b"\0") if item]
        return [path for path in paths if (root / path).exists()]


class RepositoryLanguageApplication:
    def run(self, arguments: Sequence[str]) -> int:
        root = Path(arguments[0]).resolve() if arguments else Path.cwd()
        try:
            paths = GitRepositoryFileDiscovery().discover(root)
            violations = RepositoryLanguagePolicy().validate(root, paths)
        except RepositoryLanguageError as error:
            print(f"error: {error}", file=sys.stderr)
            return 2

        if violations:
            for violation in violations:
                print(
                    f"{violation.path.as_posix()}:{violation.line_number}: "
                    f"{violation.reason}",
                    file=sys.stderr,
                )
            print(
                f"error: repository language policy found {len(violations)} violation(s)",
                file=sys.stderr,
            )
            return 1

        print("Repository source and documentation language policy passed.")
        return 0


if __name__ == "__main__":
    raise SystemExit(RepositoryLanguageApplication().run(sys.argv[1:]))
