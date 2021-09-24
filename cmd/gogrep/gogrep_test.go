package main_test

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func content() []string {
	return []string{
		"grand theft wumps",
		"replublics of haskell",
		"a sunset is a sunset because it's crimson, beautiful, and I want it to be crimson",
		"hold fast that which is good",
		"domains of interest to people",
		"snowflake",
		"strict or lazy",
		"ehekatl of luck",
		"crime of using a side effect",
		"you must not request the world to stagnate",
	}
}

// TestMain performs end to end test.
func TestMain(t *testing.T) {
	g, err := newGrepper()
	fatalOnError(t, err)
	defer g.close()
	// prepare grep targets
	target := strings.Join(content(), "\n")
	fatalOnError(t, g.createFile("testmain0", target))
	fatalOnError(t, g.copyFile("testmain1", "testmain0"))

	test := func(t *testing.T, args, want []string) {
		cmd := exec.Command(g.command, args...)
		stdout, err := cmd.StdoutPipe()
		fatalOnError(t, err)
		fatalOnError(t, cmd.Start())
		gotBytes, err := io.ReadAll(stdout)
		fatalOnError(t, err)
		fatalOnError(t, cmd.Wait())
		got := strings.Split(strings.TrimSuffix(string(gotBytes), "\n"), "\n")
		assert.Equal(t, len(want), len(got))
		sort.Strings(want)
		sort.Strings(got)
		for i, w := range want {
			g := got[i]
			assert.Equal(t, w, g)
		}
	}

	t.Run("files", func(t *testing.T) {
		wantContent := []string{
			"grand theft wumps",
			"snowflake",
		}
		filenames := []string{
			g.filePath("testmain0"),
			g.filePath("testmain1"),
		}
		want := []string{}
		for _, c := range wantContent {
			for _, p := range filenames {
				want = append(want, fmt.Sprintf("%s:%s", p, c))
			}
		}
		args := []string{`snowflake|wumps`}
		args = append(args, filenames...)
		test(t, args, want)
	})

	t.Run("file", func(t *testing.T) {
		want := []string{
			"grand theft wumps",
			"snowflake",
		}
		args := []string{
			`snowflake|wumps`,
			g.filePath("testmain0"),
		}
		test(t, args, want)
	})

	t.Run("stdin", func(t *testing.T) {
		want := []string{
			"grand theft wumps",
			"snowflake",
		}

		cmd := exec.Command(g.command, `snowflake|wumps`)
		stdin, err := cmd.StdinPipe()
		fatalOnError(t, err)
		stdout, err := cmd.StdoutPipe()
		fatalOnError(t, err)
		fatalOnError(t, cmd.Start())
		go func() {
			defer stdin.Close()
			io.WriteString(stdin, target)
		}()
		gotBytes, err := io.ReadAll(stdout)
		fatalOnError(t, err)
		fatalOnError(t, cmd.Wait())
		got := strings.Split(strings.TrimSuffix(string(gotBytes), "\n"), "\n")
		assert.Equal(t, len(want), len(got))
		sort.Strings(want)
		sort.Strings(got)
		for i, w := range want {
			g := got[i]
			assert.Equal(t, w, g)
		}
	})
}

type grepper struct {
	workDir string // temporary directory
	command string // gogrep binary path
}

// newGrepper creates a temporary directory, compiles gogrep and install it into the directory.
func newGrepper() (*grepper, error) {
	workDir, err := os.MkdirTemp("", "gogrep")
	if err != nil {
		return nil, err
	}
	command := filepath.Join(workDir, "gogrep")
	if err := run("go", "build", "-o", command); err != nil {
		return nil, err
	}
	return &grepper{
		workDir: workDir,
		command: command,
	}, nil
}

func (s *grepper) close()                         { os.RemoveAll(s.workDir) }
func (s *grepper) filePath(name string) string    { return filepath.Join(s.workDir, name) }
func (s *grepper) copyFile(to, from string) error { return copyFile(s.filePath(to), s.filePath(from)) }
func (s *grepper) createFile(name string, content string) error {
	f, err := os.Create(s.filePath(name))
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, content)
	return err
}

func copyFile(to, from string) error {
	toFile, err := os.Create(to)
	if err != nil {
		return err
	}
	defer toFile.Close()
	fromFile, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fromFile.Close()
	_, err = io.Copy(toFile, fromFile)
	return err
}

func run(name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Dir = "."
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatalOnError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
