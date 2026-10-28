import { test, expect, type Page } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { extname, join } from "node:path";

const origin = "https://example.org";
const versionsJson = [
  { version: "1.0", title: "1.0", aliases: ["latest"] },
  { version: "0.9", title: "0.9", aliases: [] },
  { version: "development", title: "development", aliases: [], properties: { development: true } },
];

// One build per version, as the documentation sites publish them: the baseURL
// ends with the version, and the home calls the banner component.
function build(siteDir: string, version: string) {
  const publishDir = join(siteDir, `public-${version}`);
  execFileSync(
    "hugo",
    [
      "--source", siteDir,
      "--themesDir", join(siteDir, "themes"),
      "--destination", publishDir,
      "--baseURL", `${origin}/docs/${version}/`,
      "--environment", "development",
    ],
    { cwd: process.cwd(), stdio: "pipe" }
  );
  return publishDir;
}

function buildSite() {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-version-banner-"));
  const themesDir = join(siteDir, "themes");
  mkdirSync(themesDir);
  symlinkSync(process.cwd(), join(themesDir, "hextra"), "dir");
  writeFileSync(join(siteDir, "hugo.yaml"), `title: Test\nbaseURL: ${origin}/docs/1.0/\ntheme: hextra\n`);
  mkdirSync(join(siteDir, "content"), { recursive: true });
  writeFileSync(join(siteDir, "content", "_index.md"), "---\ntitle: Home\n---\nHello.\n");
  const hookDir = join(siteDir, "layouts", "_partials", "custom");
  mkdirSync(hookDir, { recursive: true });
  writeFileSync(join(hookDir, "content-begin.html"), '{{ partial "components/version-banner.html" . }}\n');
  const dirs: Record<string, string> = {};
  for (const v of ["1.0", "0.9", "development"]) dirs[v] = build(siteDir, v);
  return { siteDir, dirs };
}

const contentTypes: Record<string, string> = {
  ".html": "text/html",
  ".js": "text/javascript",
  ".css": "text/css",
  ".json": "application/json",
  ".svg": "image/svg+xml",
};

// Serves /docs/<version>/ from that version's build and /docs/versions.json
// from the test (a 404 when versions is null).
async function serve(page: Page, dirs: Record<string, string>, versions: unknown) {
  await page.route(`${origin}/**`, async (route) => {
    const path = decodeURIComponent(new URL(route.request().url()).pathname);
    if (path === "/docs/versions.json") {
      if (versions === null) return route.fulfill({ status: 404 });
      return route.fulfill({ body: JSON.stringify(versions), contentType: "application/json" });
    }
    const m = path.match(/^\/docs\/([^/]+)\/(.*)$/);
    if (!m || !dirs[m[1]]) return route.fulfill({ status: 404 });
    let rel = m[2];
    if (rel === "" || rel.endsWith("/")) rel += "index.html";
    const file = join(dirs[m[1]], rel);
    if (!existsSync(file)) return route.fulfill({ status: 404 });
    return route.fulfill({
      body: readFileSync(file),
      contentType: contentTypes[extname(file)] || "application/octet-stream",
    });
  });
}

async function banner(page: Page, dirs: Record<string, string>, version: string, versions: unknown) {
  await serve(page, dirs, versions);
  await page.goto(`${origin}/docs/${version}/`);
  // The banner is revealed after versions.json is read: give it time to not appear.
  await page.waitForTimeout(1000);
  return page.locator(".hextra-version-banner");
}

test("the version banner follows versions.json", async ({ page }) => {
  const { siteDir, dirs } = buildSite();
  try {
    // The version holding the alias "latest": no banner.
    await expect(await banner(page, dirs, "1.0", versionsJson)).toBeHidden();

    // An older version: the outdated notice, linking to the docs root.
    let b = await banner(page, dirs, "0.9", versionsJson);
    await expect(b).toBeVisible();
    await expect(b.locator('[data-text="outdated"]')).toBeVisible();
    await expect(b.locator('[data-text="outdated"]')).toContainText("outdated version");
    await expect(b.locator('[data-text="development"]')).toBeHidden();
    await expect(b.locator("a")).toHaveAttribute("href", "/docs/");

    // The development version: its own notice.
    b = await banner(page, dirs, "development", versionsJson);
    await expect(b).toBeVisible();
    await expect(b.locator('[data-text="development"]')).toBeVisible();
    await expect(b.locator('[data-text="development"]')).toContainText("development version");
    await expect(b.locator('[data-text="outdated"]')).toBeHidden();

    // No versions.json (a local build): no banner.
    await expect(await banner(page, dirs, "0.9", null)).toBeHidden();
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("a version whose alias moved is outdated the next time the page is read", async ({ page }) => {
  const { siteDir, dirs } = buildSite();
  try {
    // 1.0 is latest: no banner. Then a release moves "latest" to 1.1: the same
    // built page now shows the notice, with no rebuild.
    await expect(await banner(page, dirs, "1.0", versionsJson)).toBeHidden();
    const moved = [{ version: "1.1", title: "1.1", aliases: ["latest"] }, ...versionsJson.map((v) => ({ ...v, aliases: [] }))];
    const b = await banner(page, dirs, "1.0", moved);
    await expect(b).toBeVisible();
    await expect(b).toContainText("outdated version");
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});
