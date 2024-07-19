package missing_header_fields

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
)

type header struct {
	number     uint64
	difficulty uint64
	extra      []byte
}

var expectedMissingHeaders1 = []header{
	{0, 2, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000001970727fcd1de96f749d4cc9c42cba2c0693dd650acf2f23dcf7d4607ea046dbd901284ae68c3754d6c10ea39d2c30e8dcf682071d0c08c96d5ed1d6d868e347510")},
	{1, 1, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000002f347fafeee461e99075bbcb4ed3bbf3da6094bb3bdec839a78e8bca37e05af9667a54ec59b99e44491857dadbd467d8f3ffb50d5863e21267466930fb832dc6072")},
	{2, 2, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000002f358c9b2c41a94c47029ee61ecf52da080ab1026c384f2608e9b4e6d9355408702c95d30b49e0d34944efd811f06b2d4acd55d6b2792219d63dbc89450484166fd7a935859db38dc0a1927a881ab1b77d0bea1d0c6")},
	{3, 1, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000008c71d25c9c4401f0d0a33b125b4a73d04c87c74f6d0fa17396bd29c5fa10310e3adce56dd1183375e4bf6a89a652f8037ad53c0c178d17c16264979f7012ef05303a5c5f659e9060e268432fe3fe6511e1936126fbc")},
	{4, 2, common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000017c0aebb5813ac87c25b0420add7067a5c4105307b461831922af9e7bbfa828efc053bb75611d660d9002776c3e7d1c1c7aa2f3265214d1831bdf1d4b45c7d32d0a")},
	{5, 2, common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000018a706ae7ae04beab64a74c2d9c85bfb0597c3fdf59033ad758f19859aaa6317b6c2efe887d13f05fee70f7dacccf7dad39d3d62178b5901372ceb6d94da2f1a47b")},
	{6, 2, common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000015f14949a5eaf4e56194b84435397c39450e397bbe2708065722cc74312b2d7f309330ae752e06f81a958b0cb3c15a07d17ed9f907d533827f593fbc272e5246438")},
	{7, 2, common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000015c340fb761e273aeae5b8dc5d12b3dce95069ddb8d8dfdd354e32f1c7e8c590fbed5ff520968a74648132dddb1bf31e215a5203af77143fb7170fe813cc1617c4c")},
	{8, 2, common.FromHex("0x00000000000000000000000000000000000000000000000000000000000000013fc20fe02529338255b6cfb595bf4c959bcea92d6af2d78f411afec367e9c2155b52177633c2a9e94601746b48abe44adcb514624c59e67b0eaf8787915c1d74bc")},
	{9, 2, common.FromHex("0x000000000000000000000000000000000000000000000000000000000000000183346f100df02e19f5aaf907d6eff4272b976b683bbda82f50524bff099e3ac57ccaf28f7ee8ce75f7bfde8b5af88f51ecd4ca48e6eca56cf34cc43869839563cd")},
	{10, 2, common.FromHex("0x000000000000000000000000000000000000000000000000000000000000000150154d779ea6f3a74d07cd572b5c951f705b269d72a30b913369a302d4d16fb4f4b27fad875761ce2b5fff09ab3f6e3316cec19c679f2e0a161cf32ffaf5330810")},
	{11, 2, common.FromHex("0x000000000000000000000000000000000000000000000000000000000000000194590bbf6f291ad4e4aeaf2cf7fc6712824f586bcb070e0964220ee3864dc679bf1789afca4816b0e68a40b05ca7bfc533a02894df78e6f4c2c77d71e0a7e3da30")},
	{12, 2, common.FromHex("0x0000000000000000000000000000000000000000000000000000000000000001c7c1c37841da26fd9a2fa796b4972faab58de1b14a255b9b6331342bf511351bebaf5ceba901257a6341bf29d880fc2dba3d41b700f42295f50ea9b28f8ea4bd53")},
}

func TestReader_Read(t *testing.T) {
	expectedVanities := map[int][32]byte{
		0: {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		1: {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
		2: {0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08},
	}

	reader, err := NewReader("testdata/missing_headers_1.dedup")
	require.NoError(t, err)

	require.Len(t, reader.sortedVanities, len(expectedVanities))
	for i, expectedVanity := range expectedVanities {
		require.Equal(t, expectedVanity, reader.sortedVanities[i])
	}

	readAndAssertHeader(t, reader, expectedMissingHeaders1, 0)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 0)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 1)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 6)

	// we don't allow reading previous headers
	_, _, err = reader.Read(5)
	require.Error(t, err)

	readAndAssertHeader(t, reader, expectedMissingHeaders1, 8)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 8)

	// we don't allow reading previous headers
	_, _, err = reader.Read(5)
	require.Error(t, err)

	// we don't allow reading previous headers
	_, _, err = reader.Read(6)
	require.Error(t, err)

	readAndAssertHeader(t, reader, expectedMissingHeaders1, 9)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 10)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 11)
	readAndAssertHeader(t, reader, expectedMissingHeaders1, 12)

	// no data anymore
	_, _, err = reader.Read(13)
	require.Error(t, err)
}

func readAndAssertHeader(t *testing.T, reader *Reader, expectedHeaders []header, headerNum uint64) {
	difficulty, extra, err := reader.Read(headerNum)
	require.NoError(t, err)
	require.Equalf(t, expectedHeaders[headerNum].difficulty, difficulty, "expected difficulty %d, got %d", expectedHeaders[headerNum].difficulty, difficulty)
	require.Equal(t, expectedHeaders[headerNum].extra, extra)
}
