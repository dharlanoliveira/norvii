import { expect, test, type Page } from "@playwright/test";

const starterQuestionLabels = {
  en: "Suggested research questions",
  pt: "Perguntas sugeridas para pesquisa",
} as const;

const localFixtureHeaders = { "cache-control": "no-store" };

const corpusFixtures = [
  {
    id: "10000000-0000-4000-8000-000000000001",
    name: "Brazilian Personal Data Protection (LGPD)",
    language: "pt",
    jurisdiction: "Brazil",
    snapshot: snapshot("30000000-0000-4000-8000-000000000001", "a", 1),
    suggestions: {
      en: [
        {
          caseId: "lgpd-002-en",
          rank: 1,
          question:
            "What is the difference between a controller and an operator under the LGPD?",
        },
        {
          caseId: "lgpd-003-en",
          rank: 2,
          question:
            "What do the purpose and necessity principles require when processing personal data?",
        },
        {
          caseId: "lgpd-004-en",
          rank: 3,
          question:
            "Can a data subject request confirmation that processing exists and access to their data?",
        },
        {
          caseId: "lgpd-005-en",
          rank: 4,
          question:
            "What is the deadline for providing a clear and complete statement in response to a confirmation or access request?",
        },
        {
          caseId: "lgpd-007-en",
          rank: 5,
          question:
            "When must a security incident be reported to the ANPD and the data subject?",
        },
      ],
      pt: [
        {
          caseId: "lgpd-002-pt",
          rank: 1,
          question:
            "Qual \u00E9 a diferen\u00E7a entre controlador e operador na LGPD?",
        },
        {
          caseId: "lgpd-003-pt",
          rank: 2,
          question:
            "O que exigem os princ\u00EDpios da finalidade e da necessidade no tratamento de dados pessoais?",
        },
        {
          caseId: "lgpd-004-pt",
          rank: 3,
          question:
            "O titular pode solicitar confirma\u00E7\u00E3o da exist\u00EAncia de tratamento e acesso aos seus dados?",
        },
        {
          caseId: "lgpd-005-pt",
          rank: 4,
          question:
            "Qual \u00E9 o prazo para fornecer uma declara\u00E7\u00E3o clara e completa em respost\u0061 a pedido de confirma\u00E7\u00E3o ou acesso?",
        },
        {
          caseId: "lgpd-007-pt",
          rank: 5,
          question:
            "Quando um incidente de seguran\u00E7a deve ser comunicado \u00E0 ANPD e ao titular?",
        },
      ],
    },
  },
  {
    id: "10000000-0000-4000-8000-000000000003",
    name: "Brazilian Anti-Corruption and White-Collar Crime",
    language: "pt",
    jurisdiction: "Brazil",
    snapshot: snapshot("30000000-0000-4000-8000-000000000003", "b", 1),
    suggestions: {
      en: [
        {
          caseId: "brac-001-en",
          rank: 1,
          question:
            "What type of liability does Brazilian Law 12,846/2013 establish for legal entities?",
        },
        {
          caseId: "brac-002-en",
          rank: 2,
          question:
            "Can a company that indirectly offers an undue advantage to a public official commit a harmful act under the Brazilian Anti-Corruption Law?",
        },
        {
          caseId: "brac-003-en",
          rank: 3,
          question:
            "Which administrative sanctions does Article 6 of Law 12,846/2013 provide for legal entities?",
        },
        {
          caseId: "brac-005-en",
          rank: 4,
          question:
            "How does the Brazilian Penal Code define active corruption?",
        },
        {
          caseId: "brac-007-en",
          rank: 5,
          question:
            "What general anti-money-laundering duties apply to persons subject to Brazilian Law 9,613/1998?",
        },
      ],
      pt: [
        {
          caseId: "brac-001-pt",
          rank: 1,
          question:
            "Que tipo de responsabiliza\u00E7\u00E3o a Lei n\u00BA 12.846/2013 estabelece para pessoas jur\u00EDdicas?",
        },
        {
          caseId: "brac-002-pt",
          rank: 2,
          question:
            "Uma empresa que oferece indiretamente vantagem indevida a um agente p\u00FAblico pode praticar ato lesivo nos termos da Lei Anticorrup\u00E7\u00E3o?",
        },
        {
          caseId: "brac-003-pt",
          rank: 3,
          question:
            "Quais san\u00E7\u00F5es administrativas o art. 6\u00BA da Lei n\u00BA 12.846/2013 prev\u00EA para pessoas jur\u00EDdicas?",
        },
        {
          caseId: "brac-005-pt",
          rank: 4,
          question:
            "Como o C\u00F3digo Penal define corrup\u00E7\u00E3o ativa?",
        },
        {
          caseId: "brac-007-pt",
          rank: 5,
          question:
            "Quais deveres gerais de preven\u00E7\u00E3o \u00E0 lavagem de dinheiro recaem sobre as pessoas obrigadas pela Lei n\u00BA 9.613/1998?",
        },
      ],
    },
  },
  {
    id: "10000000-0000-4000-8000-000000000004",
    name: "United States Fair Housing and Disability Accommodations",
    language: "en",
    jurisdiction: "United States",
    snapshot: snapshot("30000000-0000-4000-8000-000000000004", "c", 1),
    suggestions: {
      en: [
        {
          caseId: "fh-002-en",
          rank: 1,
          question:
            "Does the Fair Housing Act address discrimination against a buyer or renter because of disability?",
        },
        {
          caseId: "fh-003-en",
          rank: 2,
          question:
            "What is a reasonable accommodation under the Fair Housing Act?",
        },
        {
          caseId: "fh-004-en",
          rank: 3,
          question:
            "How does a reasonable modification differ from a reasonable accommodation in existing housing?",
        },
        {
          caseId: "fh-005-en",
          rank: 4,
          question:
            "Can an assistance animal be treated as an ordinary pet for a housing accommodation request?",
        },
        {
          caseId: "fh-007-en",
          rank: 5,
          question:
            "What information does HUD ask a person to provide when reporting housing discrimination?",
        },
      ],
      pt: [
        {
          caseId: "fh-002-pt",
          rank: 1,
          question:
            "A Fair Housing Act trata da discrimina\u00E7\u00E3o contra comprador ou inquilino por causa de defici\u00EAncia?",
        },
        {
          caseId: "fh-003-pt",
          rank: 2,
          question:
            "O que \u00E9 uma acomoda\u00E7\u00E3o razo\u00E1vel sob a Fair Housing Act?",
        },
        {
          caseId: "fh-004-pt",
          rank: 3,
          question:
            "Como uma modifica\u00E7\u00E3o razo\u00E1vel difere de uma acomoda\u00E7\u00E3o razo\u00E1vel em moradia existente?",
        },
        {
          caseId: "fh-005-pt",
          rank: 4,
          question:
            "Um animal de assist\u00EAncia pode ser tratado como animal de estima\u00E7\u00E3o comum em pedido de acomoda\u00E7\u00E3o habitacional?",
        },
        {
          caseId: "fh-007-pt",
          rank: 5,
          question:
            "Quais informa\u00E7\u00F5es a HUD pede quando algu\u00E9m comunica discrimina\u00E7\u00E3o habitacional?",
        },
      ],
    },
  },
] as const;

