package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"io"
	"os"
	"path/filepath"
)

type ModelHeader struct {
	Magic         uint32
	Unknown1      [2]uint32
	TextureCount  uint32
	MeshCount     uint32
	Unknown2      uint32
	TextureOffset uint32
	Unknown3      [7]uint32
}

type ModelTexture struct {
	Filename   [64]byte
	DataSize   uint32
	DataOffset uint32
	Flags      uint32
}

func clen(n []byte) int {
	for i := 0; i < len(n); i++ {
		if n[i] == 0 {
			return i
		}
	}
	return len(n)
}

func cstring(data []byte) string {
	data = data[:clen(data)]
	return string(data)
}

func handleModelTextures(reader io.ReadSeeker, outPath string, toPng bool) {
	header := ModelHeader{}
	err := binary.Read(reader, binary.LittleEndian, &header)
	check(err)

	_, err = reader.Seek(int64(header.TextureOffset), io.SeekStart)
	check(err)

	textures := make([]ModelTexture, header.TextureCount)
	err = binary.Read(reader, binary.LittleEndian, &textures)
	check(err)

	err = os.MkdirAll(outPath, 0755)
	check(err)

	for _, texture := range textures {
		_, err = reader.Seek(int64(texture.DataOffset), io.SeekStart)
		check(err)

		data := make([]byte, texture.DataSize)
		_, err = reader.Read(data)
		check(err)

		filename := cstring(texture.Filename[:])

		if !toPng {
			filename = fmt.Sprintf("%s.atc", filename)
			filePath := filepath.Join(outPath, filename)

			err = os.WriteFile(filePath, data, 0644)
			check(err)
			continue
		}

		textureReader := bytes.NewReader(data)
		img := readTexture(textureReader)

		filename = fmt.Sprintf("%s.png", filename)
		filePath := filepath.Join(outPath, filename)

		pngFile, err := os.Create(filePath)
		check(err)

		encoder := &png.Encoder{
			CompressionLevel: png.BestSpeed,
		}
		err = encoder.Encode(pngFile, img)
		check(err)

		pngFile.Close()
	}
}
