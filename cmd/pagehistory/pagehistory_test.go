package main

import (
	"bytes"
	"encoding/json"
	"hash/fnv"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func gitT(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commit stages everything and commits as name at date.
func commit(t *testing.T, dir, name, date, msg string) {
	t.Helper()
	env := []string{
		"GIT_AUTHOR_NAME=" + name, "GIT_AUTHOR_EMAIL=x@example.org", "GIT_AUTHOR_DATE=" + date,
		"GIT_COMMITTER_NAME=" + name, "GIT_COMMITTER_EMAIL=x@example.org", "GIT_COMMITTER_DATE=" + date,
	}
	gitT(t, dir, env, "add", "-A")
	gitT(t, dir, env, "commit", "-q", "-m", msg)
}

// uniq is a page body unlike every other seed's, so that git's rename
// detection (which --follow uses) only links a file to its real ancestor.
func uniq(seed string) string {
	h := fnv.New64a()
	h.Write([]byte(seed))
	r := rand.New(rand.NewSource(int64(h.Sum64())))
	var b strings.Builder
	b.WriteString("---\ntitle: " + seed + "\n---\n\n")
	for i := 0; i < 40; i++ {
		b.WriteString(strconv.FormatUint(r.Uint64(), 36) + " " + strconv.FormatUint(r.Uint64(), 36) + "\n")
	}
	return b.String()
}

// fixture builds a repository with several authors, renames (also a rename
// followed by an edit by someone else), names that need escaping, a space and
// a quote in file names, a non-content file and a file outside content/.
func fixture(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	gitT(t, d, nil, "init", "-q", "-b", "main")
	gitT(t, d, nil, "config", "core.autocrlf", "false")
	write(t, d, "content/a.md", uniq("a"))
	commit(t, d, "Alice Liddell", "2026-01-02T03:04:05+01:00", "a")
	write(t, d, "content/b.markdown", uniq("b"))
	write(t, d, "content/img.png", "png")
	commit(t, d, "Bob", "2026-02-03T10:00:00-05:00", "b")
	if err := os.MkdirAll(filepath.Join(d, "content/sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	gitT(t, d, nil, "mv", "content/a.md", "content/sub/a-renamed.md")
	commit(t, d, "Carol", "2026-03-04T11:11:11+00:00", "rename")
	write(t, d, "content/sub/a-renamed.md", uniq("a")+"edited\n")
	commit(t, d, "Dave", "2026-04-05T12:00:00+09:00", "edit")
	write(t, d, "content/q\"uote and space.md", uniq("q"))
	commit(t, d, `Dan "D" O'Neil`, "2026-05-06T13:00:00+02:00", "quote")
	write(t, d, "content/back.org", uniq("back"))
	commit(t, d, `Eve\Back`, "2026-06-07T14:00:00+02:00", "backslash")
	write(t, d, "content/fr/zoe.html", uniq("zoe"))
	commit(t, d, "Zoë Müller", "2026-07-08T15:00:00+02:00", "unicode")
	write(t, d, "content/doc.adoc", uniq("adoc"))
	write(t, d, "other/outside.md", uniq("outside"))
	commit(t, d, "Frank", "2026-08-09T16:00:00+02:00", "adoc")
	return d
}

func goJSON(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	cmd := RootCmd()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(append([]string{"-C", root}, args...))
	err := cmd.Execute()
	return out.String(), err
}

func legacyJSON(t *testing.T, root string, args ...string) string {
	t.Helper()
	script, err := filepath.Abs(filepath.Join("testdata", "page-history.legacy.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", append([]string{script}, args...)...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("legacy script: %v", err)
	}
	return string(out)
}

// TestGoldenAgainstLegacyScript is the control: the Go program and the script
// it replaces print the same bytes on the same repository.
func TestGoldenAgainstLegacyScript(t *testing.T) {
	d := fixture(t)
	for _, args := range [][]string{nil, {"content"}, {"content/sub"}, {"other"}, {"nothing-here"}} {
		want := legacyJSON(t, d, args...)
		got, err := goJSON(t, d, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if got != want {
			t.Errorf("args %v differ\n--- legacy\n%s--- go\n%s", args, want, got)
		}
		if !json.Valid([]byte(got)) {
			t.Errorf("args %v: invalid JSON\n%s", args, got)
		}
	}
	// the content the fixture is meant to exercise is really there
	got, _ := goJSON(t, d)
	for _, must := range []string{
		`"content/sub/a-renamed.md": {"created": "2026-01-02T03:04:05+01:00", "author": "Alice Liddell"}`, // creation followed across the rename
		`"author": "Dan \"D\" O'Neil"`,
		`"author": "Eve\\Back"`,
		`"author": "Zoë Müller"`,
		`"content/q\"uote and space.md"`,
	} {
		if !strings.Contains(got, must) {
			t.Errorf("missing %s in\n%s", must, got)
		}
	}
	if strings.Contains(got, "img.png") || strings.Contains(got, "outside.md") {
		t.Errorf("non-content or out-of-dir file listed:\n%s", got)
	}
}

// TestPlantedDifferenceFails proves the comparison can see a difference.
func TestPlantedDifferenceFails(t *testing.T) {
	d := fixture(t)
	want := legacyJSON(t, d)
	other, err := goJSON(t, d, "content/sub")
	if err != nil {
		t.Fatal(err)
	}
	if other == want {
		t.Fatal("a different directory printed identical JSON: the golden comparison is blind")
	}
	got, _ := goJSON(t, d)
	if strings.Replace(got, "Alice Liddell", "Alice Liddel", 1) == want {
		t.Fatal("a one-letter change in an author went unseen")
	}
}

func TestShallowCloneRefused(t *testing.T) {
	d := fixture(t)
	c := t.TempDir()
	gitT(t, c, nil, "clone", "-q", "--depth", "1", "file://"+d, "shallow")
	_, err := goJSON(t, filepath.Join(c, "shallow"))
	if err == nil || !strings.Contains(err.Error(), "shallow") {
		t.Fatalf("shallow clone not refused: %v", err)
	}
}

func TestMaxFiles(t *testing.T) {
	d := fixture(t)
	if _, err := goJSON(t, d, "--max-files", "2"); err == nil {
		t.Fatal("max-files not enforced")
	}
	if _, err := goJSON(t, d, "--max-files", "100"); err != nil {
		t.Fatal(err)
	}
}

func TestDirValidation(t *testing.T) {
	d := fixture(t)
	for _, dir := range []string{"..", "../x", "/etc", "-p", "a/../../b", "a\nb"} {
		if _, err := goJSON(t, d, "--", dir); err == nil {
			t.Errorf("dir %q accepted", dir)
		}
	}
}

// A file whose name looks like a pathspec magic must be a file, not a pattern.
func TestLiteralPathspec(t *testing.T) {
	d := t.TempDir()
	gitT(t, d, nil, "init", "-q", "-b", "main")
	write(t, d, "content/real.md", uniq("real"))
	commit(t, d, "Ann", "2026-01-01T00:00:00+01:00", "real")
	write(t, d, "content/:(glob)*.md", uniq("magic"))
	commit(t, d, "Mallory", "2026-02-02T00:00:00+01:00", "magic")
	got, err := goJSON(t, d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"content/real.md": {"created": "2026-01-01T00:00:00+01:00", "author": "Ann"}`) ||
		!strings.Contains(got, `"content/:(glob)*.md": {"created": "2026-02-02T00:00:00+01:00", "author": "Mallory"}`) {
		t.Fatalf("pathspec magic was interpreted:\n%s", got)
	}
}

func TestControlCharInPathRefused(t *testing.T) {
	d := t.TempDir()
	gitT(t, d, nil, "init", "-q", "-b", "main")
	write(t, d, "content/new\nline.md", uniq("nl"))
	commit(t, d, "Ann", "2026-01-01T00:00:00+00:00", "x")
	if _, err := goJSON(t, d); err == nil {
		t.Fatal("a path with a newline was accepted")
	}
}

func TestBadFlags(t *testing.T) {
	d := fixture(t)
	if _, err := goJSON(t, d, "--jobs", "0"); err == nil {
		t.Fatal("jobs 0 accepted")
	}
	if _, err := goJSON(t, d, "a", "b"); err == nil {
		t.Fatal("two dirs accepted")
	}
}

func TestValidRelPath(t *testing.T) {
	ok := []string{"content", "content/a.md", "a b/c.md", "é/ü.md", "a..b/c", "content/q\"x.md"}
	bad := []string{"", "/abs", "\\abs", "-x", "a/../b", "..", "a\x00b", "a\tb", "a\x7fb", "a\xffb", strings.Repeat("a", 5000)}
	for _, p := range ok {
		if err := validRelPath(p); err != nil {
			t.Errorf("%q refused: %v", p, err)
		}
	}
	for _, p := range bad {
		if validRelPath(p) == nil {
			t.Errorf("%q accepted", p)
		}
	}
}

func TestParseLineAndLastLine(t *testing.T) {
	d, a, ok := parseLine("2026-01-02T03:04:05+01:00\tAda\tLovelace")
	if !ok || d != "2026-01-02T03:04:05+01:00" || a != "Ada\tLovelace" {
		t.Fatalf("%q %q %v", d, a, ok)
	}
	for _, bad := range []string{"", "no tab", "\tname"} {
		if _, _, ok := parseLine(bad); ok {
			t.Errorf("%q parsed", bad)
		}
	}
	if got := lastLine([]byte("a\tb\nc\td\n")); got != "c\td" {
		t.Errorf("lastLine = %q", got)
	}
	if got := lastLine(nil); got != "" {
		t.Errorf("lastLine(nil) = %q", got)
	}
}

func FuzzParseLine(f *testing.F) {
	for _, s := range []string{"", "2026-01-01T00:00:00+00:00\tAda", "\t", "a\tb\tc", "x"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		d, a, ok := parseLine(s)
		if ok && d+"\t"+a != s {
			t.Fatalf("parse lost bytes: %q -> %q %q", s, d, a)
		}
		_ = lastLine([]byte(s))
	})
}

// Whatever a repository puts in a name, the output stays valid JSON and
// decodes back to the name.
func FuzzEsc(f *testing.F) {
	for _, s := range []string{"", `a"b\c`, "tab\there", "nl\nx", "\x00\x1f\x7f", "Zoë", "\xff\xfe", "</script>"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var back string
		if err := json.Unmarshal([]byte(`"`+esc(s)+`"`), &back); err != nil {
			t.Fatalf("invalid JSON for %q: %v", s, err)
		}
		if want := strings.ToValidUTF8(s, "�"); back != want {
			t.Fatalf("round trip %q -> %q", want, back)
		}
	})
}

func FuzzValidRelPath(f *testing.F) {
	for _, s := range []string{"content", "../x", "/a", "a\nb", "-x", "é"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if validRelPath(s) != nil {
			return
		}
		if s == "" || !utf8.ValidString(s) || strings.HasPrefix(s, "/") || strings.HasPrefix(s, "-") {
			t.Fatalf("accepted %q", s)
		}
		for _, r := range s {
			if r < 0x20 || r == 0x7f {
				t.Fatalf("accepted control char in %q", s)
			}
		}
		for _, c := range strings.Split(s, "/") {
			if c == ".." {
				t.Fatalf("accepted .. in %q", s)
			}
		}
	})
}
