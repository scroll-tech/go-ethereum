package cmd

import (
	"bufio"
	"container/heap"
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/spf13/cobra"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch missing block header fields from a running Scroll L2 node via RPC and store in a file",
	Long: `Fetch allows to retrieve the missing block header fields from a running Scroll L2 node via RPC.
It produces a binary file and optionally a human readable csv file with the missing fields.`,
	Run: func(cmd *cobra.Command, args []string) {
		rpc, err := cmd.Flags().GetString("rpc")
		if err != nil {
			log.Fatalf("Error reading rpc flag: %v", err)
		}
		client, err := ethclient.Dial(rpc)
		if err != nil {
			log.Fatalf("Error connecting to RPC: %v", err)
		}
		startBlockNum, err := cmd.Flags().GetUint64("start")
		if err != nil {
			log.Fatalf("Error reading start flag: %v", err)
		}
		endBlockNum, err := cmd.Flags().GetUint64("end")
		if err != nil {
			log.Fatalf("Error reading end flag: %v", err)
		}
		batchSize, err := cmd.Flags().GetUint64("batch")
		if err != nil {
			log.Fatalf("Error reading batch flag: %v", err)
		}
		maxParallelGoroutines, err := cmd.Flags().GetInt("parallelism")
		if err != nil {
			log.Fatalf("Error reading parallelism flag: %v", err)
		}
		outputFile, err := cmd.Flags().GetString("output")
		if err != nil {
			log.Fatalf("Error reading output flag: %v", err)
		}
		humanReadableOutputFile, err := cmd.Flags().GetString("humanOutput")
		if err != nil {
			log.Fatalf("Error reading humanReadable flag: %v", err)
		}

		runFetch(client, startBlockNum, endBlockNum, batchSize, maxParallelGoroutines, outputFile, humanReadableOutputFile)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)

	fetchCmd.Flags().String("rpc", "http://localhost:8545", "RPC URL")
	fetchCmd.Flags().Uint64("start", 0, "start block number")
	fetchCmd.Flags().Uint64("end", 1000, "end block number")
	fetchCmd.Flags().Uint64("batch", 100, "batch size")
	fetchCmd.Flags().Int("parallelism", 10, "max parallel goroutines each working on batch size blocks")
	fetchCmd.Flags().String("output", "headers.bin", "output file")
	fetchCmd.Flags().String("humanOutput", "", "additionally produce human readable csv file")
}

func headerByNumberWithRetry(client *ethclient.Client, blockNum uint64, maxRetries int) (*types.Header, error) {
	var innerErr error
	for i := 0; i < maxRetries; i++ {
		header, err := client.HeaderByNumber(context.Background(), big.NewInt(int64(blockNum)))
		if err == nil {
			return &types.Header{
				Number:     header.Number.Uint64(),
				Difficulty: header.Difficulty.Uint64(),
				ExtraData:  header.Extra,
			}, nil
		}

		innerErr = err // save the last error to return it if all retries fail

		// Wait before retrying
		time.Sleep(time.Duration(i*200) * time.Millisecond)
		log.Printf("Retrying header fetch for block %d, retry %d, error %v", blockNum, i+1, err)
	}

	return nil, fmt.Errorf("error fetching header for block %d: %v", blockNum, innerErr)
}

func fetchHeaders(client *ethclient.Client, start, end uint64, headersChan chan<- *types.Header) {
	for i := start; i <= end; i++ {
		header, err := headerByNumberWithRetry(client, i, 15)
		if err != nil {
			log.Fatalf("Error fetching header %d: %v", i, err)
		}

		headersChan <- header
	}
}

