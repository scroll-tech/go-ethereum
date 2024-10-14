package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

// TODO: instead of duplicating this file, missing_header_fields.Reader should be used in toolkit

type missingHeader struct {
	headerNum  uint64
	difficulty uint64
	extraData  []byte
}

type Reader struct {
	file           *os.File
	reader         *bufio.Reader
	sortedVanities map[int][32]byte
	lastReadHeader *missingHeader
}

func NewReader(filePath string) (*Reader, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}

	r := &Reader{
		file:   f,
		reader: bufio.NewReader(f),
	}

	// read the count of unique vanities
	vanityCount, err := r.reader.ReadByte()
	if err != nil {
		return nil, err
	}

	// read the unique vanities
	r.sortedVanities = make(map[int][32]byte)
	for i := uint8(0); i < vanityCount; i++ {
		var vanity [32]byte
		if _, err = r.reader.Read(vanity[:]); err != nil {
			return nil, err
		}
		r.sortedVanities[int(i)] = vanity
	}

	return r, nil
}

func (r *Reader) Read(headerNum uint64) (difficulty uint64, extraData []byte, err error) {
	if r.lastReadHeader == nil {
		if _, _, err = r.ReadNext(); err != nil {
			return 0, nil, err
		}
	}

	if headerNum > r.lastReadHeader.headerNum {
		// skip the headers until the requested header number
		for i := r.lastReadHeader.headerNum; i < headerNum; i++ {
			if _, _, err = r.ReadNext(); err != nil {
				return 0, nil, err
			}
		}
	}

	if headerNum == r.lastReadHeader.headerNum {
		return r.lastReadHeader.difficulty, r.lastReadHeader.extraData, nil
	}

	// headerNum < r.lastReadHeader.headerNum is not supported
	return 0, nil, fmt.Errorf("requested header %d below last read header number %d", headerNum, r.lastReadHeader.headerNum)
}

func (r *Reader) ReadNext() (difficulty uint64, extraData []byte, err error) {
	// read the bitmask
	bitmaskByte, err := r.reader.ReadByte()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read bitmask: %v", err)
	}

	bits := newBitMaskFromByte(bitmaskByte)

	seal := make([]byte, bits.sealLen())

	if _, err = io.ReadFull(r.reader, seal); err != nil {
		return 0, nil, fmt.Errorf("failed to read seal: %v", err)
	}

	// construct the extraData field
	vanity := r.sortedVanities[bits.vanityIndex()]
	var b bytes.Buffer
	b.Write(vanity[:])
	b.Write(seal)

	// we don't have the header number, so we'll just increment the last read header number
	// we assume that the headers are written in order, starting from 0
	if r.lastReadHeader == nil {
		r.lastReadHeader = &missingHeader{
			headerNum:  0,
			difficulty: uint64(bits.difficulty()),
			extraData:  b.Bytes(),
		}
	} else {
		r.lastReadHeader.headerNum++
		r.lastReadHeader.difficulty = uint64(bits.difficulty())
		r.lastReadHeader.extraData = b.Bytes()
	}

	return difficulty, b.Bytes(), nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}
