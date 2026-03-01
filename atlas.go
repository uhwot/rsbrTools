package main

import (
	"bufio"
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var lineRegex = regexp.MustCompile(`^(\w+) = (\d+) (\d+) (\d+) (\d+)$`)

type AtlasTexture struct {
	name   string
	x      uint64
	y      uint64
	width  uint64
	height uint64
}

// https://stackoverflow.com/questions/16072910/trouble-getting-a-subimage-of-an-image-in-go
type SubImager interface {
	SubImage(r image.Rectangle) image.Image
}

func parseTextures(atlasData []byte) []AtlasTexture {
	textures := make([]AtlasTexture, 0)

	reader := bytes.NewReader(atlasData)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		texture := AtlasTexture{}

		var err error

		texture.name = matches[1]
		texture.x, err = strconv.ParseUint(matches[2], 10, 32)
		check(err)
		texture.y, err = strconv.ParseUint(matches[3], 10, 32)
		check(err)
		texture.width, err = strconv.ParseUint(matches[4], 10, 32)
		check(err)
		texture.height, err = strconv.ParseUint(matches[5], 10, 32)
		check(err)

		textures = append(textures, texture)
	}

	return textures
}

func handleAtlas(img image.Image, atlasData []byte, outPath string) {
	err := os.MkdirAll(outPath, os.ModePerm)
	check(err)

	atlas := img.(SubImager)
	atlasTextures := parseTextures(atlasData)

	for _, texture := range atlasTextures {
		x, y := int(texture.x), int(texture.y)
		img := atlas.SubImage(image.Rect(x, y, x+int(texture.width), y+int(texture.height)))

		filename := fmt.Sprintf("%s.png", texture.name)
		outPath := filepath.Join(outPath, filename)

		outFile, err := os.Create(outPath)
		check(err)
		defer outFile.Close()

		err = png.Encode(outFile, img)
		check(err)
	}
}