type InterfaceLanguage = keyof typeof starterQuestionLabels;
type CorpusFixture = (typeof corpusFixtures)[number];

test.beforeEach(async ({ page }) => configureCorpusOpeningSuggestionsAPI(page));

for (const fixture of corpusFixtures) {
  for (const language of Object.keys(
    starterQuestionLabels,
  ) as InterfaceLanguage[]) {
    test(`shows only the ${language} starter selection and submits it for ${fixture.name}`, async ({
      page,
    }) => {
      await page.goto("/");
      await page
        .getByRole("link", { name: `Open corpus ${fixture.name}` })
        .click();
      await page
        .getByRole("combobox", { name: "Interface language" })
        .selectOption(language);

      const starterQuestions = page.getByLabel(starterQuestionLabels[language]);
      await expect(starterQuestions.getByRole("button")).toHaveText(
        fixture.suggestions[language].map(({ question }) => question),
      );
      await expect(starterQuestions.getByRole("button")).toHaveCount(5);

      await starterQuestions.getByRole("button").first().click();

      await expect(
        page.getByLabel(starterQuestionLabels[language]),
      ).toHaveCount(0);
      await expect(
        page.getByText(fixture.suggestions[language][0].question),
      ).toBeVisible();
      await expect
        .poll(() => chatSubmissions(page))
        .toEqual([
          {
            corpusId: fixture.id,
            body: JSON.stringify({
              question: fixture.suggestions[language][0].question,
              interfaceLanguage: language,
              strategy: "hybrid",
            }),
          },
        ]);
    });
  }
}

test("hides stale suggestions after the active snapshot changes", async ({
  page,
}) => {
  const fixture = corpusFixtures[0];
  const nextSnapshot = snapshot("30000000-0000-4000-8000-000000000101", "d", 2);
  let corpusRequestCount = 0;
  let suggestionRequestCount = 0;

  await page.route(`**/api/v1/corpora/${fixture.id}`, async (route) => {
    corpusRequestCount += 1;
    await route.fulfill({
      json: corpusResponse(
        fixture,
        corpusRequestCount === 1 ? fixture.snapshot : nextSnapshot,
      ),
      headers: localFixtureHeaders,
    });
  });
  await page.route(
    `**/api/v1/corpora/${fixture.id}/opening-suggestions?interfaceLanguage=en`,
    async (route) => {
      suggestionRequestCount += 1;
      await route.fulfill({
        json: openingSuggestionResponse(fixture, "en", fixture.snapshot),
        headers: localFixtureHeaders,
      });
    },
  );

  await page.goto(`/corpora/${fixture.id}`);
  await expect(
    page.getByLabel(starterQuestionLabels.en).getByRole("button"),
  ).toHaveText(fixture.suggestions.en.map(({ question }) => question));

  await page.reload();

  await expect.poll(() => corpusRequestCount).toBe(2);
  await expect.poll(() => suggestionRequestCount).toBe(2);

  await expect(page.getByLabel(starterQuestionLabels.en)).toHaveCount(0);
  await expect(
    page.getByText(fixture.suggestions.en[0].question, { exact: true }),
  ).toHaveCount(0);
  await expect(
    page.getByText(corpusFixtures[1].suggestions.en[0].question, {
      exact: true,
    }),
  ).toHaveCount(0);
});

