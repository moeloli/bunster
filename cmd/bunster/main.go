package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/yassinebenaid/bunster"
	"github.com/yassinebenaid/bunster/builder"
	"github.com/yassinebenaid/bunster/lexer"
	"github.com/yassinebenaid/bunster/parser"
	"github.com/yassinebenaid/godump"
)

func main() {
	app := cli.Command{
		Name:  "bunster",
		Usage: "compile shell script to self-contained executable programs",
		Commands: []*cli.Command{
			{
				Name:   "ast",
				Usage:  "Print the script ast",
				Action: astCMD,
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "no-ansi", Aliases: []string{"n"}},
				},
			},
			{
				Name:   "build",
				Usage:  "Build a module",
				Action: buildCMD,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Required: true},
				},
			},
			{
				Name:   "generate",
				Usage:  "Generate the Go source out of a module",
				Action: geneateCMD,
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "o", Required: true},
				},
			},
			{
				Name:  "version",
				Usage: "Print bunster version",
				Action: func(ctx context.Context, c *cli.Command) error {
					fmt.Println(strings.TrimSpace(bunster.Version))
					return nil
				},
			},
		},
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
		os.Exit(1)
	}
}

func astCMD(_ context.Context, cmd *cli.Command) error {
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

func buildCMD(_ context.Context, cmd *cli.Command) error {
	destination := cmd.String("o")
	if !path.IsAbs(destination) {
		currWorkdir, err := os.Getwd()
		if err != nil {
			return err
		}
		destination = path.Join(currWorkdir, destination)
	}

	builder := builder.Builder{
		Workdir:    ".",
		Builddir:   path.Join(os.TempDir(), "bunster-build"),
		OutputFile: destination,
		MainScript: cmd.Args().First(),
	}

	return builder.Build()
}

func geneateCMD(_ context.Context, cmd *cli.Command) error {
	builder := builder.Builder{
		Workdir:    ".",
		Builddir:   cmd.String("o"),
		MainScript: cmd.Args().First(),
		Gofmt:      true,
	}

	return builder.Generate()
}