func writeHeadersToFile(outputFile string, humanReadableOutputFile string, startBlockNum uint64, headersChan <-chan *types.Header) {
	writer := newFilesWriter(outputFile, humanReadableOutputFile)
	defer writer.close()

	headerHeap := &types.HeaderHeap{}
	heap.Init(headerHeap)

	nextHeaderNum := startBlockNum

	// receive all headers and write them in order by using a sorted heap
	for header := range headersChan {
		heap.Push(headerHeap, header)

		// write all headers that are in order
		for headerHeap.Len() > 0 && (*headerHeap)[0].Number == nextHeaderNum {
			nextHeaderNum++
			sortedHeader := heap.Pop(headerHeap).(*types.Header)
			writer.write(sortedHeader)
		}
	}

	fmt.Println("Finished writing headers to file, last block number:", nextHeaderNum-1)
}

func runFetch(client *ethclient.Client, startBlockNum uint64, endBlockNum uint64, batchSize uint64, maxGoroutines int, outputFile string, humanReadableOutputFile string) {
	headersChan := make(chan *types.Header, maxGoroutines*int(batchSize))
	tasks := make(chan task, maxGoroutines)

	var wgConsumer sync.WaitGroup
	// start consumer goroutine to sort and write headers to file
	go func() {
		wgConsumer.Add(1)
		writeHeadersToFile(outputFile, humanReadableOutputFile, startBlockNum, headersChan)
		wgConsumer.Done()
	}()

	var wgProducers sync.WaitGroup
	// start producer goroutines to fetch headers
	for i := 0; i < maxGoroutines; i++ {
		wgProducers.Add(1)
		go func() {
			for {
				t, ok := <-tasks
				if !ok {
					break
				}
				fetchHeaders(client, t.start, t.end, headersChan)
			}
			wgProducers.Done()
		}()
	}

	// create tasks/work packages for producer goroutines
	for start := startBlockNum; start <= endBlockNum; start += batchSize {
		end := start + batchSize - 1
		if end > endBlockNum {
			end = endBlockNum
		}
		fmt.Println("Fetching headers for blocks", start, "to", end)

		tasks <- task{start, end}
	}

	close(tasks)
	wgProducers.Wait()
	close(headersChan)
	wgConsumer.Wait()
}

type task struct {
	start uint64
	end   uint64
}

// filesWriter is a helper struct to write headers to binary and human-readable csv files at the same time.
type filesWriter struct {
	binaryFile   *os.File
	binaryWriter *bufio.Writer

	humanReadable bool
	csvFile       *os.File
	csvWriter     *bufio.Writer
}

func newFilesWriter(outputFile string, humanReadableOutputFile string) *filesWriter {
	binaryFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("Error creating binary file: %v", err)
	}

	f := &filesWriter{
		binaryFile:    binaryFile,
		binaryWriter:  bufio.NewWriter(binaryFile),
		humanReadable: humanReadableOutputFile != "",
	}

	if humanReadableOutputFile != "" {
		csvFile, err := os.Create(humanReadableOutputFile)
		if err != nil {
			log.Fatalf("Error creating human readable file: %v", err)
		}
		f.csvFile = csvFile
		f.csvWriter = bufio.NewWriter(csvFile)
	}

	return f
}

func (f *filesWriter) close() {
	if err := f.binaryWriter.Flush(); err != nil {
		log.Fatalf("Error flushing binary buffer: %v", err)
	}
	if f.humanReadable {
		if err := f.csvWriter.Flush(); err != nil {
			log.Fatalf("Error flushing csv buffer: %v", err)
		}
	}

	f.binaryFile.Close()
	if f.humanReadable {
		f.csvFile.Close()
	}
}
func (f *filesWriter) write(header *types.Header) {
	bytes, err := header.Bytes()
	if err != nil {
		log.Fatalf("Error converting header to bytes: %v", err)
	}

	if _, err = f.binaryWriter.Write(bytes); err != nil {
		log.Fatalf("Error writing to binary file: %v", err)
	}

	if f.humanReadable {
		if _, err = f.csvWriter.WriteString(header.String()); err != nil {
			log.Fatalf("Error writing to human readable file: %v", err)
		}
	}
}
