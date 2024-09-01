package blob_client

import (
	"context"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
)

type BlobClientList struct {
	list   []BlobClient
	curPos int
}

func NewBlobClientList(blobClients ...BlobClient) *BlobClientList {
	return &BlobClientList{
		list:   blobClients,
		curPos: 0,
	}
}

func (c *BlobClientList) GetBlobByVersionedHashAndBlockTime(ctx context.Context, versionedHash common.Hash, blockTime uint64) (*kzg4844.Blob, error) {
	if len(c.list) == 0 {
		return nil, fmt.Errorf("BlobClientList.GetBlobByVersionedHash: list of BlobClients is empty")
	}

	for i := 0; i < len(c.list); i++ {
		blob, err := c.list[c.curPos].GetBlobByVersionedHashAndBlockTime(ctx, versionedHash, blockTime)
		if err == nil {
			return blob, nil
		}
		c.nextPos()
		// there was an error, try the next blob client in following iteration
		log.Warn("BlobClientList: failed to get blob by versioned hash from BlobClient", "err", err, "blob client pos in BlobClientList", c.curPos)
	}

	// if we iterated over entire list, return a temporary error that will be handled in syncing_pipeline with a backoff and retry
	return nil, serrors.NewTemporaryError(errors.New("BlobClientList.GetBlobByVersionedHash: failed to get blob by versioned hash from all BlobClients"))
}

func (c *BlobClientList) nextPos() {
	c.curPos = (c.curPos + 1) % len(c.list)
}

func (c *BlobClientList) AddBlobClient(blobClient BlobClient) {
	c.list = append(c.list, blobClient)
}

func (c *BlobClientList) RemoveBlobClient(blobClient BlobClient) {
	c.list = append(c.list, blobClient)
	for pos, client := range c.list {
		if client == blobClient {
			c.list = append(c.list[:pos], c.list[pos+1:]...)
			c.curPos %= len(c.list)
			return
		}
	}
}
func (c *BlobClientList) Size() int {
	return len(c.list)
}
