package missing_header_fields

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/log"
)

func TestManagerDownload(t *testing.T) {
	t.Skip("skipping test due to long runtime/downloading file")
	log.Root().SetHandler(log.StdoutHandler)

	// TODO: replace with actual sha256 hash and downloadURL
	sha256 := [32]byte(common.FromHex("0x575858a53b8cdde8d63a2cc1a5b90f1bbf0c2243b292a66a1ab2931d571eb260"))
	downloadURL := "https://ftp.halifax.rwth-aachen.de/ubuntu-releases/24.04/ubuntu-24.04-netboot-amd64.tar.gz"
	filePath := filepath.Join(t.TempDir(), "test_file_path")
	manager := NewManager(context.Background(), filePath, downloadURL, sha256)

	_, _, err := manager.GetMissingHeaderFields(0)
	require.NoError(t, err)

	// Check if the file was downloaded and tmp file was removed
	_, err = os.Stat(filePath)
	require.NoError(t, err)
	_, err = os.Stat(filePath + ".tmp")
	require.Error(t, err)
}

func TestManagerChecksum(t *testing.T) {
	// Checksum doesn't match
	{
		sha256 := [32]byte(common.FromHex("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
		downloadURL := "" // since the file exists we don't need to download it
		filePath := "testdata/missing_headers_1.dedup"
		manager := NewManager(context.Background(), filePath, downloadURL, sha256)

		_, _, err := manager.GetMissingHeaderFields(0)
		require.ErrorContains(t, err, "checksum mismatch")
	}

	// Checksum matches
	{
		sha256 := [32]byte(common.FromHex("0x5dee238e74c350c7116868bfe6c5218d440be3613f47f8c052bd5cef46f4ae04"))
		downloadURL := "" // since the file exists we don't need to download it
		filePath := "testdata/missing_headers_1.dedup"
		manager := NewManager(context.Background(), filePath, downloadURL, sha256)

		difficulty, extra, err := manager.GetMissingHeaderFields(0)
		require.NoError(t, err)
		require.Equal(t, expectedMissingHeaders1[0].difficulty, difficulty)
		require.Equal(t, expectedMissingHeaders1[0].extra, extra)
	}
}
