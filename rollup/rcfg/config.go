package rcfg

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

// UsingScroll is used to enable or disable functionality necessary for the SVM.
var UsingScroll bool

// TODO:
// 1. use config
// 2. allow different networks / hardforks
// 3. vefify in consensus layer when decentralizing sequencer

var (
	// L2GasPriceOracleAddress is the address of the L2GasPriceOracle
	// predeploy
	// see scroll-tech/scroll/contracts/src/L2/predeploys/L2GasPriceOracle.sol
	L2GasPriceOracleAddress = common.HexToAddress("0x420000000000000000000000000000000000000F")
	Precision               = new(big.Int).SetUint64(1e9)
	OverheadSlot            = common.BigToHash(big.NewInt(1))
	ScalarSlot              = common.BigToHash(big.NewInt(2))

	// L1BlockContainerAddress is the address of the L1BlockContainer
	// predeploy
	// see scroll-tech/scroll/contracts/src/L2/predeploys/L1BlockContainer.sol
	L1BlockContainerAddress = common.HexToAddress("0x420000000000000000000000000000000000000F")
	LatestBlockHashSlot     = common.BigToHash(big.NewInt(2))
	MetadataSlot            = common.BigToHash(big.NewInt(4))
	BaseFeeMetadataIndex    = big.NewInt(2)
)
