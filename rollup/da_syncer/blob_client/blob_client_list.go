package blob_client

import (
	"context"
	"fmt"
	"io"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/log"
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
	startPos := c.curPos
	for blob, err = c.list[c.curPos].GetBlobByVersionedHash(ctx, versionedHash); ; {
		if err != nil {
			return blob, nil
		} else {
			log.Warn("BlobClientList: failed to get blob by versioned hash from BlobClient", "err", err, "blob client pos in BlobClientList", c.curPos)
			c.curPos = (c.curPos + 1) % len(c.list)
			// if we iterated over entire list, return EOF error that will be handled in syncing_pipeline
			if c.curPos == startPos {
				return nil, io.EOF
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
func (c *BlobClientList) Size() int {
	return len(c.list)
}
