package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
)

// labelRE is a documentation version: MAJOR.MINOR. "main" is the other label,
// for the build made while no release is tagged.
var labelRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// basePathRE is where the site is served: /docs for a project named docs. Only
// characters that need no escaping in a URL path or a JavaScript string.
var basePathRE = regexp.MustCompile(`^/[A-Za-z0-9._~-]+(/[A-Za-z0-9._~-]+)*$`)

const mainLabel = "main"

type options struct {
	keepMain bool
	basePath string
}

// RootCmd builds the command; tests drive it with SetArgs/SetOut/SetErr.
func RootCmd() *cobra.Command {
	o := options{}
	cmd := &cobra.Command{
		Use:   "publishversion <version> <built site> <gh-pages checkout>",
		Short: "publish one documentation build into the gh-pages checkout, one directory per version",
		Long: `Publishes a built site as <gh-pages>/<version>/ and nothing else, then rewrites,
from the version directories present:

    versions.json   mike's format ({version, title, aliases, properties}), newest
                    release first, "main" last as the development version; the
                    theme's version selector and version banner read it;
    latest          a symbolic link to the newest release (to main when no
                    release exists);
    index.html      the root, redirecting to latest/;
    404.html        written once when a release is published, redirecting a link
                    into the retired /docs/main/ to the same page of latest/.

<version> is MAJOR.MINOR (0.12), or "main" while no release is tagged.

Publishing a release RETIRES a leftover main/ build: nothing rebuilds it once
releases are published, so it would go on showing out-of-date pages as the
development version. --keep-main keeps it.

Nothing is written outside the gh-pages checkout. The built site and the
checkout must not contain one another, and a symbolic link in the built site
is refused rather than followed.

    go run github.com/tannevaled/hextra/cmd/publishversion@<tag> 0.12 site gh-pages`,
		Args:          cobra.ExactArgs(3),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.OutOrStdout(), o, args[0], args[1], args[2])
		},
	}
	cmd.Flags().BoolVar(&o.keepMain, "keep-main", false, "keep a main/ build when a release is published")
	cmd.Flags().StringVar(&o.basePath, "base-path", "/docs", "where the site is served, for the 404 page's redirect")
	return cmd
}

func run(out io.Writer, o options, version, site, pages string) error {
	if version != mainLabel && !labelRE.MatchString(version) {
		return fmt.Errorf("version %q is neither MAJOR.MINOR (0.12) nor main", version)
	}
	if !basePathRE.MatchString(o.basePath) {
		return fmt.Errorf("--base-path %q is not a plain absolute path such as /docs", o.basePath)
	}
	if fi, err := os.Stat(filepath.Join(site, "index.html")); err != nil || fi.IsDir() {
		return fmt.Errorf("%s/index.html is missing: build the site first", site)
	}
	if fi, err := os.Stat(pages); err != nil || !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", pages)
	}
	if err := checkApart(site, pages); err != nil {
		return err
	}

	// Stage the new build beside its destination, and swap it in only once it
	// is fully copied: a failure leaves the published version as it was.
	stage := filepath.Join(pages, ".publishversion-"+version+".tmp")
	if err := os.RemoveAll(stage); err != nil {
		return err
	}
	if err := copyTree(site, stage); err != nil {
		_ = os.RemoveAll(stage)
		return fmt.Errorf("copying the built site: %w", err)
	}
	dest := filepath.Join(pages, version)
	// A link (an alias) goes, not what it pointed to.
	if err := os.RemoveAll(dest); err != nil {
		_ = os.RemoveAll(stage)
		return err
	}
	if err := os.Rename(stage, dest); err != nil {
		_ = os.RemoveAll(stage)
		return err
	}

	if version != mainLabel && !o.keepMain {
		if fi, err := os.Lstat(filepath.Join(pages, mainLabel)); err == nil && fi.IsDir() {
			if err := os.RemoveAll(filepath.Join(pages, mainLabel)); err != nil {
				return err
			}
			fmt.Fprintln(out, "retired the superseded main build")
		}
	}
	if version != mainLabel {
		if err := writeOnce(filepath.Join(pages, "404.html"), notFoundPage(o.basePath)); err != nil {
			return err
		}
	}

	versions, latest, err := reconcile(pages)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "published %s; latest is %s; versions: %s\n", version, latest, strings.Join(versions, ", "))
	return nil
}

// checkApart refuses a built site that contains the checkout or is contained
// by it: publishing would copy a tree into itself.
func checkApart(site, pages string) error {
	s, err := filepath.EvalSymlinks(site)
	if err != nil {
		return err
	}
	p, err := filepath.EvalSymlinks(pages)
	if err != nil {
		return err
	}
	s, _ = filepath.Abs(s)
	p, _ = filepath.Abs(p)
	if within(s, p) || within(p, s) {
		return fmt.Errorf("the built site %s and the checkout %s must not contain one another", site, pages)
	}
	return nil
}

