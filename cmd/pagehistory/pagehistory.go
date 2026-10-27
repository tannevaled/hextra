package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// Bounds. A repository's file names and history are untrusted input.
const (
	defaultMaxFiles = 50000
	maxListBytes    = 64 << 20 // git ls-files output
	maxLogBytes     = 16 << 20 // git log output for ONE file
)

// contentExts are the page extensions the theme reads.
var contentExts = []string{".md", ".markdown", ".html", ".adoc", ".org"}

type options struct {
	root     string
	maxFiles int
	jobs     int
	timeout  time.Duration
}

// RootCmd builds the command; tests drive it with SetArgs/SetOut/SetErr.
func RootCmd() *cobra.Command {
	o := options{}
	cmd := &cobra.Command{
		Use:   "pagehistory [dir]",
		Short: "when, and by whom, every content page was created, from git, as JSON",
		Long: `Hugo's GitInfo knows only a page's LAST commit. pagehistory writes its first
one, followed across renames, with the author's NAME only (never the e-mail),
for the theme's page history line (params.page.history).

Run it from the site's root, before hugo:

    go run github.com/tannevaled/hextra/cmd/pagehistory@<tag> > data/pagehistory.json

dir (default "content") is a path relative to the root. It needs the full
history: a shallow clone is refused.`,
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "content"
			if len(args) == 1 {
				dir = args[0]
			}
			if o.jobs < 1 {
				return errors.New("--jobs must be at least 1")
			}
			if o.maxFiles < 1 {
				return errors.New("--max-files must be at least 1")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), o.timeout)
			defer cancel()
			return run(ctx, cmd.OutOrStdout(), o, dir)
		},
	}
	jobs := runtime.NumCPU()
	if jobs > 8 {
		jobs = 8
	}
	cmd.Flags().StringVarP(&o.root, "root", "C", ".", "the site's root, where git runs")
	cmd.Flags().IntVar(&o.maxFiles, "max-files", defaultMaxFiles, "refuse more content files than this")
	cmd.Flags().IntVarP(&o.jobs, "jobs", "j", jobs, "git log processes run at once")
	cmd.Flags().DurationVar(&o.timeout, "timeout", 10*time.Minute, "give up after this long")
	return cmd
}

// validRelPath refuses what could escape the repository or break the output:
// empty, absolute, a ".." component, control characters, invalid UTF-8, and a
// leading "-" (an option, were it ever passed without "--").
func validRelPath(p string) error {
	switch {
	case p == "":
		return errors.New("empty path")
	case strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\"):
		return errors.New("absolute path")
	case strings.HasPrefix(p, "-"):
		return errors.New("path starts with '-'")
	case !utf8.ValidString(p):
		return errors.New("path is not valid UTF-8")
	case len(p) > 4096:
		return errors.New("path too long")
	}
	for _, r := range p {
		if r < 0x20 || r == 0x7f {
			return errors.New("control character in path")
		}
	}
	for _, c := range strings.FieldsFunc(p, func(r rune) bool { return r == '/' || r == '\\' }) {
		if c == ".." {
			return errors.New("'..' component in path")
		}
	}
	return nil
}

func isContent(p string) bool {
	for _, e := range contentExts {
		if strings.HasSuffix(p, e) {
			return true
		}
	}
	return false
}

// parseLine splits the last line of "git log --format=%aI%x09%an": the author
// date, a tab, the author name. The name may itself contain tabs.
func parseLine(line string) (date, author string, ok bool) {
	i := strings.IndexByte(line, '\t')
	if i <= 0 {
		return "", "", false
	}
	return line[:i], line[i+1:], true
}

// lastLine is "tail -n 1": the last line of git's output, without its newline.
func lastLine(b []byte) string {
	b = bytes.TrimSuffix(b, []byte("\n"))
	if i := bytes.LastIndexByte(b, '\n'); i >= 0 {
		b = b[i+1:]
	}
	return string(b)
}

// esc escapes a JSON string body. Backslash and quote exactly as the shell
// script did; control characters (which the script left raw, making invalid
// JSON) as \u00XX; invalid UTF-8 as U+FFFD.
func esc(s string) string {
	s = strings.ToValidUTF8(s, "�")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\\':
			b.WriteString(`\\`)
		case r == '"':
			b.WriteString(`\"`)
		case r < 0x20 || r == 0x7f:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

type limitedBuffer struct {
	buf bytes.Buffer
	max int
}

func (l *limitedBuffer) Write(p []byte) (int, error) {
	if l.buf.Len()+len(p) > l.max {
		return 0, fmt.Errorf("git output exceeds %d bytes", l.max)
	}
	return l.buf.Write(p)
}

// git runs git with an explicit argument vector, never a shell. Pathspecs are
// literal: a file called ":(glob)*.md" is a file, not a pattern.
func git(ctx context.Context, root string, max int, args ...string) ([]byte, error) {
	full := append([]string{"--no-pager", "--literal-pathspecs", "-C", root}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)
	out := &limitedBuffer{max: max}
	var stderr bytes.Buffer
	cmd.Stdout = out
	cmd.Stderr = &stderr
	cmd.Stdin = nil
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out.buf.Bytes(), nil
}

type result struct {
	path, date, author string
	ok                 bool
	err                error
}

func run(ctx context.Context, w io.Writer, o options, dir string) error {
	if err := validRelPath(dir); err != nil {
		return fmt.Errorf("dir %q: %w", dir, err)
	}
	sh, err := git(ctx, o.root, 1024, "rev-parse", "--is-shallow-repository")
	if err == nil && strings.TrimSpace(string(sh)) == "true" {
		return errors.New("shallow clone, creation dates would be wrong (fetch the full history, e.g. GIT_DEPTH: 0)")
	}
	list, err := git(ctx, o.root, maxListBytes, "ls-files", "-z", "--", dir)
	if err != nil {
		return err
	}
	var files []string
	for _, f := range strings.Split(string(list), "\x00") {
		if f == "" || !isContent(f) {
			continue
		}
		if err := validRelPath(f); err != nil {
			return fmt.Errorf("refusing %q: %w", f, err)
		}
		files = append(files, f)
		if len(files) > o.maxFiles {
			return fmt.Errorf("more than %d content files (raise --max-files)", o.maxFiles)
		}
	}

	results := make([]result, len(files))
	idx := make(chan int)
	var wg sync.WaitGroup
	for j := 0; j < o.jobs; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range idx {
				results[i] = history(ctx, o.root, files[i])
			}
		}()
	}
	for i := range files {
		idx <- i
	}
	close(idx)
	wg.Wait()

	bw := bufio.NewWriter(w)
	bw.WriteString("{")
	sep := ""
	for _, r := range results {
		if r.err != nil {
			return r.err
		}
		if !r.ok {
			continue
		}
		fmt.Fprintf(bw, "%s\n  \"%s\": {\"created\": \"%s\", \"author\": \"%s\"}", sep, esc(r.path), esc(r.date), esc(r.author))
		sep = ","
	}
	bw.WriteString("\n}\n")
	return bw.Flush()
}

func history(ctx context.Context, root, f string) result {
	out, err := git(ctx, root, maxLogBytes, "log", "--follow", "--diff-filter=A", "--format=%aI%x09%an", "--", f)
	if err != nil {
		return result{err: err}
	}
	line := lastLine(out)
	if line == "" {
		return result{}
	}
	date, author, ok := parseLine(line)
	if !ok {
		return result{err: fmt.Errorf("unexpected git log line for %q", f)}
	}
	return result{path: f, date: date, author: author, ok: true}
}
