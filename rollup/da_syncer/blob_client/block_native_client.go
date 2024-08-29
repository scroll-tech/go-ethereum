package blob_client

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
)

type BlockNativeClient struct {
	apiEndpoint string
}

func NewBlockNativeClient(apiEndpoint string) *BlockNativeClient {
	return &BlockNativeClient{
		apiEndpoint: apiEndpoint,
	}
}

func (c *BlockNativeClient) GetBlobByVersionedHashAndBlockNumber(ctx context.Context, versionedHash common.Hash, blockNumber uint64) (*kzg4844.Blob, error) {
	// blocknative api docs https://docs.blocknative.com/blocknative-data-archive/blob-archive
	path, err := url.JoinPath(c.apiEndpoint, versionedHash.String())
	if err != nil {
		return nil, fmt.Errorf("failed to join path, err: %w", err)
	}
	resp, err := http.Get(path)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var res ErrorRespBlockNative
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
		}
		return nil, fmt.Errorf("error while fetching blob, message: %s, code: %d, versioned hash: %s", res.Error.Message, res.Error.Code, versionedHash.String())
	}
	var result BlobRespBlockNative
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	blobBytes, err := hex.DecodeString(result.Blob.Data[2:])
	if err != nil {
		return nil, fmt.Errorf("failed to decode data to bytes, err: %w", err)
	}
	if len(blobBytes) != lenBlobBytes {
		return nil, fmt.Errorf("len of blob data is not correct, expected: %d, got: %d", lenBlobBytes, len(blobBytes))
	}
	blob := kzg4844.Blob(blobBytes)
	return &blob, nil
}

type BlobRespBlockNative struct {
	Blob struct {
		Data string `json:"data"`
	} `json:"blob"`
}

type ErrorRespBlockNative struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
