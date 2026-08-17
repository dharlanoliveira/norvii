import {
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileText,
  FolderOpen,
} from "lucide-react";
import { useMemo, useState, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

import type {
  CorpusFixture,
  CorpusSource,
  SourceKind,
} from "../../fixtures/models";

interface SourceTreeProps {
  readonly corpus: CorpusFixture;
  readonly selectedSourceId: string | null;
  readonly onSelectSource: (sourceId: string) => void;
}

interface SourceGroup {
  readonly kind: SourceKind;
  readonly sources: readonly CorpusSource[];
}

export function SourceTree({
  corpus,
  selectedSourceId,
  onSelectSource,
}: SourceTreeProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState<Record<SourceKind, boolean>>({
    pdf: true,
    external: true,
  });
  const groups = useMemo<readonly SourceGroup[]>(
    () => [
      {
        kind: "pdf",
        sources: corpus.sources.filter((source) => source.kind === "pdf"),
      },
      {
        kind: "external",
        sources: corpus.sources.filter((source) => source.kind === "external"),
      },
    ],
    [corpus.sources],
  );

  function moveFocus(event: KeyboardEvent<HTMLElement>) {
    if (event.key !== "ArrowDown" && event.key !== "ArrowUp") {
      return;
    }
    const tree = event.currentTarget.closest('[role="tree"]');
    const items = tree
      ? Array.from(tree.querySelectorAll<HTMLElement>('[role="treeitem"]'))
      : [];
    const index = items.indexOf(event.currentTarget);
    const offset = event.key === "ArrowDown" ? 1 : -1;
    const target = items[index + offset];
    if (target) {
      event.preventDefault();
      target.focus();
    }
  }

  return (
    <div
      className="source-tree"
      role="tree"
      aria-label={t("workspace.treeLabel", { corpus: corpus.label })}
    >
      <div
        className="tree-root"
        role="treeitem"
        tabIndex={0}
        aria-expanded="true"
        onKeyDown={moveFocus}
      >
        <span className="tree-root-icon" aria-hidden="true">
          <FolderOpen size={16} />
        </span>
        <span>{corpus.label}</span>
      </div>
      <div role="group" className="tree-groups">
        {groups.map((group) => {
          const isOpen = expanded[group.kind];
          const groupLabel = t(
            group.kind === "pdf"
              ? "workspace.pdfGroup"
              : "workspace.externalGroup",
          );
          return (
            <div className="source-group" key={group.kind}>
              <button
                type="button"
                role="treeitem"
                aria-expanded={isOpen}
                className="tree-group-button"
                onClick={() =>
                  setExpanded((current) => ({
                    ...current,
                    [group.kind]: !current[group.kind],
                  }))
                }
                onKeyDown={moveFocus}
              >
                {isOpen ? (
                  <ChevronDown aria-hidden="true" size={15} />
                ) : (
                  <ChevronRight aria-hidden="true" size={15} />
                )}
                <span>{groupLabel}</span>
                <span className="tree-count">{group.sources.length}</span>
              </button>
              {isOpen ? (
                <div role="group" className="source-leaves">
                  {group.sources.map((source) => {
                    const isSelected = source.id === selectedSourceId;
                    return (
                      <button
                        type="button"
                        role="treeitem"
                        aria-selected={isSelected}
                        tabIndex={isSelected ? 0 : -1}
                        className={
                          isSelected ? "source-leaf selected" : "source-leaf"
                        }
                        key={source.id}
                        onClick={() => onSelectSource(source.id)}
                        onKeyDown={(event) => {
                          moveFocus(event);
                          if (event.key === "Enter" || event.key === " ") {
                            event.preventDefault();
                            onSelectSource(source.id);
                          }
                        }}
                      >
                        {source.kind === "pdf" ? (
                          <FileText aria-hidden="true" size={16} />
                        ) : (
                          <ExternalLink aria-hidden="true" size={16} />
                        )}
                        <span className="source-leaf-copy">
                          <span>{source.shortTitle}</span>
                          <small>
                            {source.status === "available"
                              ? t("workspace.available")
                              : t("workspace.unavailable")}
                          </small>
                        </span>
                      </button>
                    );
                  })}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
