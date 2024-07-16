package cmd

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"sort"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

type missingHeaderFileWriter struct {
	file   *os.File
	writer *bufio.Writer

	missingHeaderWriter *missingHeaderWriter
}

func newMissingHeaderFileWriter(filename string, seenVanity map[[32]byte]bool) *missingHeaderFileWriter {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Error creating file: %v", err)
	}

	writer := bufio.NewWriter(file)
	return &missingHeaderFileWriter{
		file:                file,
		writer:              writer,
		missingHeaderWriter: newMissingHeaderWriter(writer, seenVanity),
	}
}

func (m *missingHeaderFileWriter) close() {
	if err := m.writer.Flush(); err != nil {
		log.Fatalf("Error flushing writer: %v", err)
	}
	if err := m.file.Close(); err != nil {
		log.Fatalf("Error closing file: %v", err)
	}
}

type missingHeaderWriter struct {
	writer io.Writer

	sortedVanities    [][32]byte
	sortedVanitiesMap map[[32]byte]int
	seenDifficulty    map[uint64]int
	seenSealLen       map[int]int
}

func newMissingHeaderWriter(writer io.Writer, seenVanity map[[32]byte]bool) *missingHeaderWriter {
	// sort the vanities and assign an index to each so that we can write the index of the vanity in the header
	sortedVanities := make([][32]byte, 0, len(seenVanity))
	for vanity := range seenVanity {
		sortedVanities = append(sortedVanities, vanity)
	}
	sort.Slice(sortedVanities, func(i, j int) bool {
		return bytes.Compare(sortedVanities[i][:], sortedVanities[j][:]) < 0
	})
	sortedVanitiesMap := make(map[[32]byte]int)
	for i, vanity := range sortedVanities {
		sortedVanitiesMap[vanity] = i
	}

	return &missingHeaderWriter{
		writer:            writer,
		sortedVanities:    sortedVanities,
		sortedVanitiesMap: sortedVanitiesMap,
	}
}

func (m *missingHeaderWriter) writeVanities() {
	// write the count of unique vanities
	if _, err := m.writer.Write([]byte{uint8(len(m.sortedVanitiesMap))}); err != nil {
		log.Fatalf("Error writing unique vanity count: %v", err)
	}

	// write the unique vanities
	for _, vanity := range m.sortedVanities {
		if _, err := m.writer.Write(vanity[:]); err != nil {
			log.Fatalf("Error writing vanity: %v", err)
		}
	}
}

func (m *missingHeaderWriter) write(header *types.Header) {
	// 1. write the index of the vanity in the unique vanity list
	if _, err := m.writer.Write([]byte{uint8(m.sortedVanitiesMap[header.Vanity()])}); err != nil {
		log.Fatalf("Error writing vanity index: %v", err)
	}

	// 2. write the bitmask
	// - bit 0: 0 if difficulty is 2, 1 if difficulty is 1
	// - bit 1: 0 if seal length is 65, 1 if seal length is 85
	// - rest: 0
	bitmask := uint8(0)
	if header.Difficulty == 1 {
		bitmask |= 1 << 0
	}
	if header.SealLen() == 85 {
		bitmask |= 1 << 1
	}

	if _, err := m.writer.Write([]byte{bitmask}); err != nil {
		log.Fatalf("Error writing bitmask: %v", err)
	}

	if _, err := m.writer.Write(header.Seal()); err != nil {
		log.Fatalf("Error writing seal: %v", err)
	}
}
