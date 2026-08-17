from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT_DIRECTORY = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(SCRIPT_DIRECTORY))

from validate_repository_language import RepositoryLanguagePolicy  # noqa: E402


class RepositoryLanguagePolicyTests(unittest.TestCase):
    def setUp(self) -> None:
        self._temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self._temporary_directory.cleanup)
        self._root = Path(self._temporary_directory.name)
        self._policy = RepositoryLanguagePolicy()

    def test_accepts_english_source_and_documentation(self) -> None:
        path = self._write("docs/example.md", "Describe the repository behavior.\n")

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(violations, [])

    def test_rejects_non_ascii_text_outside_content_exception(self) -> None:
        portuguese_text = "Documenta" + chr(0xE7) + chr(0xE3) + "o do projeto.\n"
        path = self._write("docs/example.md", portuguese_text)

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(len(violations), 1)
        self.assertIn("non-ASCII", violations[0].reason)

    def test_rejects_common_portuguese_term_without_accents(self) -> None:
        portuguese_term = "reposi" + "torio"
        path = self._write("apps/api/example.go", f"// {portuguese_term}\n")

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(len(violations), 1)
        self.assertIn(portuguese_term, violations[0].reason)

    def test_allows_portuguese_localization_content(self) -> None:
        localized_value = "Aplica" + chr(0xE7) + chr(0xE3) + "o"
        path = self._write("apps/web/src/i18n/pt/messages.json", localized_value)

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(violations, [])

    def test_allows_preserved_legal_corpus_content(self) -> None:
        legal_text = "Constitui" + chr(0xE7) + chr(0xE3) + "o"
        path = self._write("corpora/brazil/source.txt", legal_text)

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(violations, [])

    def test_rejects_portuguese_file_or_directory_name(self) -> None:
        portuguese_path = "docs/" + "proce" + "sso.md"
        path = self._write(portuguese_path, "English content.\n")

        violations = self._policy.validate(self._root, [path])

        self.assertEqual(len(violations), 1)
        self.assertIn("path term", violations[0].reason)

    def _write(self, relative_path: str, content: str) -> Path:
        path = Path(relative_path)
        absolute_path = self._root / path
        absolute_path.parent.mkdir(parents=True, exist_ok=True)
        absolute_path.write_text(content, encoding="utf-8")
        return path


if __name__ == "__main__":
    unittest.main()
