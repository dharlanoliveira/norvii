import { expect, test } from "@playwright/test";

test("opens a corpus, preserves chat state, and navigates a citation", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.goto("/");

  await expect(
    page.getByRole("heading", { name: /choose the body of law/i }),
  ).toBeVisible();
  await page
    .getByRole("article")
    .first()
    .getByRole("link", { name: /open corpus/i })
    .click();
  await expect(
    page.getByRole("heading", { name: "Brazilian Data Protection" }),
  ).toBeVisible();

  await page.getByRole("treeitem", { name: /lgpd.*available/i }).click();
  await expect(page.getByText("Current location")).toBeVisible();

  await page.getByRole("tab", { name: "Chat" }).click();
  const composer = page.getByRole("textbox", {
    name: /question for this corpus/i,
  });
  await composer.fill("Draft question");
  await page.getByRole("tab", { name: "Source" }).click();
  await page.getByRole("tab", { name: "Chat" }).click();
  await expect(composer).toHaveValue("Draft question");

  await composer.fill("");
  await page
    .getByRole("button", { name: /what rights does a data subject/i })
    .click();
  await page
    .getByRole("button", { name: /open citation lgpd, art. 18/i })
    .click();
  await expect(
    page.getByRole("heading", { name: "Direitos do titular" }),
  ).toBeVisible();

  await page.getByRole("tab", { name: "Chat" }).click();
  await expect(
    page.getByText(/confirmation that processing exists/i),
  ).toBeVisible();
});

test("changes interface language without translating legal content", async ({
  page,
}) => {
  await page.goto("/corpora/brazil-data-protection");
  await page.getByRole("treeitem", { name: /lgpd.*available/i }).click();

  await page
    .getByRole("combobox", { name: /interface language/i })
    .selectOption("pt");

  await expect(page.getByRole("tab", { name: "Fonte" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  await expect(page.locator("html")).toHaveAttribute("lang", "pt");
  await expect(page.getByText("Publicado por")).toBeVisible();
});

test("completes the primary research journey with keyboard activation", async ({
  page,
}) => {
  await page.goto("/");

  const openCorpus = page
    .getByRole("article")
    .first()
    .getByRole("link", { name: /open corpus/i });
  await openCorpus.focus();
  await page.keyboard.press("Enter");

  const source = page.getByRole("treeitem", { name: /lgpd.*available/i });
  await source.focus();
  await page.keyboard.press("Enter");
  await expect(page.getByText("Current location")).toBeVisible();

  const chatTab = page.getByRole("tab", { name: "Chat" });
  await chatTab.focus();
  await page.keyboard.press("Enter");

  const composer = page.getByRole("textbox", {
    name: /question for this corpus/i,
  });
  await composer.focus();
  await page.keyboard.type("What rights does a data subject have?");
  await page.keyboard.press("Enter");

  const citation = page.getByRole("button", {
    name: /open citation lgpd, art. 18/i,
  });
  await citation.focus();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Direitos do titular" }),
  ).toBeVisible();
});

for (const viewport of [
  { name: "notebook", width: 1280, height: 720 },
  { name: "wide-desktop", width: 1440, height: 900 },
] as const) {
  test(`matches the ${viewport.name} catalog and workspace baseline`, async ({
    page,
  }) => {
    await page.setViewportSize({
      width: viewport.width,
      height: viewport.height,
    });
    await page.goto("/");
    await expect(page).toHaveScreenshot(`${viewport.name}-catalog.png`, {
      animations: "disabled",
      fullPage: true,
    });

    await page
      .getByRole("article")
      .nth(1)
      .getByRole("link", { name: /open corpus/i })
      .click();
    await expect(page).toHaveScreenshot(`${viewport.name}-workspace.png`, {
      animations: "disabled",
      fullPage: true,
    });
  });
}
