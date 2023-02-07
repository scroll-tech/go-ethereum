package rcfg

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

func init() {
	// TODO: use conifg; allow hardforks and different networks; check on L2 consensus layer when going decentralized
	l2GasPriceOracleAddr := common.HexToAddress("0x420000000000000000000000000000000000000F")
	L2GasPriceOracleAddress = &l2GasPriceOracleAddr
}

var (
	// L2GasPriceOracleAddress is the address of the L2GasPriceOracle
	// predeploy
	L2GasPriceOracleAddress *common.Address
	// L1BlockContainerAddress is the address of the L1BlockContainer
	// predeploy
	L1BlockContainerAddress *common.Address

	Precision = new(big.Int).SetUint64(1e9)

	OverheadSlot = common.BigToHash(big.NewInt(1))
	ScalarSlot   = common.BigToHash(big.NewInt(2))
)
