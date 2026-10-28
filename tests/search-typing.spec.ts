import { test, expect, type Page } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";

const origin = "https://example.org";
const word = "toolkit";

function buildSite() {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-search-typing-"));
  const contentDir = join(siteDir, "content");
  const publishDir = join(siteDir, "public");
  const themesDir = join(siteDir, "themes");

  mkdirSync(themesDir);
  symlinkSync(process.cwd(), join(themesDir, "hextra"), "dir");
  writeFileSync(
    join(siteDir, "hugo.yaml"),
    `title: Test
baseURL: ${origin}/
theme: hextra
menu:
  main:
    - name: Search
      weight: 1
      params:
        type: search
`
  );
  mkdirSync(contentDir, { recursive: true });
  writeFileSync(join(contentDir, "_index.md"), "---\ntitle: Home\n---\n");
  writeFileSync(join(contentDir, "page.md"), `---\ntitle: The page\n---\nA ${word} of widgets.\n`);

  execFileSync("hugo", ["--source", siteDir, "--themesDir", themesDir, "--destination", publishDir, "--environment", "development"], {
    cwd: process.cwd(),
    stdio: "pipe",
  });
  return { siteDir, publishDir };
}

const contentTypes: Record<string, string> = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
};

// Serves the built site at the origin; the search data answers after delayMs.
async function serve(page: Page, publishDir: string, delayMs: number) {
  await page.route(`${origin}/**`, async (route) => {
    let path = decodeURIComponent(new URL(route.request().url()).pathname);
    if (path.endsWith("/")) path += "index.html";
    const file = join(publishDir, path);
    if (!existsSync(file)) return route.fulfill({ status: 404 });
    if (delayMs && path.includes("search-data")) await new Promise((r) => setTimeout(r, delayMs));
    return route.fulfill({
      body: readFileSync(file),
      contentType: contentTypes[extname(file)] || "application/octet-stream",
    });
  });
}

const input = (page: Page) => page.locator("#hextra-search-dialog .hextra-search-input");
const results = (page: Page) => page.locator("#hextra-search-dialog .hextra-search-results a[role=option]");

test("characters typed as the dialog opens are all kept", async ({ page }) => {
  const { siteDir, publishDir } = buildSite();
  try {
    await serve(page, publishDir, 0);
    await page.goto(`${origin}/`);
    await page.keyboard.press("/");
    // The first character goes in at once; the dialog's next frame must not
    // select it, or the second character would replace it ("oolkit").
    await page.keyboard.type(word[0]);
    await page.evaluate(() => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))));
    await page.keyboard.type(word.slice(1));
    await expect(input(page)).toHaveValue(word);
    await expect(results(page).first()).toBeVisible();
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("a query typed before the index is ready runs when it is, and says it is loading", async ({ page }) => {
  const { siteDir, publishDir } = buildSite();
  try {
    await serve(page, publishDir, 1500);
    await page.goto(`${origin}/`);
    await page.keyboard.press("/");
    await page.keyboard.type(word);
    await expect(input(page)).toHaveValue(word);
    // While the index loads: a loading status, not "no results".
    const status = page.locator("#hextra-search-dialog .hextra-search-status");
    await expect(status).toContainText("Loading");
    await expect(page.locator("#hextra-search-dialog [data-search-empty]")).toBeHidden();
    // Then the pending query is answered, without typing again.
    await expect(results(page).first()).toBeVisible({ timeout: 10_000 });
    await expect(status).not.toContainText("No results");
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});
