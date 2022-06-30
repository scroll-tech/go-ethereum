package zkproof

import (
	"github.com/scroll-tech/go-ethereum/core/types"
)

type opIterator interface {
	next() *types.AccountWrapper
}

type opOrderer interface {
	absorb(*types.AccountWrapper)
	end_absorb() opIterator
}

type iterateOp []*types.AccountWrapper

func (ops *iterateOp) next() *types.AccountWrapper {

	sl := *ops

	if len(sl) == 0 {
		return nil
	}

	*ops = sl[1:]
	return sl[0]
}

type simpleOrderer struct {
	savedOp []*types.AccountWrapper
}

func (od *simpleOrderer) absorb(st *types.AccountWrapper) {
	od.savedOp = append(od.savedOp, st)
}

func (od *simpleOrderer) end_absorb() opIterator {
	ret := iterateOp(od.savedOp)
	return &ret
}
