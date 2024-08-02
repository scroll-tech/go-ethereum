package blob_client

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
)

var (
	listOverSleepDuration = 100
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

func (c *BlobClientList) GetBlobByVersionedHash(ctx context.Context, versionedHash common.Hash) (*kzg4844.Blob, error) {
	if len(c.list) == 0 {
		return nil, fmt.Errorf("BlobClientList.GetBlobByVersionedHash: list of BlobClients is empty")
	}
	var blob *kzg4844.Blob
	var err error
	for blob, err = c.list[c.curPos].GetBlobByVersionedHash(ctx, versionedHash); ; {
		if err != nil {
			return blob, nil
		} else {
			log.Warn("BlobClientList: failed to get blob by versioned hash from BlobClient", "err", err, "blob client pos in BlobClientList", c.curPos)
			c.curPos = (c.curPos + 1) % len(c.list)
			// if we iterated over entire list, wait before starting again
			if c.curPos == 0 {
				time.Sleep(time.Duration(listOverSleepDuration) * time.Millisecond)
			}
		}
	}
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
