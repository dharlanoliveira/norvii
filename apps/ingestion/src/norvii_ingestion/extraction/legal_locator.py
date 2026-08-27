"""Derive stable, auditable legal locator aliases from extracted unit markers."""

from __future__ import annotations

import re

_ARTICLE_MARKER = re.compile(
    r"^(?:article|artigo|art\.)\s*(?P<number>\d+)(?:[\u00ba\u00b0o])?(?P<suffix>-[a-z]+)?",
    re.IGNORECASE,
)
_HEADING_MARKER = re.compile(
    r"^(title|t\u00edtulo|chapter|cap\u00edtulo|section|se\u00e7\u00e3o|recital)\s+([\divxlcdm]+)",
    re.IGNORECASE,
)
_PARAGRAPH_MARKER = re.compile(
    r"^(?:\u00a7\s*)?(?P<number>\d+)(?:[\u00ba\u00b0o])?(?P<suffix>-[a-z]+)?\.?",
    re.IGNORECASE,
)
_ROMAN_ITEM_MARKER = re.compile(r"^([ivxlcdm]+)\s*[-\u2013\u2014]", re.IGNORECASE)
_LETTER_ITEM_MARKER = re.compile(r"^(?:\(([a-z])\)|([a-z])\))", re.IGNORECASE)
_US_CODE_MARKER = re.compile(
    r"^(?P<title>\d+)\s+u\.s\.c\.\s+\u00a7\s*(?P<section>\d+(?:\.\d+)*)"
    r"(?P<items>(?:\([a-z0-9]+\))*)",
    re.IGNORECASE,
)
_CFR_MARKER = re.compile(
    r"^(?P<title>\d+)\s+cfr\s+\u00a7\s*(?P<section>\d+(?:\.\d+)*)"
    r"(?P<items>(?:\([a-z0-9]+\))*)",
    re.IGNORECASE,
)
_SECTION_MARKER = re.compile(
    r"^(?:section|sec\.)\s*(?P<section>\d+(?:\.\d+)*)"
    r"(?P<items>(?:\([a-z0-9]+\))*)",
    re.IGNORECASE,
)
_ITEM_GROUP = re.compile(r"\(([a-z0-9]+)\)", re.IGNORECASE)


def canonical_legal_locator(
    kind: str,
    marker: str | None,
    parent_locator: str | None,
) -> str | None:
    """Return a stable alias for one legal marker, or ``None`` when it is structural only."""
    if marker is None:
        return None
    normalized_marker = " ".join(marker.split())
    if standalone := _standalone_legal_locator(normalized_marker, parent_locator):
        return standalone
    return _nested_legal_locator(kind, normalized_marker, parent_locator)


def _standalone_legal_locator(marker: str, parent_locator: str | None) -> str | None:
    if us_code := _US_CODE_MARKER.match(marker):
        return _section_locator("us-code", us_code)
    if cfr := _CFR_MARKER.match(marker):
        return _section_locator("cfr", cfr)
    if section := _SECTION_MARKER.match(marker):
        return _scoped_locator(parent_locator, _section_locator(None, section))
    if article := _ARTICLE_MARKER.match(marker):
        return f"article:{_provision_identifier(article)}"
    if heading := _HEADING_MARKER.match(marker):
        heading_kind = {
            "t\u00edtulo": "title",
            "cap\u00edtulo": "chapter",
            "se\u00e7\u00e3o": "section",
        }.get(heading.group(1).casefold(), heading.group(1).casefold())
        locator = f"{heading_kind}:{heading.group(2).casefold()}"
        return _scoped_locator(parent_locator, locator)
    return None


def _nested_legal_locator(kind: str, marker: str, parent_locator: str | None) -> str | None:
    if kind == "paragraph" and (paragraph := _PARAGRAPH_MARKER.match(marker)):
        return _nested_locator(parent_locator, "paragraph", _provision_identifier(paragraph))
    if kind == "item" and (roman_item := _ROMAN_ITEM_MARKER.match(marker)):
        return _nested_locator(parent_locator, "item", roman_item.group(1).casefold())
    if kind == "item" and (letter_item := _LETTER_ITEM_MARKER.match(marker)):
        value = letter_item.group(1) or letter_item.group(2)
        return _nested_locator(parent_locator, "item", value.casefold())
    return None


def _section_locator(prefix: str | None, match: re.Match[str]) -> str:
    components = [] if prefix is None else [f"{prefix}:{match.group('title').casefold()}"]
    components.append(f"section:{match.group('section').casefold()}")
    components.extend(
        f"item:{item.casefold()}" for item in _ITEM_GROUP.findall(match.group("items"))
    )
    return "/".join(components)


def _provision_identifier(match: re.Match[str]) -> str:
    return f"{match.group('number')}{match.group('suffix') or ''}".casefold()


def _scoped_locator(parent_locator: str | None, locator: str) -> str:
    if parent_locator is None:
        return locator
    kind, value = locator.split(":", maxsplit=1)
    return f"{parent_locator}/{kind}:{value}"


def _nested_locator(parent_locator: str | None, kind: str, value: str) -> str | None:
    if parent_locator is None:
        return None
    return f"{parent_locator}/{kind}:{value.casefold()}"