// within reports whether a is b or lies under b.
func within(a, b string) bool {
	rel, err := filepath.Rel(b, a)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// copyTree copies regular files and directories. A symbolic link or any other
// kind of file is an error: following one could publish a file from outside
// the built site, and a build has no business containing one.
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type().IsRegular():
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, b, 0o644)
		default:
			return fmt.Errorf("%s is not a regular file or a directory (a symbolic link?): refused", path)
		}
	})
}

// writeOnce writes a file that does not exist yet and leaves an existing one
// alone: it may have been edited by hand.
func writeOnce(path, content string) error {
	if _, err := os.Lstat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return writeFile(path, content)
}

// writeFile writes through a temporary file, so a reader never sees half of it.
func writeFile(path, content string) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type entry struct {
	Version    string          `json:"version"`
	Title      string          `json:"title"`
	Aliases    []string        `json:"aliases"`
	Properties map[string]bool `json:"properties,omitempty"`
}

// published lists the version directories present, newest release first
// (0.10 before 0.9: compared as numbers), then main. Links are not versions.
func published(pages string) ([]string, error) {
	entries, err := os.ReadDir(pages)
	if err != nil {
		return nil, err
	}
	var releases []string
	hasMain := false
	for _, e := range entries {
		if e.Type()&fs.ModeSymlink != 0 || !e.IsDir() {
			continue
		}
		switch {
		case e.Name() == mainLabel:
			hasMain = true
		case labelRE.MatchString(e.Name()):
			releases = append(releases, e.Name())
		}
	}
	sort.Slice(releases, func(i, j int) bool {
		ai, aj := parts(releases[i]), parts(releases[j])
		if ai[0] != aj[0] {
			return ai[0] > aj[0]
		}
		return ai[1] > aj[1]
	})
	if hasMain {
		releases = append(releases, mainLabel)
	}
	return releases, nil
}

func parts(label string) [2]int {
	maj, min, _ := strings.Cut(label, ".")
	a, _ := strconv.Atoi(maj)
	b, _ := strconv.Atoi(min)
	return [2]int{a, b}
}

// reconcile rewrites versions.json, the latest link, the root redirect and
// .nojekyll from the version directories present.
func reconcile(pages string) (versions []string, latest string, err error) {
	versions, err = published(pages)
	if err != nil {
		return nil, "", err
	}
	if len(versions) == 0 {
		return nil, "", errors.New("no version directory is present")
	}
	latest = versions[0]

	list := make([]entry, 0, len(versions))
	for _, v := range versions {
		e := entry{Version: v, Title: v, Aliases: []string{}}
		if v == latest {
			e.Aliases = []string{"latest"}
		}
		if v == mainLabel {
			e.Properties = map[string]bool{"development": true}
		}
		list = append(list, e)
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return nil, "", err
	}
	if err := writeFile(filepath.Join(pages, "versions.json"), string(b)+"\n"); err != nil {
		return nil, "", err
	}

	link := filepath.Join(pages, "latest")
	if err := os.RemoveAll(link); err != nil {
		return nil, "", err
	}
	if err := os.Symlink(latest, link); err != nil {
		return nil, "", err
	}
	if err := writeFile(filepath.Join(pages, "index.html"), rootRedirect); err != nil {
		return nil, "", err
	}
	if _, err := os.Lstat(filepath.Join(pages, ".nojekyll")); errors.Is(err, fs.ErrNotExist) {
		if err := os.WriteFile(filepath.Join(pages, ".nojekyll"), nil, 0o644); err != nil {
			return nil, "", err
		}
	}
	return versions, latest, nil
}

const rootRedirect = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Redirecting</title>
  <noscript>
    <meta http-equiv="refresh" content="1; url=latest/" />
  </noscript>
  <script>
    window.location.replace(
      "latest/" + window.location.search + window.location.hash
    );
  </script>
</head>
<body>
  Redirecting to <a href="latest/">latest/</a>...
</body>
</html>
`

var notFoundTmpl = template.Must(template.New("404").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Page not found</title>
  <script>
    // The build that was published at {{.Base}}/main/ is retired: a link into it
    // opens the same page of the latest version.
    (function () {
      var base = "{{.Base}}", gone = base + "/main", p = window.location.pathname;
      if (p === gone || p.indexOf(gone + "/") === 0) {
        window.location.replace(base + "/latest" + (p.slice(gone.length) || "/") + window.location.search + window.location.hash);
      }
    })();
  </script>
</head>
<body style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem;">
  <h1>Page not found</h1>
  <p>This page does not exist in the documentation. Try <a href="{{.Base}}/latest/">the latest version</a>.</p>
</body>
</html>
`))

// notFoundPage renders the 404 page for a site served at basePath, which the
// caller has validated as a plain path.
func notFoundPage(basePath string) string {
	var b strings.Builder
	if err := notFoundTmpl.Execute(&b, struct{ Base string }{basePath}); err != nil {
		panic(err) // a template and a validated string: cannot fail
	}
	return b.String()
}
