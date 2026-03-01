package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const PakMagic uint32 = 0x50414b31

var XorBuffer = makeXorBuffer(0xbaff1ed, 0x305)

type PakHeader struct {
	Magic      uint32
	EntryCount uint32
	Unknown    [257]uint32
}

type PakEntry struct {
	Crc32            uint32
	Unknown          uint32
	UncompressedSize uint32
	CompressedSize   uint32
	Offset           uint32
}

func makeXorBuffer(seed uint32, size uint32) []byte {
	prng := NewMT19937()
	prng.Seed(uint64(seed))

	buffer := make([]byte, size)
	for i := range len(buffer) {
		buffer[i] = byte(prng.Uint32() & 0xFF)
	}

	return buffer
}

func makeCrcTable(pathListTxt string) map[uint32]string {
	file, err := os.Open(pathListTxt)
	check(err)
	defer file.Close()

	table := make(map[uint32]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		path := scanner.Text()
		path = strings.TrimSpace(path)
		if len(path) == 0 {
			continue
		}

		table[PakStringCrc32(path)] = path
	}

	return table
}

func getFileData(pak *os.File, entry PakEntry) []byte {
	pak.Seek(int64(entry.Offset), io.SeekStart)

	data := make([]byte, entry.CompressedSize)
	_, err := io.ReadFull(pak, data)
	check(err)

	for position := range entry.CompressedSize {
		xorIndex := (entry.Offset + position) % uint32(len(XorBuffer))
		data[position] ^= XorBuffer[xorIndex]
	}

	// zlib decompression
	if entry.CompressedSize < entry.UncompressedSize {
		reader := bytes.NewReader(data)
		zlibReader, err := zlib.NewReader(reader)
		check(err)
		defer zlibReader.Close()

		data = make([]byte, entry.UncompressedSize)

		_, err = io.ReadFull(zlibReader, data)
		check(err)
	}

	return data
}

func unpack(pakPath string, outPath string, convert bool, pathListTxt string) {
	crcTable := makeCrcTable(pathListTxt)

	pak, err := os.Open(pakPath)
	check(err)
	defer pak.Close()

	pakHeader := PakHeader{}
	binary.Read(pak, binary.LittleEndian, &pakHeader)

	// PAK1
	if pakHeader.Magic != PakMagic {
		panic("invalid PAK magic")
	}

	entries := make([]PakEntry, pakHeader.EntryCount)
	binary.Read(pak, binary.LittleEndian, &entries)

	for _, entry := range entries {
		data := getFileData(pak, entry)

		path, crcFound := crcTable[entry.Crc32]
		if !crcFound {
			if len(data) >= 4 {
				switch string(data[:4]) {
				// 0xC0FF33
				case "\x33\xFF\xC0\x00":
					path = fmt.Sprintf("unknown/textures/%x.atc", entry.Crc32)
				case "GEOM":
					path = fmt.Sprintf("unknown/models/%x.adrenomodel", entry.Crc32)
				case "PFAB":
					path = fmt.Sprintf("unknown/prefabs/%x.pfb", entry.Crc32)
				default:
					path = fmt.Sprintf("unknown/%x", entry.Crc32)
				}
			} else {
				path = fmt.Sprintf("unknown/%x", entry.Crc32)
			}
		}

		if convert && strings.HasSuffix(path, ".atc") {
			reader := bytes.NewReader(data)
			img := readTexture(reader)

			pathWithoutExt := strings.TrimSuffix(path, ".atc")
			atlasPath := pathWithoutExt + ".atlas"
			atlasCrc := PakStringCrc32(atlasPath)

			// this is probably not well optimized. too bad!
			foundAtlas := false
			for _, entry := range entries {
				if entry.Crc32 == atlasCrc {
					atlasData := getFileData(pak, entry)
					handleAtlas(img, atlasData, filepath.Join(outPath, pathWithoutExt))

					foundAtlas = true
					break
				}
			}

			if foundAtlas {
				continue
			}

			data = convertImageToPng(img)
			path = pathWithoutExt + ".png"
		}

		path = filepath.Join(outPath, path)

		err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
		check(err)

		out, err := os.Create(path)
		check(err)
		defer out.Close()

		_, err = out.Write(data)
		check(err)
	}
}

func pack(dirPath string, outPath string, pathTxtPath string) {
	var err error
	var pathTxt *os.File
	if pathTxtPath != "" {
		pathTxt, err = os.Create(pathTxtPath)
		check(err)
		defer pathTxt.Close()
	}

	filePaths := make(map[uint32]string)

	err = filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		check(err)

		if d.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(dirPath, path)
		check(err)

		var crc32 uint32
		unknownDirPath := fmt.Sprintf("unknown%c", os.PathSeparator)

		if strings.HasPrefix(relativePath, unknownDirPath) {
			fileName := filepath.Base(relativePath)
			fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

			crc32_64, err := strconv.ParseUint(fileName, 16, 32)
			check(err)

			crc32 = uint32(crc32_64)
		} else {
			crc32 = PakStringCrc32(relativePath)

			if pathTxt != nil {
				_, err = pathTxt.WriteString(relativePath + "\n")
				check(err)
			}
		}

		filePaths[crc32] = path

		return nil
	})
	check(err)

	pakFile, err := os.Create(outPath)
	check(err)
	defer pakFile.Close()

	numFiles := uint32(len(filePaths))

	err = binary.Write(pakFile, binary.LittleEndian, PakHeader{
		Magic:      PakMagic,
		EntryCount: numFiles,
		Unknown:    [257]uint32{},
	})
	check(err)

	entryOffset, err := pakFile.Seek(0, io.SeekCurrent)
	check(err)

	// PakEntry structs to be written later
	for range numFiles {
		err = binary.Write(pakFile, binary.LittleEndian, PakEntry{})
		check(err)
	}

	var files []PakEntry

	for crc32, path := range filePaths {
		data, err := os.ReadFile(path)
		check(err)
		uncompressedSize := len(data)

		var compressed bytes.Buffer
		zlibWriter := zlib.NewWriter(&compressed)

		_, err = zlibWriter.Write(data)
		check(err)
		zlibWriter.Close()

		compressedSize := compressed.Len()
		if compressedSize < uncompressedSize {
			data = compressed.Bytes()[:compressedSize]
		} else {
			compressedSize = uncompressedSize
		}

		offset, err := pakFile.Seek(0, io.SeekCurrent)
		check(err)

		for position := range uint(compressedSize) {
			xorIndex := (uint(offset) + position) % uint(len(XorBuffer))
			data[position] ^= XorBuffer[xorIndex]
		}

		_, err = pakFile.Write(data)
		check(err)

		files = append(files, PakEntry{
			Crc32:            crc32,
			Unknown:          0xFFFFFFFF,
			UncompressedSize: uint32(uncompressedSize),
			CompressedSize:   uint32(compressedSize),
			Offset:           uint32(offset),
		})
	}

	_, err = pakFile.Seek(entryOffset, io.SeekStart)
	check(err)

	err = binary.Write(pakFile, binary.LittleEndian, files)
	check(err)
}
