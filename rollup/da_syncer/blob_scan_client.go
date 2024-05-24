package da_syncer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

const (
	blobScanApiUrl string = "https://api.blobscan.com/blobs/"
	okStatusCode   int    = 200
	lenBlobBytes   int    = 131072
)

type BlobScanClient struct {
	client *http.Client
}

func newBlobScanClient() (*BlobScanClient, error) {
	return &BlobScanClient{
		client: http.DefaultClient,
	}, nil
}

func (c *BlobScanClient) GetBlobByVersionedHash(ctx context.Context, versionedHash common.Hash) (*kzg4844.Blob, error) {
	// some api call
	req, err := http.NewRequestWithContext(ctx, "GET", blobScanApiUrl+versionedHash.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request, err: %v", err)
	}
	req.Header.Set("accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != okStatusCode {
		return nil, fmt.Errorf("response code is not ok, code: %d", resp.StatusCode)
	}
	var result BlobResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %v", err)
	}
	blobBytes, err := hex.DecodeString(result.Data[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode data to bytes, err: %v", err)
	}
	if len(blobBytes) != lenBlobBytes {
		return nil, fmt.Errorf("len of blob data is not correct, expected: %d, got: %d", lenBlobBytes, len(blobBytes))
	}
	blob := kzg4844.Blob(blobBytes)
	return &blob, nil
}

type BlobResp struct {
	Commitment            string `json:"commitment"`
	Proof                 string `json:"proof"`
	Size                  int    `json:"size"`
	VersionedHash         string `json:"versionedHash"`
	Data                  string `json:"data"`
	DataStorageReferences []struct {
		BlobStorage   string `json:"blobStorage"`
		DataReference string `json:"dataReference"`
	} `json:"dataStorageReferences"`
	Transactions []struct {
		Hash  string `json:"hash"`
		Index int    `json:"index"`
		Block struct {
			Number                int    `json:"number"`
			BlobGasUsed           string `json:"blobGasUsed"`
			BlobAsCalldataGasUsed string `json:"blobAsCalldataGasUsed"`
			BlobGasPrice          string `json:"blobGasPrice"`
			ExcessBlobGas         string `json:"excessBlobGas"`
			Hash                  string `json:"hash"`
			Timestamp             string `json:"timestamp"`
			Slot                  int    `json:"slot"`
		} `json:"block"`
		From                  string `json:"from"`
		To                    string `json:"to"`
		MaxFeePerBlobGas      string `json:"maxFeePerBlobGas"`
		BlobAsCalldataGasUsed string `json:"blobAsCalldataGasUsed"`
		Rollup                string `json:"rollup"`
		BlobAsCalldataGasFee  string `json:"blobAsCalldataGasFee"`
		BlobGasBaseFee        string `json:"blobGasBaseFee"`
		BlobGasMaxFee         string `json:"blobGasMaxFee"`
		BlobGasUsed           string `json:"blobGasUsed"`
	} `json:"transactions"`
}
