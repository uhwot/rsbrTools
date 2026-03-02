package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	cmd := cli.Command{
		Name:  "rsbrTools",
		Usage: "tools for run sackboy run!",
		Commands: []*cli.Command{
			{
				Name:    "unpack",
				Aliases: []string{"u"},
				Usage:   "extracts a .pak file",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<pak file>"},
					&cli.StringArg{Name: "[output directory]", Value: "unpacked"},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "convert",
						Aliases: []string{"c"},
						Usage:   "convert texture/texture atlas files to png",
					},
					&cli.StringFlag{
						Name:    "pathlist",
						Aliases: []string{"p"},
						Usage:   "path to .pak path list file",
						Value:   "paths.txt",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					pak := cmd.StringArg("<pak file>")
					if pak == "" {
						return fmt.Errorf("pak file path not specified")
					}

					unpack(
						pak,
						cmd.StringArg("[output directory]"),
						cmd.Bool("convert"),
						cmd.String("pathlist"),
					)
					return nil
				},
			},
			{
				Name:    "pack",
				Aliases: []string{"p"},
				Usage:   "creates a .pak file from a directory",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<input directory>"},
					&cli.StringArg{Name: "[output file]", Value: "packed.pak"},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "pathlist",
						Aliases: []string{"p"},
						Usage:   "path where path list txt gets written to",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					directory := cmd.StringArg("<input directory>")
					if directory == "" {
						return fmt.Errorf("input directory not specified")
					}

					pack(directory, cmd.StringArg("[output file]"), cmd.String("pathlist"))
					return nil
				},
			},
			{
				Name:    "atcdecode",
				Aliases: []string{"d"},
				Usage:   "converts a .atc texture to .png",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<atc file>"},
					&cli.StringArg{Name: "[output path]", Value: "decoded.png"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					atc := cmd.StringArg("<atc file>")
					if atc == "" {
						return fmt.Errorf("atc file path not specified")
					}

					atcDecode(atc, cmd.StringArg("[output path]"))
					return nil
				},
			},
			{
				Name:    "atcencode",
				Aliases: []string{"e"},
				Usage:   "converts a png/jpg image to RGBA .atc",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<image file>"},
					&cli.StringArg{Name: "[output path]", Value: "encoded.atc"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					img := cmd.StringArg("<image file>")
					if img == "" {
						return fmt.Errorf("image file path not specified")
					}

					atcEncode(img, cmd.StringArg("[output path]"))
					return nil
				},
			},
			{
				Name:    "atlasunpack",
				Aliases: []string{"a"},
				Usage:   "unpacks atlas textures from a .atc and .atlas to pngs",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<atc file>"},
					&cli.StringArg{Name: "<atlas file>"},
					&cli.StringArg{Name: "[output path]", Value: "atlas"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					atcPath := cmd.StringArg("<atc file>")
					if atcPath == "" {
						return fmt.Errorf("atc file path not specified")
					}

					atlasPath := cmd.StringArg("<atlas file>")
					if atlasPath == "" {
						return fmt.Errorf("atlas file path not specified")
					}

					atc, err := os.Open(atcPath)
					check(err)
					defer atc.Close()

					img := readTexture(atc)

					atlasData, err := os.ReadFile(atlasPath)
					check(err)

					handleAtlas(img, atlasData, cmd.StringArg("[output path]"))

					return nil
				},
			},
			{
				Name:    "modeltextures",
				Aliases: []string{"m"},
				Usage:   "extracts textures from a .adrenomodel file",
				Arguments: []cli.Argument{
					&cli.StringArg{Name: "<model file>"},
					&cli.StringArg{Name: "[output path]", Value: "model_textures"},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "raw",
						Aliases: []string{"r"},
						Usage:   "extract raw ATC textures",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					modelPath := cmd.StringArg("<model file>")
					if modelPath == "" {
						return fmt.Errorf("model file path not specified")
					}

					model, err := os.Open(modelPath)
					check(err)
					defer model.Close()

					handleModelTextures(model, cmd.StringArg("[output path]"), !cmd.Bool("raw"))

					return nil
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
