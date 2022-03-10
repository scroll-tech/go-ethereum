package trie

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types/smt"

	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/trie/db"
)

const HashEmpty = 0
const HashLeaf = 1
const HashMid = 2
const HashLeafExt = 3
const HashLeafExtFinal = 4

type Row struct {
	IsFirst bool
	Sib     *smt.Hash
	Path    string // for debug, switch to bigint later
	//PathAcc     *big.Int
	OldHashType int
	OldHash     *smt.Hash
	OldValue    *smt.Hash
	NewHashType int
	NewHash     *smt.Hash
	NewValue    *smt.Hash
	Key         *smt.Hash
}

// TODO: check root between operations
// TODO: check key vs path_acc

const numLevels = 4

func formatPathFromBool(b bool) string {
	if b {
		return "1"
	} else {
		return "0"
	}
}

func formatPathFromBools(bs []bool) string {
	result := ""
	for _, b := range bs {
		if b {
			result += "1"
		} else {
			result += "0"
		}
	}
	return result
}

func littleEndianIntBitsToBigInt(bools ...int64) *big.Int {
	// little endian
	result := big.NewInt(0)
	var base int64 = 1
	for _, b := range bools {
		result.Add(result, big.NewInt(b*base))
		base *= 2
	}
	return result
}

func isZero(b *big.Int) bool {
	return b.Cmp(big.NewInt(0)) == 0
}

// there are 3 types of insertion
// case1: overwrite 0 (include genesis: insert into empty root)
// case2: push down 1 level
// case3: push down more than 1 level
func proofToRows(p *CircomProcessorProof) ([]Row, error) {
	rows := make([]Row, 0)
	if p.Fnc == 3 || p.Fnc == 0 {
		return nil, fmt.Errorf("invalid p.Fnc %v", p.Fnc)
	}

	fullPath := getPath(numLevels, p.NewKey[:])
	fmt.Println("full path of new key", fullPath)
	fullOldPath := getPath(numLevels, p.OldKey[:])

	// just mock for test...
	newLeafKey, err := NewNodeLeaf(p.NewKey, p.NewValue, &byte32Zero, byte32Zero[:]).Key()
	if err != nil {
		return nil, err
	}
	oldLeafKey, err := NewNodeLeaf(p.OldKey, p.OldKey, &byte32Zero, byte32Zero[:]).Key()
	if err != nil {
		return nil, err
	}
	// how many HashMid rows
	midNum := 0
	for i, s := range p.Siblings {
		if isZero(s.BigInt()) {
			midNum = i
			break
		}
	}
	for i := 0; i < midNum; i++ {
		row := Row{
			IsFirst:     false,
			Sib:         p.Siblings[i],
			Path:        formatPathFromBool(fullPath[i]),
			Key:         p.NewKey,
			OldHashType: HashMid,
			OldHash:     &smt.HashZero, // place holder
			OldValue:    &smt.HashZero, // place holder
			NewHashType: HashMid,
			NewHash:     &smt.HashZero, // place holder
			NewValue:    &smt.HashZero, // place holder
		}
		rows = append(rows, row)
	}

	leafHeight := midNum
	if p.Fnc == 2 && !isZero(p.OldKey.BigInt()) {
		// push down
		// push down at least one level
		for i := 0; i < len(fullPath); i++ {
			if fullPath[i] != fullOldPath[i] {
				leafHeight = i + 1 // root is of depth 0
				break
			}
		}
		if leafHeight <= midNum {
			panic(fmt.Errorf("wtf %d %d", leafHeight, midNum))
		}
		leafSib := oldLeafKey
		for i := midNum; i < leafHeight-1; i++ {
			// make leafExt
			row := Row{
				IsFirst:     false,
				Sib:         &smt.HashZero,
				Path:        formatPathFromBool(fullPath[i]),
				Key:         p.NewKey,
				OldHashType: HashLeafExt,
				OldHash:     leafSib,
				OldValue:    leafSib,
				NewHashType: HashMid,
				NewHash:     &smt.HashZero, // place holder
				NewValue:    &smt.HashZero, // place holder
			}
			rows = append(rows, row)
		}
		// make leafExtFinal
		row := Row{
			IsFirst:     false,
			Sib:         leafSib,
			Path:        formatPathFromBool(fullPath[leafHeight]),
			Key:         p.NewKey,
			OldHashType: HashLeafExtFinal,
			OldHash:     leafSib,       // place holder
			OldValue:    &smt.HashZero, // place holder
			NewHashType: HashMid,
			NewHash:     &smt.HashZero,
			NewValue:    newLeafKey,
		}
		if fullPath[leafHeight] {
			h, err := NewNodeMiddle(newLeafKey, leafSib).Key()
			if err != nil {
				return nil, err
			}
			row.NewHash = h
		} else {
			h, err := NewNodeMiddle(leafSib, newLeafKey).Key()
			if err != nil {
				return nil, err
			}
			row.NewHash = h
		}
		rows = append(rows, row)
		row = Row{
			IsFirst:     false,
			Sib:         &smt.HashZero,
			Path:        formatPathFromBools(fullPath[leafHeight:]),
			Key:         p.NewKey,
			OldHashType: HashEmpty,
			OldHash:     &smt.HashZero, // place holder
			OldValue:    &smt.HashZero, // place holder
			NewHashType: HashLeaf,
			NewHash:     newLeafKey,
			NewValue:    p.NewValue,
		}
		rows = append(rows, row)
	} else if (p.Fnc == 2 && isZero(p.OldKey.BigInt())) || p.Fnc == 1 {
		// the leaf
		row := Row{
			IsFirst:     false,
			Sib:         &smt.HashZero,
			Path:        formatPathFromBools(fullPath[midNum:]),
			Key:         p.NewKey,
			OldHashType: HashEmpty,
			OldHash:     &smt.HashZero, // place holder
			OldValue:    &smt.HashZero, // place holder
			NewHashType: HashLeaf,
			NewHash:     newLeafKey,
			NewValue:    p.NewValue,
		}
		if p.Fnc == 1 {
			// update
			row.OldHashType = HashLeaf
			row.OldValue = p.OldValue
			row.OldHash = oldLeafKey
		}
		rows = append(rows, row)
	}
	// reconstruct mid nodes
	fmt.Printf("midNum %v leafHeight %v", midNum, leafHeight)
	for i := leafHeight - 1; i >= 0; i-- {
		rows[i].OldValue = rows[i+1].OldHash
		rows[i].NewValue = rows[i+1].NewHash
		if rows[i].OldHashType == HashMid {
			if rows[i].Path == "0" {
				h, err := NewNodeMiddle(rows[i].OldValue, rows[i].Sib).Key()
				if err != nil {
					return nil, err
				}
				rows[i].OldHash = h
			} else {
				h, err := NewNodeMiddle(rows[i].Sib, rows[i].OldValue).Key()
				if err != nil {
					return nil, err
				}
				rows[i].OldHash = h
			}
		}
		if rows[i].NewHashType == HashMid {
			if (rows[i].Path) == "0" {
				//leftChild =
				// insert left
				h, err := NewNodeMiddle(rows[i].NewValue, rows[i].Sib).Key()
				if err != nil {
					return nil, err
				}
				rows[i].NewHash = h
			} else {
				// insert right
				h, err := NewNodeMiddle(rows[i].Sib, rows[i].NewValue).Key()
				if err != nil {
					return nil, err
				}
				rows[i].NewHash = h
			}
		}
	}
	rows[0].IsFirst = true
	return rows, nil
}

