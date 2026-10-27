import { test, expect, type Page } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";

const origin = "https://example.org";

// Two Japanese pages. "accessible" has the word アクセシビリティ; "scattered"
// has every one of its characters, but never two of them side by side in the
// order of that word. One token per character matches both pages for the
// query; pairs of characters match only the first.
const pages: Record<string, string> = {
  accessible: "アクセシビリティを備えたウィジェットです。",
  scattered: "アルバム、クリーム、セール、シリーズ、ビール、リボン、テーブル、フィルム。",
};

function buildSite(cjk: string | undefined) {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-search-cjk-"));
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
defaultContentLanguage: ja
hasCJKLanguage: true
languages:
  ja:
    languageName: 日本語
menu:
  main:
    - name: Search
      weight: 1
      params:
        type: search
${cjk ? `params:\n  search:\n    flexsearch:\n      cjk: ${cjk}\n` : ""}`
  );
  mkdirSync(contentDir, { recursive: true });
  writeFileSync(join(contentDir, "_index.md"), "---\ntitle: ホーム\n---\n");
  for (const [name, text] of Object.entries(pages)) {
    writeFileSync(join(contentDir, `${name}.md`), `---\ntitle: ${name}\n---\n${text}\n`);
  }

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

// Serves the built site at the origin, without a web server.
async function serve(page: Page, publishDir: string) {
  await page.route(`${origin}/**`, async (route) => {
    let path = decodeURIComponent(new URL(route.request().url()).pathname);
    if (path.endsWith("/")) path += "index.html";
    const file = join(publishDir, path);
    if (!existsSync(file)) return route.fulfill({ status: 404 });
    return route.fulfill({
      body: readFileSync(file),
      contentType: contentTypes[extname(file)] || "application/octet-stream",
    });
  });
}

// The pages the query finds, by path.
async function search(page: Page, query: string) {
  await page.goto(`${origin}/`);
  await page.keyboard.press("/");
  await page.locator("#hextra-search-dialog .hextra-search-input").fill(query);
  const results = page.locator("#hextra-search-dialog .hextra-search-results a[role=option]");
  // A query with no hit shows no option: give the index time to answer.
  await page.waitForTimeout(1500);
  const hrefs = await results.evaluateAll((links) => links.map((a) => new URL((a as HTMLAnchorElement).href).pathname));
  return [...new Set(hrefs)].sort();
}
// "scattered" has no two of the word's characters side by side, in any order
// that makes one of its pairs, so no pair of the query is in it.

test("a CJK query matches the pages holding the characters next to each other", async ({ page }) => {
  const { siteDir, publishDir } = buildSite(undefined);
  try {
    await serve(page, publishDir);
    expect(await search(page, "アクセシビリティ")).toEqual(["/accessible/"]);
    // The same characters in another order are not a match.
    expect(await search(page, "ィテリビシセクア")).toEqual([]);
    // One character still finds the page it begins a pair in.
    expect(await search(page, "ウ")).toEqual(["/accessible/"]);
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("cjk: char keeps one token per character", async ({ page }) => {
  const { siteDir, publishDir } = buildSite("char");
  try {
    await serve(page, publishDir);
    // Every character of the word is on both pages, so both match.
    expect(await search(page, "アクセシビリティ")).toEqual(["/accessible/", "/scattered/"]);
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});
