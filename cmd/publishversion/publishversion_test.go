package main

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// write creates a file under dir, with its parents.
func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(dir, rel string) bool {
	_, err := os.Lstat(filepath.Join(dir, rel))
	return err == nil
}

// site builds a minimal built site.
func site(t *testing.T, marker string) string {
	t.Helper()
	d := t.TempDir()
	write(t, d, "index.html", "<html>"+marker+"</html>")
	write(t, d, "commands/index.html", "commands "+marker)
	return d
}

// publish runs the command and returns what it printed and its error.
func publish(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := RootCmd()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

type versionEntry struct {
	Version    string          `json:"version"`
	Title      string          `json:"title"`
	Aliases    []string        `json:"aliases"`
	Properties map[string]bool `json:"properties"`
}

func versionsOf(t *testing.T, pages string) []versionEntry {
	t.Helper()
	var v []versionEntry
	if err := json.Unmarshal([]byte(read(t, pages, "versions.json")), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

// snapshot lists every path under dir with its content (or link target), to
// compare two states of a checkout.
func snapshot(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			l, _ := os.Readlink(p)
			out = append(out, rel+" -> "+l)
		case d.IsDir():
			out = append(out, rel+"/")
		default:
			b, _ := os.ReadFile(p)
			out = append(out, rel+": "+string(b))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)
	return out
}

func TestFirstPublishOfMain(t *testing.T) {
	pages := t.TempDir()
	out, err := publish(t, "main", site(t, "m1"), pages)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "published main; latest is main; versions: main") {
		t.Errorf("output %q", out)
	}
	if read(t, pages, "main/commands/index.html") != "commands m1" {
		t.Error("the build was not copied")
	}
	v := versionsOf(t, pages)
	if len(v) != 1 || v[0].Version != "main" || len(v[0].Aliases) != 1 || v[0].Aliases[0] != "latest" || !v[0].Properties["development"] {
		t.Errorf("versions.json = %+v", v)
	}
	if l, _ := os.Readlink(filepath.Join(pages, "latest")); l != "main" {
		t.Errorf("latest -> %q", l)
	}
	if !strings.Contains(read(t, pages, "index.html"), "latest/") || !exists(pages, ".nojekyll") {
		t.Error("root redirect or .nojekyll missing")
	}
	if exists(pages, "404.html") {
		t.Error("a main publish must not write the 404 page")
	}
}

// The planted positive: the situation the tool exists for. A gh-pages checkout
// with a leftover main/ and older releases, a release published over it.
func TestReleaseRetiresMainAndWritesTheNotFoundPageOnce(t *testing.T) {
	pages := t.TempDir()
	write(t, pages, "main/index.html", "stale development build")
	write(t, pages, "0.10/index.html", "ten")
	write(t, pages, "0.9/index.html", "nine")
	write(t, pages, "0.8/index.html", "eight")
	write(t, pages, "unrelated/index.html", "not a version")
	if err := os.Symlink("0.10", filepath.Join(pages, "latest")); err != nil {
		t.Fatal(err)
	}

	out, err := publish(t, "0.12", site(t, "r1"), pages)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "retired the superseded main build") {
		t.Errorf("no retirement reported: %q", out)
	}
	if exists(pages, "main") {
		t.Error("the leftover main/ is still there")
	}
	var got []string
	for _, e := range versionsOf(t, pages) {
		got = append(got, e.Version)
		if e.Version == "main" {
			t.Error("versions.json still lists main")
		}
	}
	if strings.Join(got, ",") != "0.12,0.10,0.9,0.8" {
		t.Errorf("versions = %v (0.10 must come before 0.9)", got)
	}
	if v := versionsOf(t, pages); v[0].Aliases[0] != "latest" || len(v[1].Aliases) != 0 {
		t.Errorf("aliases = %+v", v)
	}
	if l, _ := os.Readlink(filepath.Join(pages, "latest")); l != "0.12" {
		t.Errorf("latest -> %q", l)
	}
	if !exists(pages, "0.10/index.html") || !exists(pages, "unrelated/index.html") {
		t.Error("another version or an unrelated directory was touched")
	}

	nf := read(t, pages, "404.html")
	if !strings.Contains(nf, `"/docs"`) || !strings.Contains(nf, "/latest") {
		t.Errorf("404 page has no redirect: %s", nf)
	}

	// Written ONCE: a hand edit survives the next publish.
	write(t, pages, "404.html", "edited by hand")
	if _, err := publish(t, "0.12", site(t, "r2"), pages); err != nil {
		t.Fatal(err)
	}
	if read(t, pages, "404.html") != "edited by hand" {
		t.Error("the 404 page was rewritten")
	}
	if read(t, pages, "0.12/commands/index.html") != "commands r2" {
		t.Error("the version was not replaced by the second build")
	}
}

