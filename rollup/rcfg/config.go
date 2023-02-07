package rcfg

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

// UsingSVM is used to enable or disable functionality necessary for the SVM.
var UsingSVM bool

var (
	// L2GasPriceOracleAddress is the address of the L2GasPriceOracle
	// predeploy
	L2GasPriceOracleAddress = common.HexToAddress("0x420000000000000000000000000000000000000F")
	Precision               = new(big.Int).SetUint64(1e9)
	OverheadSlot            = common.BigToHash(big.NewInt(1))
	ScalarSlot              = common.BigToHash(big.NewInt(2))

	// L1BlockContainerAddress is the address of the L1BlockContainer
	// predeploy
	L1BlockContainerAddress = common.HexToAddress("0x420000000000000000000000000000000000000F")
	LatestBlockHashSlot     = common.BigToHash(big.NewInt(2))
	MetadataSlot            = common.BigToHash(big.NewInt(4))
	BaseFeeMetadataIndex    = big.NewInt(2)
)
