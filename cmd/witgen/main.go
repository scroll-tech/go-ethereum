package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/trie/zkproof"
)

func main() {

	inputName := "trace.json"
	outputName := "witness.json"

	for i, arg := range os.Args[1:] {
		switch i {
		case 0:
			inputName = arg
		case 1:
			outputName = arg
		}
	}

	fIn, err := os.Open(inputName)
	if err != nil {
		panic(err)
	}
	defer fIn.Close()

	fOut, err := os.Create(outputName)
	if err != nil {
		panic(err)
	}
	defer fOut.Close()

	bt, err := io.ReadAll(fIn)
	if err != nil {
		panic(err)
	}
	readObj := new(types.BlockResult)

	err = json.Unmarshal(bt, readObj)
	if err != nil {
		panic(err)
	}

	outArr, err := zkproof.HandleBlockResult(readObj)
	if err != nil {
		panic(err)
	}

	bt, err = json.Marshal(outArr)
	if err != nil {
		panic(err)
	}

	_, err = fOut.Write(bt)
	if err != nil {
		panic(err)
	}
}
