package withdrawtrie

import (
	"github.com/scroll-tech/go-ethereum/common"
)

// StateDB represents the StateDB interface
// required to get withdraw trie root
type StateDB interface {
	GetState(common.Address, common.Hash) common.Hash
}

func ReadWTRSlot(addr common.Address, state StateDB) common.Hash {
	return common.Hash{}
}
