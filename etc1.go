package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"io"
)

// code based on
// https://github.com/K0lb3/texture2ddecoder/blob/a111654b8b891e72b8dca19a0f06551dcf63b1a0/src/Texture2DDecoder/etc.cpp#L40

var Etc1SubblockTable [2][16]byte = [2][16]byte{
	{0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1},
	{0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 1, 1},
}

var Etc1ModifierTable [8][2]byte = [8][2]byte{
	{2, 8}, {5, 17}, {9, 29}, {13, 42},
	{18, 60}, {24, 80}, {33, 106}, {47, 183},
}

func clamp(n int) byte {
	if n < 0 {
		return 0
	}

	if n > 255 {
		return 255
	}

	return byte(n)
}

func applicateColor(c [3]byte, m int) color.NRGBA {
	return color.NRGBA{
		R: clamp(int(c[0]) + m),
		G: clamp(int(c[1]) + m),
		B: clamp(int(c[2]) + m),
		A: 255,
	}
}

func decodeEtc1Block(data []byte) *image.NRGBA {
	code := [2]byte{data[3] >> 5, data[3] >> 2 & 7} // table codewords
	table := Etc1SubblockTable[data[3]&1]
	var c [2][3]byte

	if (data[3] & 2) > 0 {
		// diff bit == 1
		c[0][0] = data[0] & 0xf8
		c[0][1] = data[1] & 0xf8
		c[0][2] = data[2] & 0xf8
		c[1][0] = c[0][0] + (data[0] << 3 & 0x18) - (data[0] << 3 & 0x20)
		c[1][1] = c[0][1] + (data[1] << 3 & 0x18) - (data[1] << 3 & 0x20)
		c[1][2] = c[0][2] + (data[2] << 3 & 0x18) - (data[2] << 3 & 0x20)
		c[0][0] |= c[0][0] >> 5
		c[0][1] |= c[0][1] >> 5
		c[0][2] |= c[0][2] >> 5
		c[1][0] |= c[1][0] >> 5
		c[1][1] |= c[1][1] >> 5
		c[1][2] |= c[1][2] >> 5
	} else {
		// diff bit == 0
		c[0][0] = (data[0] & 0xf0) | data[0]>>4
		c[1][0] = (data[0] & 0x0f) | data[0]<<4
		c[0][1] = (data[1] & 0xf0) | data[1]>>4
		c[1][1] = (data[1] & 0x0f) | data[1]<<4
		c[0][2] = (data[2] & 0xf0) | data[2]>>4
		c[1][2] = (data[2] & 0x0f) | data[2]<<4
	}

	j := binary.BigEndian.Uint16(data[6:8])
	k := binary.BigEndian.Uint16(data[4:6])

	block := image.NewNRGBA(image.Rect(0, 0, 4, 4))

	idx := 0

	for x := range 4 {
		for y := range 4 {
			s := table[idx]
			m := int(Etc1ModifierTable[code[s]][j&1])
			if (k & 1) > 0 {
				m = -m
			}

			block.SetNRGBA(x, y, applicateColor(c[s], m))

			j >>= 1
			k >>= 1
			idx++
		}
	}

	return block
}

func decodeEtc1(reader io.Reader, width, height int) (image.Image, error) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	numBlocksX := (width + 3) / 4
	numBlocksY := (height + 3) / 4

	data := make([]byte, 8)

	for by := range numBlocksY {
		for bx := range numBlocksX {
			_, err := reader.Read(data)
			if err != nil {
				return nil, err
			}

			block := decodeEtc1Block(data)

			for subY := range 4 {
				y := (by * 4) + subY
				if y >= height {
					continue
				}

				for subX := range 4 {
					x := (bx * 4) + subX
					if x >= width {
						continue
					}

					pixel := block.At(subX, subY)
					img.Set(x, height-1-y, pixel)
				}
			}
		}
	}

	return img, nil
}
