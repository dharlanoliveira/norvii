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

interface TreeNavigationState {
  readonly currentIndex: number;
  readonly items: readonly HTMLElement[];
  readonly target: HTMLElement;
  readonly focusItem: (item: HTMLElement | undefined) => void;
  readonly toggleGroup: (group: SourceGroup) => void;
  readonly tree: HTMLDivElement;
}

function preventAndMove(
  event: KeyboardEvent<HTMLDivElement>,
  navigation: TreeNavigationState,
  index: number,
): void {
  event.preventDefault();
  navigation.focusItem(navigation.items[index]);
}

function moveByOffset(
  event: KeyboardEvent<HTMLDivElement>,
  navigation: TreeNavigationState,
  offset: number,
): void {
  const nextIndex = Math.min(
    Math.max(navigation.currentIndex + offset, 0),
    navigation.items.length - 1,
  );
  preventAndMove(event, navigation, nextIndex);
}

function moveToBoundary(
  event: KeyboardEvent<HTMLDivElement>,
  navigation: TreeNavigationState,
  index: number,
): void {
  preventAndMove(event, navigation, index);
}

function moveRight(
  event: KeyboardEvent<HTMLDivElement>,
  navigation: TreeNavigationState,
): void {
  event.preventDefault();
  if (navigation.target.dataset.treeKind === "root") {
    navigation.focusItem(navigation.items[navigation.currentIndex + 1]);
    return;
  }
  if (navigation.target.dataset.treeKind !== "group") return;

  const groupId = navigation.target.dataset.groupId as SourceGroup;
  if (navigation.target.getAttribute("aria-expanded") === "false") {
    navigation.toggleGroup(groupId);
    return;
  }
  navigation.focusItem(navigation.items[navigation.currentIndex + 1]);
}

function moveLeft(
  event: KeyboardEvent<HTMLDivElement>,
  navigation: TreeNavigationState,
): void {
  event.preventDefault();
  if (
    navigation.target.dataset.treeKind === "group" &&
    navigation.target.getAttribute("aria-expanded") === "true"
  ) {
    navigation.toggleGroup(navigation.target.dataset.groupId as SourceGroup);
    return;
  }

  const parentId = navigation.target.dataset.parentId;
  const parent = parentId
    ? navigation.tree.querySelector<HTMLElement>(`[data-tree-id='${parentId}']`)
    : navigation.items[0];
  navigation.focusItem(parent ?? undefined);
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
    const tree = treeRef.current;
    const target = (event.target as HTMLElement).closest<HTMLElement>(
      "[role='treeitem']",
    );
    const items = Array.from(
      tree?.querySelectorAll<HTMLElement>("[role='treeitem']") ?? [],
    );
    if (!tree || !target || !tree.contains(target)) return;

    const currentIndex = items.indexOf(target);
    if (currentIndex < 0) return;

    const navigation: TreeNavigationState = {
      currentIndex,
      items,
      target,
      focusItem,
      toggleGroup,
      tree,
    };

    switch (event.key) {
      case "ArrowDown":
        moveByOffset(event, navigation, 1);
        break;
      case "ArrowUp":
        moveByOffset(event, navigation, -1);
        break;
      case "Home":
        moveToBoundary(event, navigation, 0);
        break;
      case "End":
        moveToBoundary(event, navigation, items.length - 1);
        break;
      case "ArrowRight":
        moveRight(event, navigation);
        break;
      case "ArrowLeft":
        moveLeft(event, navigation);
        break;
    }
  };

  return (
    <div
      ref={treeRef}
      className="source-tree"
      role="tree"
      aria-label={t("tree.label")}
      tabIndex={-1}
      onKeyDown={handleTreeKeyDown}
    >
      <div
        className="source-tree__root"
        role="treeitem"
        aria-expanded="true"
        aria-level={1}
        aria-selected={false}
        data-tree-id="root"
        data-tree-kind="root"
        tabIndex={activeItemId === "root" ? 0 : -1}
        onFocus={() => setActiveItemId("root")}
      >
        <FolderTree aria-hidden="true" size={16} />
        <span>{corpus.name}</span>
      </div>
      <div>
        {groups.map((group) => (
          <div className="source-tree__group" key={group.id}>
            <button
              type="button"
              className="source-tree__group-button"
              role="treeitem"
              aria-expanded={expanded[group.id]}
              aria-level={2}
              aria-selected={false}
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
              <div className="source-tree__leaves">
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
                  const selectionLabel = selected
                    ? `, ${t("tree.selected")}`
                    : "";
                  const sourceLabel = `${typeLabel}: ${source.title}${selectionLabel}`;
                  return (
                    <button
                      type="button"
                      role="treeitem"
                      aria-level={3}
                      aria-selected={selected}
                      aria-describedby={availabilityId}
                      aria-label={sourceLabel}
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
