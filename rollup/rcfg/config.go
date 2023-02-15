package rcfg

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

// UsingScroll is used to enable or disable functionality necessary for scroll rollup.
var UsingScroll bool

// TODO:
// 1. use config
// 2. allow different networks / hardforks
// 3. vefify in consensus layer when decentralizing sequencer

var (
	// L2MessageQueueAddress is the address of the L2MessageQueue
	// predeploy
	// see scroll-tech/scroll/contracts/src/L2/predeploys/L2MessageQueue.sol
	L2MessageQueueAddress = common.HexToAddress("0x5300000000000000000000000000000000000000")
	WithdrawTrieRootSlot  = common.BigToHash(big.NewInt(0))
)
