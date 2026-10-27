# Changelog (tannevaled fork)

Releases follow [semantic versioning](https://semver.org). Each states the upstream base it is built on.

## v0.21.0

Upstream base: imfing/hextra `main` at e58687e (unchanged).

- **page history generator in Go.** `cmd/pagehistory` (its own module, cobra) replaces `scripts/page-history.sh`. The JSON is byte-identical (golden test against the legacy script on a fixture repository with renames and several authors, and on this repository). It runs git with an explicit argument vector, literal pathspecs, parallel and bounded (`--max-files`, per-run timeout), refuses shallow clones and file names with control characters, `..`, or a leading `-`. Fuzz tests cover the parsing of git output, JSON escaping and path validation.
- Use: `go run github.com/tannevaled/hextra/cmd/pagehistory@v0.21.0 > data/pagehistory.json`. The Go module is tagged `cmd/pagehistory/v0.21.0` as Go requires for a nested module.
- `scripts/page-history.sh` is deprecated: it prints a note and runs the Go program (needs a Go toolchain). It will be removed in the next MINOR release.

## v0.20.0

- Language switch as a combo like the theme toggle, with configurable placement (#30).
- No stray `ZgotmplZ` attribute on the search dialog (#28).

## v0.19.0

- Any flag of the world by its code for `params.flag` (#29).
