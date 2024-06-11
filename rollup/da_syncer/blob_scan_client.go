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
	okStatusCode int = 200
	lenBlobBytes int = 131072
)

type BlobScanClient struct {
	client              *http.Client
	blobScanApiEndpoint string
}

func newBlobScanClient(blobScanApiEndpoint string) (*BlobScanClient, error) {
	return &BlobScanClient{
		client:              http.DefaultClient,
		blobScanApiEndpoint: blobScanApiEndpoint,
	}, nil
}

func (c *BlobScanClient) GetBlobByVersionedHash(ctx context.Context, versionedHash common.Hash) (*kzg4844.Blob, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.blobScanApiEndpoint+versionedHash.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create request, err: %w", err)
	}
	req.Header.Set("accept", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != okStatusCode {
		if resp.StatusCode == 404 {
			return nil, fmt.Errorf("no blob with versioned hash : %s", versionedHash.String())
		}
		var res ErrorResp
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
		}
		return nil, fmt.Errorf("error while fetching blob, message: %s, code: %s, versioned hash: %s", res.Message, res.Code, versionedHash.String())
	}
	var result BlobResp
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	blobBytes, err := hex.DecodeString(result.Data[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode data to bytes, err: %w", err)
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

type ErrorResp struct {
	Message string `json:"message"`
	Code    string `json:"code"`
	Issues  []struct {
		Message string `json:"message"`
	} `json:"issues"`
}