// notes: swap endians(little endian bits) to make the tree full utilized
func generateTestData() error {
	tree, err := NewMerkleTree(db.NewEthKVStorage(memorydb.New()), numLevels)
	if err != nil {
		return err
	}
	/*
	   step1: insert    20 at 0100(2) (overrite empty)
	   step2: insert    21 at 1000(1) (push down one level)
	   step3: insert    22 at 1010(5) (push down >1 level)
	   step4: update to 23 at 1010(5)
	   step5: insert    24 at 1111(15) (overrite empty)
	*/

	p1, err := tree.AddAndGetCircomProof(littleEndianIntBitsToBigInt(0, 1, 0, 0), big.NewInt(20))
	if err != nil {
		return err
	}
	fmt.Printf("p1 %+v\n", p1)
	p2, err := tree.AddAndGetCircomProof(littleEndianIntBitsToBigInt(1, 0, 0, 0), big.NewInt(21))
	if err != nil {
		return err
	}
	fmt.Printf("p2 %+v\n", p2)
	p3, err := tree.AddAndGetCircomProof(littleEndianIntBitsToBigInt(1, 0, 1, 0), big.NewInt(22))
	if err != nil {
		return err
	}
	fmt.Printf("p3 %+v\n", p3)
	p4, err := tree.Update(littleEndianIntBitsToBigInt(1, 0, 1, 0), big.NewInt(23), &byte32Zero, byte32Zero[:])
	if err != nil {
		return err
	}
	fmt.Printf("p4 %+v\n", p4)
	p5, err := tree.AddAndGetCircomProof(littleEndianIntBitsToBigInt(1, 1, 1, 1), big.NewInt(24))
	if err != nil {
		return err
	}
	fmt.Printf("p5 %+v\n", p5)

	var rows []Row
	var proofRows []Row
	proofRows, err = proofToRows(p1)
	fmt.Printf("rows:\n%+v\n", proofRows)
	if err != nil {
		return err
	}
	rows = append(rows, proofRows...)
	proofRows, err = proofToRows(p2)
	fmt.Printf("rows:\n%+v\n", proofRows)
	if err != nil {
		return err
	}
	rows = append(rows, proofRows...)
	proofRows, err = proofToRows(p3)
	fmt.Printf("rows:\n%+v\n", proofRows)
	if err != nil {
		return err
	}
	rows = append(rows, proofRows...)

	proofRows, err = proofToRows(p4)
	fmt.Printf("rows:\n%+v\n", proofRows)
	if err != nil {
		return err
	}
	rows = append(rows, proofRows...)
	proofRows, err = proofToRows(p5)
	fmt.Printf("rows:\n%+v\n", proofRows)
	if err != nil {
		return err
	}
	// rows = append(rows, proofRows...)

	// TODO: check all the constraints of rows
	return nil
}

func TestHalo2Rows(t *testing.T) {
	err := generateTestData()
	if err != nil {
		panic(err)
	}
}
