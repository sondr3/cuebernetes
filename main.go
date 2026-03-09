package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/encoding/yaml"
	docs "github.com/urfave/cli-docs/v3"
	"github.com/urfave/cli/v3"
)

type Manifest struct {
	Name  string
	Value []byte
}

type Handler struct {
	Manifests map[string][]Manifest
}

func walkDirIgnores(d fs.DirEntry) error {
	if d.IsDir() {
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		} else if d.Name() == "cue.mod" {
			return filepath.SkipDir
		}
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

func parseManifest(v cue.Value) (bool, error) {
	apiVersion := v.LookupPath(cue.ParsePath("apiVersion"))
	kind := v.LookupPath(cue.ParsePath("kind"))
	if apiVersion.Exists() && kind.Exists() {
		err := v.Validate(
			cue.All(),
			cue.Attributes(true),
			cue.Definitions(true),
			cue.InlineImports(true),
			cue.Concrete(true),
			cue.Final(),
			cue.DisallowCycles(true),
			cue.Hidden(true),
			cue.Optional(true),
		)
		if err != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

func (h *Handler) parseFile(file string) error {
	ctx := cuecontext.New()
	instances := load.Instances([]string{file}, nil)
	values, err := ctx.BuildInstances(instances)
	if err != nil {
		panic(err)
	}

	for _, value := range values {
		iter, err := value.Fields()
		if err != nil {
			return err
		}
		for iter.Next() {
			label := iter.Selector().String()
			v := iter.Value()
			isManifest, err := parseManifest(v)
			if err != nil {
				return err
			}
			if !isManifest {
				val, err := yaml.Encode(v)
				if err != nil {
					return err
				}
				h.Manifests[file] = append(h.Manifests[file], Manifest{
					Name:  label,
					Value: val,
				})
			}
		}
	}
	return nil
}

func StringifyManifests(file string, manifests []Manifest) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# generated from %s DO NOT EDIT\n", file))
	for i, manifest := range manifests {
		b.Write(manifest.Value)
		if i != len(manifests)-1 {
			b.WriteString("---\n")
		}
	}
	return b.String()
}

func (h *Handler) Print() string {
	var b strings.Builder
	lbl := 0
	for file, manifests := range h.Manifests {
		b.WriteString(StringifyManifests(file, manifests))
		if lbl != len(h.Manifests)-1 {
			b.WriteString("---\n")
			lbl++
		}
	}
	return b.String()
}

func (h *Handler) Write(out string, split bool) error {
	_, err := os.Stat(out)
	if os.IsNotExist(err) {
		err := os.Mkdir(out, 0755)
		if err != nil && !os.IsExist(err) {
			return err
		}
	}

	for file, manifests := range h.Manifests {
		if split {
			for _, manifest := range manifests {
				dir := filepath.Join(out, filepath.Dir(file))
				err := os.MkdirAll(dir, 0755)
				if err != nil {
					return err
				}
				newFile := filepath.Join(out, strings.TrimSuffix(file, filepath.Ext(file))+"-"+strings.ToLower(manifest.Name)+".yaml")
				err = os.WriteFile(newFile, manifest.Value, 0644)
				if err != nil {
					return err
				}
			}
		}

		dir := filepath.Join(out, filepath.Dir(file))
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		newFile := filepath.Join(out, strings.TrimSuffix(file, filepath.Ext(file))+".yaml")
		err = os.WriteFile(newFile, []byte(StringifyManifests(file, manifests)), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func run(path, out, mode string, split bool) error {
	files, err := findCueFiles(path)
	if err != nil {
		return err
	}
	handler := Handler{Manifests: make(map[string][]Manifest)}
	for _, file := range files {
		err = handler.parseFile(file)
	}
	if err != nil {
		return err
	}
	switch mode {
	case "print":
		fmt.Println(handler.Print())
	case "write":
		err := handler.Write(out, split)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var path string
	var out string
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
			&cli.StringFlag{
				Name:        "out",
				Aliases:     []string{"o"},
				Usage:       "Directory to output files to",
				Value:       "_yaml",
				DefaultText: "_yaml",
				Destination: &out,
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
			return run(path, out, mode, split)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
