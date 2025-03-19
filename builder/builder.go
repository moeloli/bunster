package builder

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yassinebenaid/bunster"
	"github.com/yassinebenaid/bunster/analyser"
	"github.com/yassinebenaid/bunster/generator"
	"github.com/yassinebenaid/bunster/ir"
	"github.com/yassinebenaid/bunster/lexer"
	"github.com/yassinebenaid/bunster/parser"
)

type Builder struct {
	Workdir    string
	Builddir   string
	OutputFile string
}

func (b *Builder) Build() error {
	if err := b.prepare(); err != nil {
		return err
	}

	v, err := os.ReadFile(path.Join(b.Workdir, "main.sh"))
	if err != nil {
		return err
	}

	script, err := parser.Parse(lexer.New([]rune(string(v))))
	if err != nil {
		return err
	}

	if err := analyser.Analyse(script); err != nil {
		return err
	}

	program := generator.Generate(script)

	err = os.WriteFile(path.Join(b.Builddir, "program.go"), []byte(program.String()), 0600)
	if err != nil {
		return err
	}

	if err := b.cloneRuntime(); err != nil {
		return err
	}

	if err := b.cloneStubs(); err != nil {
		return err
	}

	if err := b.cloneEmbeddedFiles(program.Embeds); err != nil {
		return err
	}

	gocmd := exec.Command("go", "build", "-o", b.OutputFile)
	gocmd.Stdin = os.Stdin
	gocmd.Stdout = os.Stdout
	gocmd.Stderr = os.Stderr
	gocmd.Dir = b.Builddir
	if err := gocmd.Run(); err != nil {
		return err
	}

	return nil
}

func (b *Builder) prepare() error {
	if err := os.RemoveAll(b.Builddir); err != nil {
		return err
	}

	if err := os.MkdirAll(b.Builddir, 0700); err != nil {
		return err
	}

	if err := os.MkdirAll(path.Join(b.Builddir, ir.EmbedDirectory), 0700); err != nil {
		return err
	}

	return nil
}

func (b *Builder) cloneRuntime() error {
	return fs.WalkDir(bunster.RuntimeFS, "runtime", func(dpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		dst := path.Join(b.Builddir, dpath)

		if d.IsDir() {
			return os.MkdirAll(dst, 0766)
		}

		if strings.HasSuffix(dpath, "_test.go") {
			return nil
		}

		content, err := bunster.RuntimeFS.ReadFile(dpath)
		if err != nil {
			return err
		}

		return os.WriteFile(dst, content, 0600)
	})
}

func (b *Builder) cloneStubs() error {
	if err := os.WriteFile(path.Join(b.Builddir, "main.go"), bunster.MainGo, 0600); err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(b.Builddir, "go.mod"), bunster.Gomod, 0600); err != nil {
		return err
	}

	return nil
}

func (b *Builder) cloneEmbeddedFiles(files []string) error {
	for _, file := range files {
		src, dst := file, path.Join(b.Builddir, ir.EmbedDirectory, file)

		info, err := os.Stat(src)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := copyDir(src, path.Join(b.Builddir, ir.EmbedDirectory)); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, dst); err != nil {
				return err
			}
		}
	}

	return nil
}

var specialPathRegex = regexp.MustCompile(`^(.*\.git.*)|(.*go\.mod.*)$`)

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(_path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if specialPathRegex.MatchString(_path) {
			return nil
		}

		if info.IsDir() {
			return os.MkdirAll(path.Join(dst, _path), 0766)
		}

		return copyFile(_path, path.Join(dst, _path))
	})
}

func copyFile(src, dst string) error {
	srcf, err := os.OpenFile(src, os.O_RDONLY, 000)
	if err != nil {
		return err
	}
	defer srcf.Close()

	dir, _ := path.Split(dst)
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}

	dstf, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer dstf.Close()

	if _, err := io.Copy(dstf, srcf); err != nil {
		return err
	}

	return nil
}