const submissionsByPage = new WeakMap<Page, ChatSubmission[]>();

interface ChatSubmission {
  readonly corpusId: string;
  readonly body: string | null;
}

async function configureCorpusOpeningSuggestionsAPI(page: Page): Promise<void> {
  submissionsByPage.set(page, []);
  await page.route("**/api/v1/corpora?includeDisabled=true", async (route) =>
    route.fulfill({
      json: corpusFixtures.map((fixture) =>
        corpusResponse(fixture, fixture.snapshot),
      ),
      headers: localFixtureHeaders,
    }),
  );
  for (const fixture of corpusFixtures) {
    await page.route(`**/api/v1/corpora/${fixture.id}`, async (route) =>
      route.fulfill({
        json: corpusResponse(fixture, fixture.snapshot),
        headers: localFixtureHeaders,
      }),
    );
    await page.route(`**/api/v1/corpora/${fixture.id}/sources`, async (route) =>
      route.fulfill({ json: [], headers: localFixtureHeaders }),
    );
    await page.route(
      `**/api/v1/corpora/${fixture.id}/opening-suggestions?interfaceLanguage=*`,
      async (route) => {
        const interfaceLanguage = requestInterfaceLanguage(
          route.request().url(),
        );
        await route.fulfill({
          json: openingSuggestionResponse(
            fixture,
            interfaceLanguage,
            fixture.snapshot,
          ),
          headers: localFixtureHeaders,
        });
      },
    );
    await page.route(
      `**/api/v1/corpora/${fixture.id}/chat/stream`,
      async (route) => {
        submissionsByPage.get(page)?.push({
          corpusId: fixture.id,
          body: route.request().postData(),
        });
        await route.fulfill({
          body: completedChatStream(fixture.id),
          contentType: "text/event-stream",
        });
      },
    );
  }
}

function chatSubmissions(page: Page): readonly ChatSubmission[] {
  return submissionsByPage.get(page) ?? [];
}

function requestInterfaceLanguage(url: string): InterfaceLanguage {
  const interfaceLanguage = new URL(url).searchParams.get("interfaceLanguage");
  if (interfaceLanguage === "en" || interfaceLanguage === "pt") {
    return interfaceLanguage;
  }
  throw new Error(
    "Opening-suggestion requests must declare a supported language.",
  );
}

function corpusResponse(
  fixture: CorpusFixture,
  activeSnapshot: ReturnType<typeof snapshot>,
) {
  return {
    id: fixture.id,
    name: fixture.name,
    description: `Local fixture for ${fixture.name}.`,
    language: fixture.language,
    jurisdiction: fixture.jurisdiction,
    status: "enabled",
    sourceCount: 0,
    version: 1,
    createdAt: "2026-08-26T12:00:00Z",
    updatedAt: "2026-08-26T12:00:00Z",
    activeSnapshot,
  };
}

function openingSuggestionResponse(
  fixture: CorpusFixture,
  interfaceLanguage: InterfaceLanguage,
  activeSnapshot: ReturnType<typeof snapshot>,
) {
  return {
    corpusId: fixture.id,
    activeSnapshotId: activeSnapshot.id,
    activeSnapshotManifestSha256: activeSnapshot.manifestSha256,
    interfaceLanguage,
    suggestions: fixture.suggestions[interfaceLanguage],
  };
}

function snapshot(id: string, hashCharacter: string, releaseVersion: number) {
  return {
    id,
    manifestSha256: hashCharacter.repeat(64),
    createdAt: "2026-08-26T12:00:00Z",
    activatedAt: "2026-08-26T12:00:00Z",
    releaseVersion,
  };
}

function completedChatStream(corpusId: string): string {
  return [
    { type: "started", requestId: "synthetic-request", corpusId },
    {
      type: "completed",
      requestId: "synthetic-request",
      answer: "Synthetic local response.",
      references: [],
      telemetry: {
        outcome: "completed",
        evidenceCount: 0,
        durationMilliseconds: 1,
      },
    },
  ]
    .map((event) => `data: ${JSON.stringify(event)}\n\n`)
    .join("");
}
