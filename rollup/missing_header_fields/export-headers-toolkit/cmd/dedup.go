package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

// dedupCmd represents the dedup command
var dedupCmd = &cobra.Command{
	Use:   "dedup",
	Short: "Deduplicate the headers file, print unique values and create a new file with the deduplicated headers",
	Long: `Deduplicate the headers file, print unique values and create a new file with the deduplicated headers.

The binary layout of the deduplicated file is as follows:
- 1 byte for the count of unique vanity
- 32 bytes for each unique vanity
- for each header:
  - 1 byte for the index of the vanity in the unique vanity list
  - 1 byte (bitmask, lsb first): 
	- bit 0: 0 if difficulty is 2, 1 if difficulty is 1
    - bit 1: 0 if seal length is 65, 1 if seal length is 85
    - rest: 0
  - 65 or 85 bytes for the seal`,
	Run: func(cmd *cobra.Command, args []string) {
		inputFile, err := cmd.Flags().GetString("input")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}
		outputFile, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}

		seenDifficulty, seenVanity, seenSealLen := runAnalysis(inputFile)
		runDedup(inputFile, outputFile, seenDifficulty, seenVanity, seenSealLen)
		runSHA256(outputFile)
	},
}

func runSHA256(outputFile string) {
	f, err := os.Open(outputFile)
	defer f.Close()
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		log.Fatalf("Error hashing file: %v", err)
	}

	fmt.Printf("Deduplicated headers written to %s with sha256 checksum: %x\n", outputFile, h.Sum(nil))
}

func init() {
	rootCmd.AddCommand(dedupCmd)

	dedupCmd.Flags().String("input", "headers.bin", "headers file")
	dedupCmd.Flags().String("output", "headers-dedup.bin", "deduplicated, binary formatted file")
}

func runAnalysis(inputFile string) (seenDifficulty map[uint64]int, seenVanity map[[32]byte]bool, seenSealLen map[int]int) {
	reader := newHeaderReader(inputFile)
	defer reader.close()

	// track header fields we've seen
	seenDifficulty = make(map[uint64]int)
	seenVanity = make(map[[32]byte]bool)
	seenSealLen = make(map[int]int)

	reader.read(func(header *types.Header) {
		seenDifficulty[header.Difficulty]++
		seenVanity[header.Vanity()] = true
		seenSealLen[header.SealLen()]++
	})

	// Print distinct values and report
	fmt.Println("--------------------------------------------------")
	for diff, count := range seenDifficulty {
		fmt.Printf("Difficulty %d: %d\n", diff, count)
	}

	for vanity := range seenVanity {
		fmt.Printf("Vanity: %x\n", vanity)
	}

	for sealLen, count := range seenSealLen {
		fmt.Printf("SealLen %d bytes: %d\n", sealLen, count)
	}

	fmt.Println("--------------------------------------------------")
	fmt.Printf("Unique values seen in the headers file (last seen block: %d):\n", reader.lastHeader.Number)
	fmt.Printf("Distinct count: Difficulty:%d, Vanity:%d, SealLen:%d\n", len(seenDifficulty), len(seenVanity), len(seenSealLen))
	fmt.Printf("--------------------------------------------------\n\n")

	return seenDifficulty, seenVanity, seenSealLen
}

func runDedup(inputFile, outputFile string, seenDifficulty map[uint64]int, seenVanity map[[32]byte]bool, seenSealLen map[int]int) {
	reader := newHeaderReader(inputFile)
	defer reader.close()

	writer := newMissingHeaderFileWriter(outputFile, seenVanity)
	defer writer.close()

	writer.missingHeaderWriter.writeVanities()

	reader.read(func(header *types.Header) {
		writer.missingHeaderWriter.write(header)
	})
}

type headerReader struct {
	file       *os.File
	reader     *bufio.Reader
	lastHeader *types.Header
}

func newHeaderReader(inputFile string) *headerReader {
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Error opening input file: %v", err)
	}

	h := &headerReader{
		file:   f,
		reader: bufio.NewReader(f),
	}

	return h
}

func (h *headerReader) read(callback func(header *types.Header)) {
	headerSizeBytes := make([]byte, types.HeaderSizeSerialized)

	for {
		_, err := io.ReadFull(h.reader, headerSizeBytes)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading headerSizeBytes: %v", err)
		}
		headerSize := binary.BigEndian.Uint16(headerSizeBytes)

		headerBytes := make([]byte, headerSize)
		_, err = io.ReadFull(h.reader, headerBytes)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error reading headerBytes: %v", err)
		}
		header := new(types.Header).FromBytes(headerBytes)

		// sanity check: make sure headers are in order
		if h.lastHeader != nil && header.Number != h.lastHeader.Number+1 {
			fmt.Println("lastHeader:", h.lastHeader.String())
			log.Fatalf("Missing block: %d, got %d instead", h.lastHeader.Number+1, header.Number)
		}
		h.lastHeader = header

		callback(header)
	}
}

func (h *headerReader) close() {
	h.file.Close()
}
