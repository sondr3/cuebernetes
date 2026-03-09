package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
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

func isManifest(v cue.Value) (bool, error) {
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
		return true, nil
	}
	return false, nil
}

func (h *Handler) parseFile(file string) error {
	ctx := cuecontext.New()
	instances := load.Instances([]string{file}, nil)
	values, err := ctx.BuildInstances(instances)
	if err != nil {
		return fmt.Errorf("building CUE instances for %s: %w", file, err)
	}

	var errs []error
	for _, value := range values {
		ok, err := isManifest(value)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file, err))
			continue
		}
		if ok {
			val, err := yaml.Encode(value)
			if err != nil {
				return err
			}
			name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
			h.Manifests[file] = append(h.Manifests[file], Manifest{
				Name:  name,
				Value: val,
			})
			continue
		}

		iter, err := value.Fields()
		if err != nil {
			return err
		}
		for iter.Next() {
			label := iter.Selector().String()
			v := iter.Value()
			ok, err := isManifest(v)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %s: %w", file, label, err))
				continue
			}
			if ok {
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
	return errors.Join(errs...)
}

func StringifyManifests(file string, manifests []Manifest) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "# generated from %s -- DO NOT EDIT\n", file)
	for i, manifest := range manifests {
		b.Write(manifest.Value)
		if i != len(manifests)-1 {
			b.WriteString("---\n")
		}
	}
	return b.String()
}

func (h *Handler) sortedFiles() []string {
	files := make([]string, 0, len(h.Manifests))
	for file := range h.Manifests {
		files = append(files, file)
	}
	slices.Sort(files)
	return files
}

func (h *Handler) Print() string {
	var b strings.Builder
	files := h.sortedFiles()
	for i, file := range files {
		b.WriteString(StringifyManifests(file, h.Manifests[file]))
		if i != len(files)-1 {
			b.WriteString("---\n")
		}
	}
	return b.String()
}

func (h *Handler) Write(out string, split bool) error {
	if err := os.MkdirAll(out, 0755); err != nil {
		return err
	}

	files := h.sortedFiles()
	for _, file := range files {
		manifests := h.Manifests[file]
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
		} else {
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
	}

	fmt.Printf("Wrote %d file(s)\n", len(files))
	return nil
}

func run(path, out, mode string, split bool) error {
	files, err := findCueFiles(path)
	if err != nil {
		return err
	}
	handler := Handler{Manifests: make(map[string][]Manifest)}
	var errs []error
	for _, file := range files {
		if err := handler.parseFile(file); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
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
		Usage:                 "Convert CUE Kubernetes manifests to YAML",
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
					if s != "print" && s != "write" {
						return fmt.Errorf("invalid mode: %s, must be either 'print' or 'write'", s)
					}
					return nil
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