func TestKeepMainAndAMainLink(t *testing.T) {
	pages := t.TempDir()
	write(t, pages, "main/index.html", "kept")
	if _, err := publish(t, "--keep-main", "0.1", site(t, "x"), pages); err != nil {
		t.Fatal(err)
	}
	if !exists(pages, "main/index.html") {
		t.Error("--keep-main removed main")
	}
	if v := versionsOf(t, pages); v[len(v)-1].Version != "main" || v[0].Version != "0.1" {
		t.Errorf("main must come last: %+v", v)
	}

	// A link called main is an alias, not a build: it is neither removed nor listed.
	pages2 := t.TempDir()
	write(t, pages2, "0.1/index.html", "one")
	if err := os.Symlink("0.1", filepath.Join(pages2, "main")); err != nil {
		t.Fatal(err)
	}
	if _, err := publish(t, "0.2", site(t, "y"), pages2); err != nil {
		t.Fatal(err)
	}
	if !exists(pages2, "main") {
		t.Error("a link named main was removed")
	}
	for _, e := range versionsOf(t, pages2) {
		if e.Version == "main" {
			t.Error("a link named main was listed as a version")
		}
	}
}

func TestIdempotent(t *testing.T) {
	pages := t.TempDir()
	write(t, pages, "main/index.html", "stale")
	write(t, pages, "0.3/index.html", "three")
	s := site(t, "same")
	if _, err := publish(t, "0.4", s, pages); err != nil {
		t.Fatal(err)
	}
	first := snapshot(t, pages)
	if _, err := publish(t, "0.4", s, pages); err != nil {
		t.Fatal(err)
	}
	second := snapshot(t, pages)
	if strings.Join(first, "\n") != strings.Join(second, "\n") {
		t.Errorf("a second publish of the same build changed the checkout:\n%v\n%v", first, second)
	}
	for _, p := range second {
		if strings.Contains(p, ".publishversion-") {
			t.Errorf("a staging directory was left behind: %s", p)
		}
	}
}

func TestAVersionThatIsALinkIsReplacedWithoutTouchingItsTarget(t *testing.T) {
	pages := t.TempDir()
	write(t, pages, "0.1/index.html", "target")
	if err := os.Symlink("0.1", filepath.Join(pages, "0.2")); err != nil {
		t.Fatal(err)
	}
	if _, err := publish(t, "0.2", site(t, "n"), pages); err != nil {
		t.Fatal(err)
	}
	if read(t, pages, "0.1/index.html") != "target" {
		t.Error("the link's target was overwritten")
	}
	if fi, _ := os.Lstat(filepath.Join(pages, "0.2")); fi.Mode()&fs.ModeSymlink != 0 {
		t.Error("0.2 is still a link")
	}
}

