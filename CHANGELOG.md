# Changelog (tannevaled fork)

Releases follow [semantic versioning](https://semver.org). Each states the upstream base it is built on.

## v0.24.1 (unreleased)

- **version banner: no link to itself.** When the only published build is the development one (it holds both the alias `latest` and `properties.development`, as `pagespublish` writes before any release), its notice no longer offers "Go to the latest version", which was the same page. Tested by a new case of `tests/version-banner.spec.ts` that fails on v0.24.0.

- **`cmd/publishversion`.** A Go program (its own module, cobra, tagged `cmd/publishversion/v0.24.1`) that publishes a built site into a gh-pages checkout, one directory per version: `go run github.com/tannevaled/hextra/cmd/publishversion@v0.24.1 0.12 site gh-pages`. It replaces the copies of `publish-version.sh` the documentation workflows carried and rewrites `versions.json` (mike's format, newest release first, `main` last as the development version), the `latest` link, the root redirect and `.nojekyll`. Publishing a release **retires a leftover `main/` build** (nothing rebuilds it once releases are published; `--keep-main` keeps it) and writes a `404.html` once that sends a link into `/docs/main/…` to the same page of `latest` (`--base-path`, default `/docs`). It refuses a version that is not `MAJOR.MINOR` or `main`, a built site and a checkout that contain one another, and a symbolic link in the build; the new build is staged and swapped in, so a failure leaves the published version as it was; a second run of the same build changes nothing. Tests include the leftover-`main/` case planted, the idempotence, the refusals and a fuzz of the version argument; removing the retirement or the containment check fails them.

## v0.24.0

- **version banner component.** `components/version-banner.html` (called from `custom/content-begin.html`) replaces the per-site copies of the "outdated version" banner. Decided in the browser from `<docs root>/versions.json`: the version with the alias `latest` shows nothing, the development version shows a "development version" notice (the copies showed "outdated" there), any other version shows "outdated" with a link to the docs root; no `versions.json`, no banner. Tested by `tests/version-banner.spec.ts` (including a `latest` alias that moves after the page was built, with no rebuild; a mutated comparison fails both tests).

- **search: no characters lost when typing as the dialog opens.** Opening the dialog selected the field's content on the next frame so that typing replaces the previous query; a fast typist's first characters were already in the field, got selected, and were replaced by the next keystroke (`toolkit` became `oolkit`, so a documentation site answered "no results" for a word it has). The content is now selected only if nothing was typed since the dialog opened.
- **search: loading state.** A query typed before the index has loaded is run when it is ready (it already was), and the status line says "Loading the search index…" meanwhile (`searchLoading`, `window.hextraSearch.ready`) instead of showing nothing. Tested by `tests/search-typing.spec.ts` (with the search data delayed 1.5 s; fails on the previous code).

## v0.23.0

- **CJK search by pairs of characters.** A run of Chinese, Japanese or Korean characters is indexed (and queried) as overlapping pairs instead of one token per character, so a query no longer matches every page that contains its characters in any order. Measured on the go-widgets documentation (ja, ko, zh-cn) in Chrome, before / after, result links: `アクセシビリティ` 110 / 61, the page about it ranked first instead of an unrelated one; its reversed characters 108 / 4; `성근접` (`접근성` reversed) 58 / 0; `碍障无` (`无障碍` reversed) 52 / 0; `툴킷이름없음` 106 / 47. Latin queries return the same results (`accessibility` 14, `toolkit` 58, `painter` 26). A single character still finds the pages it begins a pair in. `params.search.flexsearch.cjk: char` restores one token per character. Tested by `tests/search-cjk.spec.ts` (the pair test fails on the previous code).

## v0.22.0

- `params.languageSwitch.flags`, to draw no flag (#33).

## v0.21.0

Upstream base: imfing/hextra `main` at e58687e (unchanged).

- **page history generator in Go.** `cmd/pagehistory` (its own module, cobra) replaces `scripts/page-history.sh`. The JSON is byte-identical (golden test against the legacy script on a fixture repository with renames and several authors, and on this repository). It runs git with an explicit argument vector, literal pathspecs, parallel and bounded (`--max-files`, per-run timeout), refuses shallow clones and file names with control characters, `..`, or a leading `-`. Fuzz tests cover the parsing of git output, JSON escaping and path validation.
- Use: `go run github.com/tannevaled/hextra/cmd/pagehistory@v0.21.0 > data/pagehistory.json`. The Go module is tagged `cmd/pagehistory/v0.21.0` as Go requires for a nested module.
- `scripts/page-history.sh` is deprecated: it prints a note and runs the Go program (needs a Go toolchain). It will be removed in the next MINOR release.

- **version selector parity.** `components/version-select.html` gains what the sites' local copies (`fs-version-select`) had: versions with `properties.hidden` are not listed, aliases are shown after the title (`1.1 (latest)`), the development version may be recognised by name (`params.versions.development`), and an option falls back page, same page in the default language, home in the current language, version home, so it never ends on a 404. Sites drop their `layouts/_partials/custom/version-select.html` and use `partial: components/version-select.html`. Tested by `tests/version-select.spec.ts` (fails on the previous component).

## v0.20.0

- Language switch as a combo like the theme toggle, with configurable placement (#30).
- No stray `ZgotmplZ` attribute on the search dialog (#28).

## v0.19.0

- Any flag of the world by its code for `params.flag` (#29).
