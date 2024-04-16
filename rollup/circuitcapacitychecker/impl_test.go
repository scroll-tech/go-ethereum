package circuitcapacitychecker

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/types"
)

func BenchmarkApplyTransaction(b *testing.B) {
	data, err := os.ReadFile("block_trace_0xac92e50305b280808a06573888c89cde3556b79d72556c8cdfa97dc6b5695e44.json")
	if err != nil {
		b.Fatalf("Error reading block trace file: %v", err)
	}

	var blockTrace types.BlockTrace
	err = json.Unmarshal(data, &blockTrace)
	if err != nil {
		b.Fatalf("Error unmarshaling block trace JSON: %v", err)
	}

	ccc := NewCircuitCapacityChecker(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ccc.ApplyTransaction(&blockTrace)
		if err != nil {
			b.Fatalf("Error applying transaction: %v", err)
		}
	}
}