func TestRefusals(t *testing.T) {
	good := site(t, "g")
	pages := t.TempDir()

	for _, v := range []string{"", "v0.12", "0.12.1", "1", "../x", "0.1/../../x", "latest", "0.01", "-1.2", "main/"} {
		if _, err := publish(t, v, good, pages); err == nil {
			t.Errorf("version %q accepted", v)
		}
	}
	if _, err := publish(t, "0.1", t.TempDir(), pages); err == nil {
		t.Error("a site without index.html accepted")
	}
	if _, err := publish(t, "0.1", good, filepath.Join(pages, "nope")); err == nil {
		t.Error("a missing checkout accepted")
	}
	if _, err := publish(t, "--base-path", "/docs\";alert(1);//", "0.1", good, pages); err == nil {
		t.Error("a base path that is not a plain path was accepted")
	}

	// A symbolic link in the build is refused, and nothing is published.
	linked := site(t, "l")
	secret := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(secret, []byte("outside the build"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(linked, "leak.txt")); err != nil {
		t.Fatal(err)
	}
	p2 := t.TempDir()
	write(t, p2, "0.1/index.html", "old")
	if _, err := publish(t, "0.2", linked, p2); err == nil {
		t.Error("a build holding a symbolic link was accepted")
	}
	if exists(p2, "0.2") || exists(p2, "leak.txt") || read(t, p2, "0.1/index.html") != "old" {
		t.Error("a refused publish changed the checkout")
	}
	if entries, _ := os.ReadDir(p2); len(entries) != 1 {
		t.Errorf("a refused publish left files behind: %v", entries)
	}

	// The site inside the checkout, and the checkout inside the site.
	inside := t.TempDir()
	write(t, inside, "built/index.html", "x")
	if _, err := publish(t, "0.1", filepath.Join(inside, "built"), inside); err == nil {
		t.Error("a site inside the checkout accepted")
	}
	outer := site(t, "o")
	if err := os.MkdirAll(filepath.Join(outer, "pages"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := publish(t, "0.1", outer, filepath.Join(outer, "pages")); err == nil {
		t.Error("a checkout inside the site accepted")
	}
}

func TestNothingIsWrittenOutsideTheCheckout(t *testing.T) {
	root := t.TempDir()
	pages := filepath.Join(root, "gh-pages")
	built := filepath.Join(root, "site")
	for _, d := range []string{pages, built} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write(t, built, "index.html", "x")
	before := snapshot(t, built)
	if _, err := publish(t, "0.5", built, pages); err != nil {
		t.Fatal(err)
	}
	if strings.Join(before, "\n") != strings.Join(snapshot(t, built), "\n") {
		t.Error("the built site was modified")
	}
	entries, _ := os.ReadDir(root)
	if len(entries) != 2 {
		t.Errorf("files appeared beside the checkout: %v", entries)
	}
}

// FuzzVersionArgument: whatever the label is, a publish that succeeds writes
// only inside the checkout, and a label that is neither main nor MAJOR.MINOR
// is refused before anything is written.
func FuzzVersionArgument(f *testing.F) {
	for _, s := range []string{"main", "0.12", "../x", "0.1/../../y", "a/b", "", ".", "..", "0.1\x00", "1.2.3"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, version string) {
		root := t.TempDir()
		pages := filepath.Join(root, "gh-pages")
		built := filepath.Join(root, "site")
		for _, d := range []string{pages, built} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(filepath.Join(built, "index.html"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := publish(t, version, built, pages)
		valid := version == "main" || labelRE.MatchString(version)
		if valid != (err == nil) {
			t.Fatalf("version %q: valid=%v err=%v", version, valid, err)
		}
		entries, _ := os.ReadDir(root)
		if len(entries) != 2 {
			t.Fatalf("version %q wrote outside the checkout: %v", version, entries)
		}
	})
}

func TestOrderIsNumeric(t *testing.T) {
	pages := t.TempDir()
	for _, v := range []string{"0.9", "0.10", "0.2", "1.0", "0.100"} {
		write(t, pages, v+"/index.html", v)
	}
	got, _, err := reconcile(pages)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "1.0,0.100,0.10,0.9,0.2" {
		t.Errorf("order = %v", got)
	}
}
