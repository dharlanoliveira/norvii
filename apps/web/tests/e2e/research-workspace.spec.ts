import { expect, test, type Page } from "@playwright/test";

import { portugueseTranslation } from "../../src/i18n/pt/translation";

const approvedViewports = [
  { name: "notebook", width: 1280, height: 720 },
  { name: "desktop", width: 1440, height: 900 },
] as const;

async function expectNoHorizontalPageScroll(page: Page): Promise<void> {
  const hasHorizontalScroll = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(hasHorizontalScroll).toBe(false);
}

for (const viewport of approvedViewports) {
  test(`catalog and workspace remain usable at ${viewport.name} size`, async ({
    page,
  }) => {
    await page.setViewportSize(viewport);
    const navigationStartedAt = Date.now();
    await page.goto("/");

    await expect(page.locator("html")).toHaveAttribute("lang", "en");
    await expect(page.getByRole("article")).toHaveCount(2);
    expect(Date.now() - navigationStartedAt).toBeLessThan(2_000);
    if (viewport.name === "notebook") {
      const interactiveMilliseconds = await page.evaluate(() =>
        Math.round(performance.now()),
      );
      expect(interactiveMilliseconds).toBeLessThan(2_000);
    }
    await expectNoHorizontalPageScroll(page);
    await expect(page).toHaveScreenshot(`catalog-${viewport.name}.png`, {
      animations: "disabled",
      fullPage: true,
      maxDiffPixelRatio: 0.001,
    });

    await page
      .getByRole("link", {
        name: "Open corpus European Data Protection Framework",
      })
      .click();
    await expect(
      page.getByRole("heading", {
        name: "European Data Protection Framework",
      }),
    ).toBeVisible();
    await expectNoHorizontalPageScroll(page);
    await expect(page).toHaveScreenshot(`workspace-${viewport.name}.png`, {
      animations: "disabled",
      fullPage: true,
      maxDiffPixelRatio: 0.001,
    });
  });
}

test("keyboard research journey preserves context and follows a citation", async ({
  page,
}) => {
  await page.setViewportSize(approvedViewports[0]);
  await page.goto("/corpora/eu-data-protection");

  const pdf = page.getByRole("treeitem", {
    name: "PDF: General Data Protection Regulation",
  });
  await pdf.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("tab", { name: "Source" })).toHaveAttribute(
    "aria-selected",
    "true",
  );

  const sourceTab = page.getByRole("tab", { name: "Source" });
  await sourceTab.focus();
  await page.keyboard.press("ArrowRight");
  await expect(page.getByRole("tab", { name: "Chat" })).toBeFocused();

  const question = page.getByRole("textbox", { name: "Research question" });
  await question.fill("What principles govern personal data processing?");
  await page.getByRole("button", { name: "Send question" }).click();
  await expect(
    page.getByText(/Article 5 establishes lawfulness/),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Open citation GDPR, Article 5" })
    .click();

  await expect(page.getByRole("tab", { name: "Source" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await expect(
    page.getByText("Article 5", { exact: true }).last(),
  ).toBeVisible();
  await page.getByRole("tab", { name: "Chat" }).click();
  await expect(
    page.getByText(/Article 5 establishes lawfulness/),
  ).toBeVisible();
});

test("locale switching and unknown-route recovery are localized", async ({
  page,
}) => {
  await page.goto("/corpora/eu-data-protection");
  const question = page.getByRole("textbox", { name: "Research question" });
  await question.fill("Preserved draft");
  await page
    .getByRole("combobox", { name: "Interface language" })
    .selectOption("pt");

  await expect(
    page.getByRole("textbox", {
      name: portugueseTranslation.chat.composerLabel,
    }),
  ).toHaveValue("Preserved draft");
  await expect(page).toHaveURL(/\/corpora\/eu-data-protection$/);

  await page.goto("/corpora/not-a-corpus");
  await page
    .getByRole("combobox", { name: "Interface language" })
    .selectOption("pt");
  await expect(
    page.getByRole("heading", {
      name: portugueseTranslation.errors.unknownCorpusTitle,
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", {
      name: portugueseTranslation.errors.returnToCatalog,
    }),
  ).toBeVisible();
});
