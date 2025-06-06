package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gofrs/flock"
	"github.com/urfave/cli/v3"
	"github.com/yassinebenaid/bunster"
	"github.com/yassinebenaid/bunster/builder"
	"github.com/yassinebenaid/bunster/lexer"
	"github.com/yassinebenaid/bunster/parser"
	"github.com/yassinebenaid/godump"
)

func main() {
	app := cli.Command{
		Name:      "bunster",
		Usage:     "compile shell scripts to static binaries",
		Version:   strings.TrimSpace(bunster.Version),
		Authors:   []any{"Yassine Benaid <yassinebenaide3@gmail.com>"},
		Copyright: "2024-2025, Yassine Benaid",
		Suggest:   true,
		Commands: []*cli.Command{
			{
				Name:   "ast",
				Usage:  "Print the script ast",
				Action: ast,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "no-ansi", Aliases: []string{"n"}},
				},
			},
			{
				Name:   "build",
				Usage:  "Build a module",
				Action: build,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Required: true},
				},
			},
			{
				Name:   "generate",
				Usage:  "Generate the Go source out of a module",
				Action: geneate,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Required: true},
				},
			},
			{
				Name:   "get",
				Usage:  "get a module from a remote registry.",
				Action: get,
				Flags:  []cli.Flag{&cli.BoolFlag{Name: "missing"}},
			},
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(1)
	}
}

func ast(_ context.Context, cmd *cli.Command) error {
	filename := cmd.Args().Get(0)
	v, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	script, err := parser.Parse(
		lexer.New([]rune(string(v))),
	)

	if err != nil {
		return err
	}

	var d godump.Dumper
	d.ShowPrimitiveNamedTypes = true

	if !cmd.Bool("no-ansi") {
		d.Theme = godump.DefaultTheme
	}

	return d.Println(script)
}

func build(_ context.Context, cmd *cli.Command) error {
	destination := cmd.String("o")
	if !path.IsAbs(destination) {
		currWorkdir, err := os.Getwd()
		if err != nil {
			return err
		}
		destination = path.Join(currWorkdir, destination)
	}

	builder := builder.Builder{
		Home:       path.Join(os.Getenv("HOME"), ".bunster"),
		Workdir:    ".",
		Builddir:   path.Join(os.TempDir(), "bunster-build"),
		OutputFile: destination,
		MainScript: cmd.Args().First(),
	}

	return builder.Build()
}

func geneate(_ context.Context, cmd *cli.Command) error {
	builder := builder.Builder{
		Home:       path.Join(os.Getenv("HOME"), ".bunster"),
		Workdir:    ".",
		Builddir:   cmd.String("o"),
		MainScript: cmd.Args().First(),
		Gofmt:      true,
	}

	return builder.Generate()
}

func get(_ context.Context, cmd *cli.Command) error {
	lock := flock.New(path.Join(os.TempDir(), "bunster.lock"))
	locked, err := lock.TryLock()
	if err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("this command is currently running in another process")
	}
	defer func() { _ = lock.Unlock() }()

	builder := builder.Builder{
		Home:    path.Join(os.Getenv("HOME"), ".bunster"),
		Workdir: ".",
	}

	return builder.ResolveDeps(cmd.Args().Slice(), cmd.Bool("missing"))
}
