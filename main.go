package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Name:        "cuebernetes",
		Description: "cuebernetes is a tool for converting Cue based k8s files to YAML",
		Version:     "0.1.0",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "mode",
				Aliases:     []string{"m"},
				Usage:       `Which mode to use (print|write)`,
				Value:       "print",
				DefaultText: "print",
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
				Name:    "split",
				Aliases: []string{"s"},
				Usage:   `Split the file into multiple YAML files`,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			fmt.Printf("cuebernetes - %s\n", cmd.String("mode"))
			return nil
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
