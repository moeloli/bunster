package bunster_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yassinebenaid/bunster"
	"github.com/yassinebenaid/bunster/pkg/diff"
	"github.com/yassinebenaid/godump"
	"gopkg.in/yaml.v3"
)

type Test struct {
	NoParallel bool `yaml:"no-parallel"`
	Cases      []struct {
		Name       string            `yaml:"name"`
		Stdin      string            `yaml:"stdin"`
		RunsOn     string            `yaml:"runs_on"`
		SetupShell string            `yaml:"setup_shell"`
		Env        []string          `yaml:"env"`
		Args       []string          `yaml:"args"`
		Files      map[string]string `yaml:"files"`
		Timeout    int               `yaml:"timeout"`
		Script     string            `yaml:"script"`
		Expect     struct {
			Stdout   string            `yaml:"stdout"`
			Stderr   string            `yaml:"stderr"`
			ExitCode int               `yaml:"exit_code"`
			Files    map[string]string `yaml:"files"`
		} `yaml:"expect"`
	}
}

var dump = (&godump.Dumper{
	Theme:                   godump.DefaultTheme,
	ShowPrimitiveNamedTypes: true,
}).Sprintln

func TestBunster(t *testing.T) {
	var logger = log.New(os.Stdout, "", log.Ltime)
	filter := os.Getenv("FILTER")

	testFiles, err := globFiles("tests")
	if err != nil {
		t.Fatalf("Failed to `Glob` test files, %v", err)
	}

	currentWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get current working directory, %v", err)
	}

	for _, testFile := range testFiles {
		t.Run(testFile, func(t *testing.T) {

			testContent, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to open test file, %v", err)
			}

			var test Test

			if err := yaml.Unmarshal(testContent, &test); err != nil {
				t.Fatalf("Failed to parse test file, %v", err)
			}

			for i, testCase := range test.Cases {
				if !strings.Contains(testCase.Name, filter) && !strings.Contains(testFile, filter) {
					// we support filtering, someone would want to run specific tests.
					continue
				}
				if testCase.RunsOn != "" && testCase.RunsOn != runtime.GOOS {
					// some tests only run on specific platforms.
					continue
				}

				workdir := path.Join(os.TempDir(), "bunster-testing")
				if err := os.RemoveAll(workdir); err != nil {
					t.Fatalf("Failed to clean old workspace, %v", err)
				}
				if err := os.MkdirAll(workdir, 0700); err != nil {
					t.Fatalf("Failed to create workspace, %v", err)
				}
				rundir := path.Join(workdir, "run")
				if err := os.MkdirAll(rundir, 0700); err != nil {
					t.Fatalf("Failed to create run workdir, %v", err)
				}

				if testCase.SetupShell != "" {
					var setupShellStderr bytes.Buffer
					sh := exec.Command("bash", "-c", testCase.SetupShell) //nolint:gosec
					sh.Stderr = &setupShellStderr
					sh.Dir = workdir
					if err := sh.Run(); err != nil {
						t.Fatalf("\nTest(#%d): %sSetup Shell Error: %sStderr: %s", i, dump(testCase.Name), dump(err.Error()), dump(setupShellStderr.String()))
					}
				}

				for filename, content := range testCase.Files {
					if err := os.WriteFile(path.Join(rundir, filename), []byte(content), 0600); err != nil {
						t.Fatalf("\nTest(#%d): %sFailed to write file %q, %v", i, dump(testCase.Name), filename, err)
					}
				}

				if err := os.Chdir(workdir); err != nil {
					t.Fatalf("cannot change current working directory, %v", err)
				}
				defer func() {
					if err := os.Chdir(currentWorkingDirectory); err != nil {
						t.Fatalf("cannot change current working directory, %v", err)
					}
				}()

				binary, err := buildBinary(workdir, []rune(testCase.Script))
				if err != nil {
					t.Fatalf("\nTest(#%d): %sBuild Error: %s", i, dump(testCase.Name), err.Error())
				}

				var stdout, stderr bytes.Buffer
				cmd := exec.Command(binary, testCase.Args...)
				cmd.Stdin = strings.NewReader(testCase.Stdin)
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr
				cmd.Dir = rundir
				cmd.Env = append(os.Environ(), testCase.Env...)
				if err := cmd.Start(); err != nil {
					t.Fatalf("\nTest(#%d): %sRuntime Error: %s", i, dump(testCase.Name), dump(err.Error()))
				}

				var done = make(chan error, 1)
				go func() {
					done <- cmd.Wait()
					close(done)
				}()

				if testCase.Timeout == 0 {
					testCase.Timeout = 1
				}

				select {
				case err := <-done:
					if _, ok := err.(*exec.ExitError); !ok && err != nil {
						t.Fatalf("\nTest(#%d): %sRuntime Error: %s", i, dump(testCase.Name), dump(err.Error()))
					}
				case <-time.After(time.Second * time.Duration(testCase.Timeout)):
					defer func() { _ = cmd.Process.Kill() }()

					t.Fatalf("\nTest(#%d): %sRuntime Error: process exceeded timeout of %d seconds ", i, dump(testCase.Name), time.Duration(testCase.Timeout))
				}

				if testCase.Expect.Stderr != stderr.String() {
					t.Fatalf("\nTest(#%d): %sExpected `STDERR` does not match actual value\ndiff:\n%s",
						i, dump(testCase.Name), diff.DiffBG(testCase.Expect.Stderr, stderr.String()))
				}

				if testCase.Expect.ExitCode != cmd.ProcessState.ExitCode() {
					t.Fatalf("\nTest(#%d): %sExpected exit code of '%d', got '%d'",
						i, dump(testCase.Name), testCase.Expect.ExitCode, cmd.ProcessState.ExitCode())
				}

				if testCase.Expect.Stdout != stdout.String() {
					t.Fatalf("\nTest(#%d): %sExpected `STDOUT` does not match actual value\ndiff:\n%s",
						i, dump(testCase.Name), diff.DiffBG(testCase.Expect.Stdout, stdout.String()))
				}

				files, err := filepath.Glob(rundir + "/*")
				if err != nil {
					t.Fatalf("\nTest(#%d): %sFailed to glob the working directory, %v", i, dump(testCase.Name), err)
				}
				if testCase.Expect.Files != nil && len(files) != len(testCase.Expect.Files) {
					t.Fatalf("\nTest(#%d): %sExpected files in working directory does not match actual count, files count is: %d, expected files count: %d",
						i, dump(testCase.Name), len(files), len(testCase.Expect.Files))
				}

				for filename, expectedContent := range testCase.Expect.Files {
					content, err := os.ReadFile(path.Join(rundir, filename))
					if err != nil {
						t.Fatalf("\nTest(#%d): %sFailed to read file %q, %v", i, dump(testCase.Name), filename, err)
					}

					if string(content) != expectedContent {
						t.Fatalf("\nTest(#%d): %sExpected file content doesn't match actual content\nfile: %s\ndiff:\n%s",
							i, dump(testCase.Name), filename, diff.DiffBG(expectedContent, string(content)))
					}
				}
				logger.Print(dump(testCase.Name))
			}
		})
	}
}

func buildBinary(workdir string, s []rune) (string, error) {
	if err := bunster.Generate(workdir, s); err != nil {
		return "", err
	}

	var stdout, stderr bytes.Buffer
	gocmd := exec.Command("go", "build", "-o", "build.bin")
	gocmd.Stdout = &stdout
	gocmd.Stderr = &stderr
	if err := gocmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed\nError: %sStderr: %sStdout: %s", dump(err.Error()), dump(stderr.String()), dump(stdout.String()))
	}

	return path.Join(workdir, "build.bin"), nil
}

func globFiles(path string) ([]string, error) {
	var paths []string

	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		paths = append(paths, path)
		return nil
	})

	return paths, err
}
