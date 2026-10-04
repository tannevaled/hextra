import { test, expect, type Page } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";

const origin = "https://example.org";
const token = "SCOPEDTOKEN";

// Builds a site with two sections, v1 and v2, whose pages share a token.
// With withHook, layouts/_partials/custom/search-scope.html scopes search to
// the current page's first section.
function buildSite(withHook: boolean) {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-search-scope-"));
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
  for (const version of ["v1", "v2"]) {
    mkdirSync(join(contentDir, version), { recursive: true });
    writeFileSync(join(contentDir, version, "_index.md"), `---\ntitle: Version ${version}\n---\n`);
    writeFileSync(join(contentDir, version, "page.md"), `---\ntitle: Page ${version}\n---\n${token} in ${version}.\n`);
  }
  if (withHook) {
    const hookDir = join(siteDir, "layouts", "_partials", "custom");
    mkdirSync(hookDir, { recursive: true });
    writeFileSync(join(hookDir, "search-scope.html"), "{{- return .FirstSection -}}\n");
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

async function searchFrom(page: Page, publishDir: string, url: string) {
  await serve(page, publishDir);
  await page.goto(`${origin}${url}`);
  await page.keyboard.press("/");
  await page.locator("#hextra-search-dialog .hextra-search-input").fill(token);
  const results = page.locator("#hextra-search-dialog .hextra-search-results a[role=option]");
  await expect(results.first()).toBeVisible();
  const hrefs = await results.evaluateAll((links) => links.map((a) => new URL((a as HTMLAnchorElement).href).pathname));
  return [...new Set(hrefs.map((h) => h.split("/")[1]))].sort();
}

test("without the hook the dialog has no scope attribute and search covers every section", async ({ page }) => {
  const { siteDir, publishDir } = buildSite(false);
  try {
    const html = readFileSync(join(publishDir, "v1", "page", "index.html"), "utf8");
    const dialog = html.match(/<dialog[^>]*id="?hextra-search-dialog[^>]*>/)?.[0] ?? "";
    expect(dialog).not.toBe("");
    expect(dialog).not.toContain("data-search-scope");
    expect(dialog.toLowerCase()).not.toContain("zgotmplz");

    expect(await searchFrom(page, publishDir, "/v1/page/")).toEqual(["v1", "v2"]);
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("custom/search-scope.html restricts search to the returned section", async ({ page }) => {
  const { siteDir, publishDir } = buildSite(true);
  try {
    const html = readFileSync(join(publishDir, "v1", "page", "index.html"), "utf8");
    expect(html).toMatch(/data-search-scope="?\/v1\/"?/);
    expect(html.toLowerCase()).not.toContain("zgotmplz");

    expect(await searchFrom(page, publishDir, "/v1/page/")).toEqual(["v1"]);
    expect(await searchFrom(page, publishDir, "/v2/page/")).toEqual(["v2"]);
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});
