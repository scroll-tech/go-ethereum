package blob_client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto/kzg4844"
	"github.com/scroll-tech/go-ethereum/rollup/rollup_sync_service"
)

type BeaconNodeClient struct {
	apiEndpoint    string
	l1Client       *rollup_sync_service.L1Client
	genesisTime    uint64
	secondsPerSlot uint64
}

var (
	beaconNodeGenesisEndpoint = "/eth/v1/beacon/genesis"
	beaconNodeSpecEndpoint    = "/eth/v1/config/spec"
	beaconNodeBlobEndpoint    = "/eth/v1/beacon/blob_sidecars"
)

func NewBeaconNodeClient(apiEndpoint string, l1Client *rollup_sync_service.L1Client) (*BeaconNodeClient, error) {
	// get genesis time
	genesisPath := path.Join(apiEndpoint, beaconNodeGenesisEndpoint)
	resp, err := http.Get(genesisPath)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}
	var genesisResp GenesisResp
	err = json.NewDecoder(resp.Body).Decode(&genesisResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	genesisTime, err := strconv.ParseUint(genesisResp.Data.GenesisTime, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode genesis time %s, err: %w", genesisResp.Data.GenesisTime, err)
	}

	// get seconds per slot from spec
	specPath := path.Join(apiEndpoint, beaconNodeSpecEndpoint)
	resp, err = http.Get(specPath)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}
	var specResp SpecResp
	err = json.NewDecoder(resp.Body).Decode(&specResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}
	secondsPerSlot, err := strconv.ParseUint(specResp.Data.SecondsPerSlot, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode seconds per slot %s, err: %w", specResp.Data.SecondsPerSlot, err)
	}
	if secondsPerSlot == 0 {
		return nil, fmt.Errorf("failed to make new BeaconNodeClient, secondsPerSlot is 0")
	}

	return &BeaconNodeClient{
		apiEndpoint:    apiEndpoint,
		l1Client:       l1Client,
		genesisTime:    genesisTime,
		secondsPerSlot: secondsPerSlot,
	}, nil
}

func (c *BeaconNodeClient) GetBlobByVersionedHashAndBlockNumber(ctx context.Context, versionedHash common.Hash, blockNumber uint64) (*kzg4844.Blob, error) {
	// get block timestamp to calculate slot
	header, err := c.l1Client.GetHeaderByNumber(blockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get header by number, err: %w", err)
	}
	slot := (header.Time - c.genesisTime) / c.secondsPerSlot

	// get blob sidecar for slot
	blobSidecarPath := path.Join(c.apiEndpoint, beaconNodeBlobEndpoint, fmt.Sprintf("%d", slot))
	resp, err := http.Get(blobSidecarPath)
	if err != nil {
		return nil, fmt.Errorf("cannot do request, err: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		return nil, fmt.Errorf("beacon node request failed, status: %s, body: %s", resp.Status, bodyStr)
	}
	var blobSidecarResp BlobSidecarResp
	err = json.NewDecoder(resp.Body).Decode(&blobSidecarResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode result into struct, err: %w", err)
	}

	// find blob with desired versionedHash
	for _, blob := range blobSidecarResp.Data {
		// calculate blob hash from commitment and check it with desired
		commitmentBytes, err := hex.DecodeString(blob.KzgCommitment[2:])
		if err != nil {
			return nil, fmt.Errorf("failed to decode data to bytes, err: %w", err)
		}
		if len(commitmentBytes) != lenKzgCommitment {
			return nil, fmt.Errorf("len of kzg commitment is not correct, expected: %d, got: %d", lenKzgCommitment, len(commitmentBytes))
		}
		commitment := kzg4844.Commitment(commitmentBytes)
		blobVersionedHash := kzg4844.CalcBlobHashV1(sha256.New(), &commitment)
		if blobVersionedHash == versionedHash {
			// found desired blob
			blobBytes, err := hex.DecodeString(blob.Blob[2:])
			if err != nil {
				return nil, fmt.Errorf("failed to decode data to bytes, err: %w", err)
			}
			if len(blobBytes) != lenBlobBytes {
				return nil, fmt.Errorf("len of blob data is not correct, expected: %d, got: %d", lenBlobBytes, len(blobBytes))
			}
			blob := kzg4844.Blob(blobBytes)
			return &blob, nil
		}
	}

	return nil, fmt.Errorf("missing blob %v in slot %d, block number %d", versionedHash, slot, blockNumber)
}

type GenesisResp struct {
	Data struct {
		GenesisTime string `json:"genesis_time"`
	} `json:"data"`
}

type SpecResp struct {
	Data struct {
		SecondsPerSlot string `json:"SECONDS_PER_SLOT"`
	} `json:"data"`
}

type BlobSidecarResp struct {
	Data []struct {
		Index             string `json:"index"`
		Blob              string `json:"blob"`
		KzgCommitment     string `json:"kzg_commitment"`
		KzgProof          string `json:"kzg_proof"`
		SignedBlockHeader struct {
			Message struct {
				Slot          string `json:"slot"`
				ProposerIndex string `json:"proposer_index"`
				ParentRoot    string `json:"parent_root"`
				StateRoot     string `json:"state_root"`
				BodyRoot      string `json:"body_root"`
			} `json:"message"`
			Signature string `json:"signature"`
		} `json:"signed_block_header"`
		KzgCommitmentInclusionProof []string `json:"kzg_commitment_inclusion_proof"`
	} `json:"data"`
}
