import { test, expect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// A planted site-provided flag, to tell it apart from the downloaded ones.
const plantedFlag = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 4 3"><rect width="4" height="3" fill="#123456"/></svg>`;

type Language = { label: string; flag?: string };

function buildSite(languages: Record<string, Language>, extra = "", assets: Record<string, string> = {}) {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-language-flags-"));
  const publishDir = join(siteDir, "public");
  const themesDir = join(siteDir, "themes");
  mkdirSync(themesDir);
  symlinkSync(process.cwd(), join(themesDir, "hextra"), "dir");

  let config = `title: Test\nbaseURL: https://example.org/\ntheme: hextra\ndefaultContentLanguage: en\n${extra}languages:\n`;
  let weight = 1;
  for (const [lang, { label, flag }] of Object.entries(languages)) {
    config += `  ${lang}:\n    label: ${label}\n    weight: ${weight++}\n`;
    if (flag) config += `    params:\n      flag: "${flag}"\n`;
  }
  writeFileSync(join(siteDir, "hugo.yaml"), config);
  for (const [path, body] of Object.entries(assets)) {
    mkdirSync(join(siteDir, "assets", path, ".."), { recursive: true });
    writeFileSync(join(siteDir, "assets", path), body);
  }

  const run = () =>
    execFileSync("hugo", ["--source", siteDir, "--themesDir", themesDir, "--destination", publishDir, "--panicOnWarning"], {
      cwd: process.cwd(),
      stdio: "pipe",
    });
  return { siteDir, publishDir, run };
}

// The flag image sources of the language switch on the home page of lang.
function flagSources(publishDir: string, lang: string) {
  const home = lang === "en" ? join(publishDir, "index.html") : join(publishDir, lang, "index.html");
  const html = readFileSync(home, "utf8");
  return [...html.matchAll(/<img src="?([^" >]*\/flags\/[^" >]+)"?[^>]*>/g)].map((m) => m[1]);
}

test("flag codes are fetched once, published with the site and shown for every language", () => {
  const { siteDir, publishDir, run } = buildSite({
    en: { label: "English", flag: "gb" },
    fr: { label: "Français", flag: "FR" },
    gd: { label: "Gàidhlig", flag: "gb-sct" },
    eo: { label: "Esperanto" },
  });
  try {
    run();
    const sources = flagSources(publishDir, "en");
    // Each switch (sidebar, footer) shows the current language's flag on its
    // button, then one per option that has one.
    const perSwitch = ["/flags/gb.svg", "/flags/gb.svg", "/flags/fr.svg", "/flags/gb-sct.svg"];
    expect(sources).toEqual([...perSwitch, ...perSwitch]);
    for (const code of ["gb", "fr", "gb-sct"]) {
      const svg = readFileSync(join(publishDir, "flags", `${code}.svg`), "utf8");
      expect(svg).toContain(`id="flag-icons-${code}"`);
      expect(svg).not.toContain("<script");
    }
    expect(flagSources(publishDir, "fr")[0]).toBe("/flags/fr.svg");
    expect(existsSync(join(publishDir, "flags", "eo.svg"))).toBe(false);
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("a site's assets/flags/<code>.svg wins, and builds offline when remote flags are disabled", () => {
  const { siteDir, publishDir, run } = buildSite(
    { en: { label: "English", flag: "gb" }, fr: { label: "Français", flag: "fr" } },
    "params:\n  flags:\n    remote:\n      enable: false\n",
    { "flags/gb.svg": plantedFlag, "flags/fr.svg": plantedFlag }
  );
  try {
    run();
    expect(readFileSync(join(publishDir, "flags", "gb.svg"), "utf8")).toContain("#123456");
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

test("an image path is used as is", () => {
  const { siteDir, publishDir, run } = buildSite({
    en: { label: "English", flag: "images/flags/gb.svg" },
    fr: { label: "Français" },
  });
  try {
    run();
    expect(flagSources(publishDir, "en")[0]).toBe("/images/flags/gb.svg");
  } finally {
    rmSync(siteDir, { recursive: true, force: true });
  }
});

for (const [name, flag, extra, message] of [
  ["an invalid code", "fr_FR", "", /invalid flag/],
  ["an unknown code", "zz", "", /could not retrieve|not found/],
  ["a missing local flag with remote flags disabled", "fr", "params:\n  flags:\n    remote:\n      enable: false\n", /remote\.enable is false/],
] as const) {
  test(`${name} fails the build with a clear message`, () => {
    const { siteDir, run } = buildSite({ en: { label: "English", flag }, fr: { label: "Français" } }, extra);
    try {
      let stderr = "";
      try {
        run();
      } catch (e) {
        stderr = String((e as { stderr?: Buffer }).stderr ?? e);
      }
      expect(stderr).toMatch(message);
    } finally {
      rmSync(siteDir, { recursive: true, force: true });
    }
  });
}
