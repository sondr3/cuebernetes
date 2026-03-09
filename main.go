package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	//"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

func walkDirIgnores(d fs.DirEntry) error {
	if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
		return filepath.SkipDir
	} else if d.IsDir() && d.Name() == "cue.mod" {
		return filepath.SkipDir
	}
	return nil
}

func findCueFiles(path string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		err = walkDirIgnores(d)
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".cue" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func parseCue(files []string) error {
	ctx := cuecontext.New()
	instances := load.Instances(files, nil)
	values, err := ctx.BuildInstances(instances)
	if err != nil {
		panic(err)
	}

	// Parse the values
	for _, value := range values {
		out, err := yaml.Encode(value)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		//value.Walk(parseResource, nil)
	}
	return nil
}

func run(path, mode string, split bool) error {
	fmt.Printf("cuebernetes - %s in %s\n", mode, path)
	files, err := findCueFiles(path)
	if err != nil {
		return err
	}
	err = parseCue(files)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var path string
	var mode string
	var split bool

	app := &cli.Command{
		Name:                  "cuebernetes",
		Description:           "cuebernetes is a tool for converting Cue based k8s files to YAML",
		Version:               "0.1.0",
		EnableShellCompletion: true,
		Suggest:               true,

		Commands: []*cli.Command{
			{
				Name:   "man",
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					man, err := docs.ToMan(cmd.Root())
					if err != nil {
						return err
					}
					fmt.Println(man)
					return nil
				},
			},
			{
				Name:   "markdown",
				Hidden: true,
				Action: func(ctx context.Context, cmd *cli.Command) error {
					man, err := docs.ToMarkdown(cmd.Root())
					if err != nil {
						return err
					}
					fmt.Println(man)
					return nil
				},
			},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Aliases:     []string{"m"},
				Usage:       `Which mode to use (print|write)`,
				Value:       "print",
				DefaultText: "print",
				Destination: &mode,
				Validator: func(s string) error {
					switch s {
					case "print":
						return nil
					case "write":
						return nil
					default:
						return fmt.Errorf("invalid mode: %s, must be either 'print' or 'write'", s)
					}
				},
			},
			&cli.BoolFlag{
				Name:        "split",
				Aliases:     []string{"s"},
				Usage:       `Split the file into multiple YAML files`,
				Destination: &split,
			},
		},
		Arguments: []cli.Argument{
			&cli.StringArg{
				Name:        "path",
				UsageText:   `File or directory to convert`,
				Value:       ".",
				Destination: &path,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(path, mode, split)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
