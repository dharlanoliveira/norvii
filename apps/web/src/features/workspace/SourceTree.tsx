import { ChevronRight, ExternalLink, FileText, FolderTree } from "lucide-react";
import { useRef, useState, type KeyboardEvent } from "react";
import { useTranslation } from "react-i18next";

import type { Corpus, Source } from "../../research/domain/models";

interface SourceTreeProps {
  readonly corpus: Corpus;
  readonly selectedSourceId: string | undefined;
  readonly onSelect: (sourceId: string) => void;
}

type SourceGroup = "pdf" | "external";

interface GroupDefinition {
  readonly id: SourceGroup;
  readonly label: string;
  readonly sources: readonly Source[];
}

export function SourceTree({
  corpus,
  selectedSourceId,
  onSelect,
}: SourceTreeProps) {
  const { t } = useTranslation();
  const treeRef = useRef<HTMLDivElement>(null);
  const [activeItemId, setActiveItemId] = useState("root");
  const [expanded, setExpanded] = useState<Record<SourceGroup, boolean>>({
    pdf: true,
    external: true,
  });
  const groups: readonly GroupDefinition[] = [
    {
      id: "pdf",
      label: t("tree.pdfDocuments"),
      sources: corpus.sources.filter((source) => source.kind === "pdf"),
    },
    {
      id: "external",
      label: t("tree.externalLinks"),
      sources: corpus.sources.filter((source) => source.kind === "external"),
    },
  ];

  const toggleGroup = (group: SourceGroup): void => {
    setExpanded((current) => ({ ...current, [group]: !current[group] }));
  };

  const focusItem = (item: HTMLElement | undefined): void => {
    if (!item?.dataset.treeId) return;
    setActiveItemId(item.dataset.treeId);
    item.focus();
  };

  const handleTreeKeyDown = (event: KeyboardEvent<HTMLDivElement>): void => {
    const target = (event.target as HTMLElement).closest<HTMLElement>(
      "[role='treeitem']",
    );
    const items = Array.from(
      treeRef.current?.querySelectorAll<HTMLElement>("[role='treeitem']") ?? [],
    );
    if (!target || !treeRef.current?.contains(target)) return;

    const currentIndex = items.indexOf(target);
    if (currentIndex < 0) return;

    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      const offset = event.key === "ArrowDown" ? 1 : -1;
      const nextIndex = Math.min(
        Math.max(currentIndex + offset, 0),
        items.length - 1,
      );
      focusItem(items[nextIndex]);
      return;
    }

    if (event.key === "Home" || event.key === "End") {
      event.preventDefault();
      focusItem(event.key === "Home" ? items[0] : items.at(-1));
      return;
    }

    if (event.key === "ArrowRight") {
      event.preventDefault();
      if (target.dataset.treeKind === "root") {
        focusItem(items[currentIndex + 1]);
        return;
      }
      if (target.dataset.treeKind !== "group") return;
      const groupId = target.dataset.groupId as SourceGroup;
      if (target.getAttribute("aria-expanded") === "false") {
        toggleGroup(groupId);
        return;
      }
      focusItem(items[currentIndex + 1]);
      return;
    }

    if (event.key !== "ArrowLeft") return;
    event.preventDefault();
    if (
      target.dataset.treeKind === "group" &&
      target.getAttribute("aria-expanded") === "true"
    ) {
      toggleGroup(target.dataset.groupId as SourceGroup);
      return;
    }
    const parentId = target.dataset.parentId;
    const parent = parentId
      ? treeRef.current.querySelector<HTMLElement>(
          `[data-tree-id='${parentId}']`,
        )
      : items[0];
    focusItem(parent ?? undefined);
  };

  return (
    <div
      ref={treeRef}
      className="source-tree"
      role="tree"
      aria-label={t("tree.label")}
      onKeyDown={handleTreeKeyDown}
    >
      <div
        className="source-tree__root"
        role="treeitem"
        aria-expanded="true"
        aria-level={1}
        data-tree-id="root"
        data-tree-kind="root"
        tabIndex={activeItemId === "root" ? 0 : -1}
        onFocus={() => setActiveItemId("root")}
      >
        <FolderTree aria-hidden="true" size={16} />
        <span>{corpus.name}</span>
      </div>
      <div role="group">
        {groups.map((group) => (
          <div className="source-tree__group" key={group.id}>
            <button
              type="button"
              className="source-tree__group-button"
              role="treeitem"
              aria-expanded={expanded[group.id]}
              aria-level={2}
              aria-label={`${group.label}, ${t("catalog.sourceCount", { count: group.sources.length })}`}
              data-tree-id={`group-${group.id}`}
              data-tree-kind="group"
              data-group-id={group.id}
              tabIndex={activeItemId === `group-${group.id}` ? 0 : -1}
              onFocus={() => setActiveItemId(`group-${group.id}`)}
              onClick={() => toggleGroup(group.id)}
            >
              <ChevronRight aria-hidden="true" size={15} />
              <span>{group.label}</span>
              <small>{group.sources.length}</small>
            </button>
            {expanded[group.id] && group.sources.length > 0 ? (
              <div role="group" className="source-tree__leaves">
                {group.sources.map((source) => {
                  const selected = source.id === selectedSourceId;
                  const typeLabel =
                    source.kind === "pdf"
                      ? t("tree.pdfType")
                      : t("tree.externalType");
                  const availabilityLabel =
                    source.kind === "external" &&
                    source.preview.status === "unavailable"
                      ? t("tree.unavailable")
                      : t("tree.available");
                  const availabilityId = `${corpus.id}-${source.id}-availability`;
                  return (
                    <button
                      type="button"
                      role="treeitem"
                      aria-level={3}
                      aria-selected={selected}
                      aria-describedby={availabilityId}
                      aria-label={`${typeLabel}: ${source.title}${selected ? `, ${t("tree.selected")}` : ""}`}
                      data-tree-id={`source-${source.id}`}
                      data-tree-kind="source"
                      data-parent-id={`group-${group.id}`}
                      className="source-tree__source"
                      key={source.id}
                      tabIndex={activeItemId === `source-${source.id}` ? 0 : -1}
                      onFocus={() => setActiveItemId(`source-${source.id}`)}
                      onClick={() => onSelect(source.id)}
                    >
                      {source.kind === "pdf" ? (
                        <FileText aria-hidden="true" size={15} />
                      ) : (
                        <ExternalLink aria-hidden="true" size={15} />
                      )}
                      <span>{source.title}</span>
                      <small id={availabilityId}>
                        <span aria-hidden="true">{typeLabel} / </span>
                        {availabilityLabel}
                      </small>
                    </button>
                  );
                })}
              </div>
            ) : null}
          </div>
        ))}
      </div>
    </div>
  );
}
