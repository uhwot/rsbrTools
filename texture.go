package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"os"

	_ "image/jpeg"
)

type AtcFormat uint32

const (
	GL_RGBA                        AtcFormat = 0
	GL_ATC_RGB_AMD                 AtcFormat = 1
	GL_ATC_RGBA_EXPLICIT_ALPHA_AMD AtcFormat = 2
	GL_ETC1_RGB8_OES               AtcFormat = 3
)

type AtcHeader struct {
	Magic      uint32
	DataOffset uint32
	DataSize   uint32
	Format     AtcFormat
	Width      uint32
	Height     uint32
}

func handleRGBA(reader io.Reader, width, height int) (image.Image, error) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	for y := height - 1; y >= 0; y-- {
		for x := range width {
			pixel := color.NRGBA{}
			err := binary.Read(reader, binary.LittleEndian, &pixel)
			if err != nil {
				return nil, err
			}

			img.SetNRGBA(x, y, pixel)
		}
	}

	return img, nil
}

func readTexture(reader io.ReadSeeker) image.Image {
	header := AtcHeader{}
	err := binary.Read(reader, binary.LittleEndian, &header)
	check(err)

	if header.Magic != 0xC0FF33 {
		panic("invalid ATC magic")
	}

	_, err = reader.Seek(int64(header.DataOffset), io.SeekStart)
	check(err)

	var img image.Image

	switch header.Format {
	case GL_RGBA:
		img, err = handleRGBA(reader, int(header.Width), int(header.Height))
	case GL_ETC1_RGB8_OES:
		img, err = decodeEtc1(reader, int(header.Width), int(header.Height))
	default:
		panic("format is unsupported :(")
	}
	check(err)

	return img
}

func convertImageToPng(img image.Image) []byte {
	var pngData bytes.Buffer
	pngWriter := bufio.NewWriterSize(&pngData, 64*1024)

	encoder := &png.Encoder{
		CompressionLevel: png.BestSpeed,
	}
	err := encoder.Encode(pngWriter, img)
	check(err)

	err = pngWriter.Flush()
	check(err)

	return pngData.Bytes()
}

func atcDecode(atcPath string, pngPath string) {
	atcFile, err := os.Open(atcPath)
	check(err)

	img := readTexture(atcFile)

	atcFile.Close()

	pngFile, err := os.Create(pngPath)
	check(err)
	defer pngFile.Close()

	err = png.Encode(pngFile, img)
	check(err)
}

func toNRGBA(img image.Image) *image.NRGBA {
	if img, ok := img.(*image.NRGBA); ok {
		return img
	}

	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)

	draw.Draw(nrgba, bounds, img, bounds.Min, draw.Src)

	return nrgba
}

func atcEncode(imgPath string, atcPath string) {
	imgFile, err := os.Open(imgPath)
	check(err)

	img, _, err := image.Decode(imgFile)
	check(err)

	imgFile.Close()

	nrgbaImg := toNRGBA(img)
	imgSize := nrgbaImg.Bounds().Size()
	imgByteSize := (imgSize.X * imgSize.Y) * 4

	atc, err := os.Create(atcPath)
	check(err)
	defer atc.Close()

	header := AtcHeader{
		Magic:      0xC0FF33,
		DataOffset: 24 + 12,
		DataSize:   uint32(imgByteSize),
		Format:     GL_RGBA,
		Width:      uint32(imgSize.X),
		Height:     uint32(imgSize.Y),
	}

	err = binary.Write(atc, binary.LittleEndian, &header)
	check(err)

	// textures seem to always have 12 bytes of padding before the actual data for some reason
	err = binary.Write(atc, binary.LittleEndian, make([]byte, 12))
	check(err)

	for y := imgSize.Y - 1; y >= 0; y-- {
		for x := range imgSize.X {
			pixel := nrgbaImg.At(x, y)
			err = binary.Write(atc, binary.LittleEndian, pixel)
			check(err)
		}
	}
}
