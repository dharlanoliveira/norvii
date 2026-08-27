import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { corpora, findCorpus } from "../../fixtures/legal-content/corpora";
import type { CorpusFixture } from "../../fixtures/models";
import { resolveOpeningSuggestions } from "../../fixtures/legal-content/opening-suggestions";
import { i18n } from "../../i18n/config";
import { renderAtRoute } from "../../test/render";
import { ResearchChat } from "./ResearchChat";

function getCorpus(corpusId: string): CorpusFixture {
  const corpus = findCorpus(corpusId);
  if (!corpus) {
    throw new Error(`Expected catalog corpus ${corpusId}.`);
  }
  return corpus;
}

function isSuggestionButton(question: string) {
  return (accessibleName: string) => accessibleName.endsWith(question);
}

describe("prototype opening suggestions", () => {
  beforeEach(async () => {
    await act(async () => {
      await i18n.changeLanguage("en");
    });
  });

  afterEach(async () => {
    await act(async () => {
      await i18n.changeLanguage("en");
    });
  });

  it.each(corpora)(
    "keeps five rank-ordered paired suggestions and prepared prompts for $label",
    (corpus) => {
      const englishSuggestions = resolveOpeningSuggestions(corpus, "en");
      const portugueseSuggestions = resolveOpeningSuggestions(corpus, "pt");

      expect(englishSuggestions.map((suggestion) => suggestion.rank)).toEqual([
        1, 2, 3, 4, 5,
      ]);
      expect(
        portugueseSuggestions.map((suggestion) => suggestion.rank),
      ).toEqual([1, 2, 3, 4, 5]);
      expect(englishSuggestions.map((suggestion) => suggestion.caseId)).toEqual(
        portugueseSuggestions.map((suggestion) => suggestion.caseId),
      );

      for (const suggestion of [
        ...englishSuggestions,
        ...portugueseSuggestions,
      ]) {
        expect(
          corpus.preparedAnswers.some((preparedAnswer) =>
            preparedAnswer.prompts.includes(suggestion.question),
          ),
        ).toBe(true);
      }
    },
  );

  it("replaces empty-chat suggestions when the catalog-selected corpus changes", async () => {
    const lgpdCorpus = getCorpus("brazil-data-protection");
    const fairHousingCorpus = getCorpus(
      "us-fair-housing-disability-accommodations",
    );
    const lgpdQuestion = resolveOpeningSuggestions(lgpdCorpus, "en").at(
      0,
    )?.question;
    const fairHousingQuestion = resolveOpeningSuggestions(
      fairHousingCorpus,
      "en",
    ).at(0)?.question;

    if (!lgpdQuestion || !fairHousingQuestion) {
      throw new Error("Expected catalog opening suggestions.");
    }

    const view = renderAtRoute(
      <ResearchChat corpus={lgpdCorpus} onOpenCitation={vi.fn()} />,
    );

    expect(
      await screen.findByRole("button", {
        name: isSuggestionButton(lgpdQuestion),
      }),
    ).toBeInTheDocument();

    view.rerender(
      <ResearchChat corpus={fairHousingCorpus} onOpenCitation={vi.fn()} />,
    );

    expect(
      await screen.findByRole("button", {
        name: isSuggestionButton(fairHousingQuestion),
      }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("button", {
        name: isSuggestionButton(lgpdQuestion),
      }),
    ).not.toBeInTheDocument();
  });

  it("shows the exact paired questions when the interface language changes", async () => {
    const corpus = getCorpus("brazil-anti-corruption-white-collar-crime");
    const englishQuestion = resolveOpeningSuggestions(corpus, "en").at(
      0,
    )?.question;
    const portugueseQuestion = resolveOpeningSuggestions(corpus, "pt").at(
      0,
    )?.question;

    if (!englishQuestion || !portugueseQuestion) {
      throw new Error("Expected paired anti-corruption questions.");
    }

    renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);

    expect(
      await screen.findByRole("button", {
        name: isSuggestionButton(englishQuestion),
      }),
    ).toBeInTheDocument();

    await act(async () => {
      await i18n.changeLanguage("pt");
    });

    await waitFor(() => {
      expect(
        screen.getByRole("button", {
          name: isSuggestionButton(portugueseQuestion),
        }),
      ).toBeInTheDocument();
    });
    expect(
      screen.queryByRole("button", {
        name: isSuggestionButton(englishQuestion),
      }),
    ).not.toBeInTheDocument();
  });

  it("does not show fallback questions when a catalog corpus has no published suggestion set", async () => {
    const fairHousingCorpus = getCorpus(
      "us-fair-housing-disability-accommodations",
    );
    const { openingSuggestionSet, ...corpusWithoutSuggestionSet } =
      fairHousingCorpus;

    expect(openingSuggestionSet).toBeDefined();

    renderAtRoute(
      <ResearchChat
        corpus={corpusWithoutSuggestionSet}
        onOpenCitation={vi.fn()}
      />,
    );

    await screen.findByRole("button", { name: "Send question" });
    expect(screen.queryAllByRole("button")).toHaveLength(1);
  });

  it("discards a catalog corpus suggestion set after its active release drifts", async () => {
    const lgpdCorpus = getCorpus("brazil-data-protection");
    const activeRelease = lgpdCorpus.activeRelease;

    if (!activeRelease) {
      throw new Error("Expected an active release fixture.");
    }

    const staleCorpus: CorpusFixture = {
      ...lgpdCorpus,
      activeRelease: {
        ...activeRelease,
        snapshotId: "lgpd-snapshot-v2",
      },
    };

    renderAtRoute(
      <ResearchChat corpus={staleCorpus} onOpenCitation={vi.fn()} />,
    );

    await screen.findByRole("button", { name: "Send question" });
    expect(screen.queryAllByRole("button")).toHaveLength(1);
  });

  it.each(corpora)(
    "submits the $label opening suggestion through the ordinary keyboard chat path",
    async (corpus) => {
      const user = userEvent.setup();
      const suggestion = resolveOpeningSuggestions(corpus, "en").at(0);

      if (!suggestion) {
        throw new Error(
          `Expected an English opening suggestion for ${corpus.id}.`,
        );
      }

      renderAtRoute(<ResearchChat corpus={corpus} onOpenCitation={vi.fn()} />);

      const button = await screen.findByRole("button", {
        name: isSuggestionButton(suggestion.question),
      });
      button.focus();
      await user.keyboard("{Enter}");

      expect(
        await screen.findByText(
          `This is the prepared synthetic prototype answer for ${suggestion.caseId} in the selected corpus.`,
        ),
      ).toBeInTheDocument();
    },
  );
});
