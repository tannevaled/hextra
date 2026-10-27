import { test, expect } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, symlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import vm from "node:vm";

// Builds a site at https://example.org/docs/<current>/ with the version
// selector in its navbar, and returns its home page.
function buildHome(current: string, extra = "") {
  const siteDir = mkdtempSync(join(tmpdir(), "hextra-version-select-"));
  const themesDir = join(siteDir, "themes");
  mkdirSync(themesDir);
  symlinkSync(process.cwd(), join(themesDir, "hextra"), "dir");
  writeFileSync(
    join(siteDir, "hugo.yaml"),
    `title: Test
baseURL: https://example.org/docs/${current}/
theme: hextra
${extra}
menu:
  main:
    - name: Version
      weight: 1
      params: { type: partial, partial: components/version-select.html }
`,
  );
  const publishDir = join(siteDir, "public");
  execFileSync("hugo", ["--source", siteDir, "--themesDir", themesDir, "--destination", publishDir, "--panicOnWarning"], {
    cwd: process.cwd(),
    stdio: "pipe",
  });
  return readFileSync(join(publishDir, "index.html"), "utf8");
}

// Runs the component's inline script against a stub DOM and a stub fetch.
// Returns the entries it listed and a click() that reports where it navigated.
async function runScript(html: string, versions: unknown, existing: Set<string>) {
  const block = html.match(/<div class="hextra-version-select[\s\S]*?<\/script>/);
  expect(block, "component rendered").not.toBeNull();
  const markup = block![0];
  const attr = (n: string) => markup.match(new RegExp(`${n}="([^"]*)"`))![1];
  const dataset: Record<string, string> = {
    docsRoot: attr("data-docs-root"),
    current: attr("data-current"),
    rest: attr("data-rest"),
    restDefault: attr("data-rest-default"),
    langPrefix: attr("data-lang-prefix"),
    dev: attr("data-dev"),
  };
  const el = (): any => {
    const e: any = { children: [], dataset: {}, attrs: {}, className: "", innerHTML: "", textContent: "", listeners: {} };
    e.setAttribute = (k: string, v: string) => (e.attrs[k] = v);
    e.appendChild = (c: any) => (e.children.push(c), c);
    e.addEventListener = (t: string, f: any) => (e.listeners[t] = f);
    return e;
  };
  const list = el();
  const first = el();
  first.className = "item-class";
  list.querySelector = () => first;
  const box: any = {
    dataset,
    querySelector: (s: string) => (s === "ul" ? list : { innerHTML: "<i></i>" }),
  };
  const location = { href: "" };
  const fetched: string[] = [];
  const fetchStub = (url: string, init?: any) => {
    if (url.endsWith("versions.json")) return Promise.resolve({ ok: true, json: () => Promise.resolve(versions) });
    fetched.push(url);
    return Promise.resolve({ ok: existing.has(url) });
  };
  const script = markup.match(/<script>([\s\S]*?)<\/script>/)![1];
  const ctx = {
    document: { currentScript: { previousElementSibling: box }, createElement: el },
    fetch: fetchStub,
    window: { location },
  };
  vm.runInNewContext(script, ctx);
  await new Promise((r) => setTimeout(r, 20));
  const items = list.children.map((li: any) => li.children[0]);
  return {
    labels: items.map((a: any) => a.children.find((c: any) => c.textContent)?.textContent as string),
    hrefs: items.map((a: any) => a.href as string),
    click: async (i: number) => {
      items[i].listeners.click({ preventDefault() {} });
      await new Promise((r) => setTimeout(r, 20));
      return { to: location.href, fetched };
    },
  };
}

const versions = [
  { version: "main", title: "main", aliases: [], properties: { development: true } },
  { version: "1.1", title: "1.1", aliases: ["latest", "stable"] },
  { version: "1.0", title: "1.0" },
  { version: "0.9", title: "0.9", properties: { hidden: true } },
];

test("server side: current version, default-language fallback data", () => {
  const html = buildHome("1.1", "params:\n  versions:\n    development: main\n");
  expect(html).toContain('class="hextra-version-select');
  expect(html).not.toContain("fs-version-select");
  expect(html).toContain('data-docs-root="/docs/"');
  expect(html).toContain('data-current="1.1"');
  expect(html).toContain('data-dev="main"');
  expect(html).toContain("data-rest-default=");
});

test("lists versions: hidden skipped, aliases shown, development by name", async () => {
  const html = buildHome("1.1", "params:\n  versions:\n    development: main\n");
  const r = await runScript(html, versions, new Set());
  expect(r.labels).toEqual(["main", "1.1 (latest, stable)", "1.0"]);
  expect(r.hrefs[1]).toBe("/docs/1.1/");
});

test("an option falls back: page, default-language page, language home, version home", async () => {
  const html = buildHome("1.1");
  // On the home page rest is "" so only the version home remains.
  let r = await runScript(html, versions, new Set());
  expect((await r.click(2)).to).toBe("/docs/1.0/");

  // Simulate a deeper page in French by editing the data attributes.
  const fr = html
    .replace(/data-rest="[^"]*"/, 'data-rest="fr/guide/"')
    .replace(/data-rest-default="[^"]*"/, 'data-rest-default="guide/"')
    .replace(/data-lang-prefix="[^"]*"/, 'data-lang-prefix="fr/"');
  r = await runScript(fr, versions, new Set(["/docs/1.0/guide/"]));
  const out = await r.click(2);
  expect(out.fetched).toEqual(["/docs/1.0/fr/guide/", "/docs/1.0/guide/"]);
  expect(out.to).toBe("/docs/1.0/guide/");

  r = await runScript(fr, versions, new Set(["/docs/1.0/fr/"]));
  expect((await r.click(2)).to).toBe("/docs/1.0/fr/");

  r = await runScript(fr, versions, new Set());
  expect((await r.click(2)).to).toBe("/docs/1.0/");
});
