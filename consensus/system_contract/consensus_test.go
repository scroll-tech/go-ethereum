package system_contract

import (
	"math/big"
	"testing"
	"time"

	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/params"
)

func TestSystemContract_CalcTimestamp(t *testing.T) {
	// Use a future timestamp to avoid the "timestamp < time.Now()" condition
	futureTime := uint64(time.Now().Unix()) + 3600 // 1 hour in the future

	tests := []struct {
		name            string
		period          uint64
		blocksPerSecond uint64
		parentTime      uint64
		parentNumber    uint64
		expectedTime    uint64
		description     string
	}{
		{
			name:            "1 second period, 1 block per second - first block",
			period:          1,
			blocksPerSecond: 1,
			parentTime:      futureTime,
			parentNumber:    0,
			expectedTime:    futureTime + 1, // Should increment by 1 second after 1 block
			description:     "First block in period, should increment timestamp",
		},
		{
			name:            "1 second period, 2 blocks per second - first block",
			period:          1,
			blocksPerSecond: 2,
			parentTime:      futureTime,
			parentNumber:    0,
			expectedTime:    futureTime, // Should not increment yet (block 0, need block 1 to complete period)
			description:     "First of two blocks in period, should not increment timestamp yet",
		},
		{
			name:            "1 second period, 2 blocks per second - second block",
			period:          1,
			blocksPerSecond: 2,
			parentTime:      futureTime,
			parentNumber:    1,
			expectedTime:    futureTime + 1, // Should increment by 1 second after 2 blocks
			description:     "Second of two blocks in period, should increment timestamp",
		},
		{
			name:            "2 second period, 1 block per second - first block",
			period:          2,
			blocksPerSecond: 1,
			parentTime:      futureTime,
			parentNumber:    0,
			expectedTime:    futureTime, // Should not increment yet (need 2 blocks for 2-second period)
			description:     "First of two blocks in 2-second period",
		},
		{
			name:            "2 second period, 1 block per second - second block",
			period:          2,
			blocksPerSecond: 1,
			parentTime:      futureTime,
			parentNumber:    1,
			expectedTime:    futureTime + 2, // Should increment by 2 seconds after 2 blocks
			description:     "Second of two blocks in 2-second period, should increment by period",
		},
		{
			name:            "2 second period, 2 blocks per second - fourth block",
			period:          2,
			blocksPerSecond: 2,
			parentTime:      futureTime,
			parentNumber:    3,              // blocks 0,1,2,3 = 4 blocks total for 2-second period
			expectedTime:    futureTime + 2, // Should increment by 2 seconds after 4 blocks
			description:     "Last block in 2-second period with 2 blocks/sec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &params.SystemContractConfig{
				Period:          tt.period,
				BlocksPerSecond: tt.blocksPerSecond,
				RelaxedPeriod:   false, // Ensure we test strict timing
			}

			sc := &SystemContract{
				config: config,
			}

			parent := &types.Header{
				Time:   tt.parentTime,
				Number: new(big.Int).SetUint64(tt.parentNumber),
			}

			result := sc.CalcTimestamp(parent)
			if result != tt.expectedTime {
				t.Errorf("%s: CalcTimestamp() = %d, want %d\nDescription: %s\nConfig: Period=%d, BlocksPerSecond=%d, ParentNumber=%d",
					tt.name, result, tt.expectedTime, tt.description, tt.period, tt.blocksPerSecond, tt.parentNumber)
			}
		})
	}
}

func TestBlockIntervalTiming(t *testing.T) {
	tests := []struct {
		name             string
		period           uint64
		blocksPerSecond  uint64
		expectedInterval time.Duration
		description      string
	}{
		{
			name:             "1 block per second",
			period:           1,
			blocksPerSecond:  1,
			expectedInterval: 1000 * time.Millisecond,
			description:      "Traditional 1 second block time",
		},
		{
			name:             "2 blocks per second",
			period:           1,
			blocksPerSecond:  2,
			expectedInterval: 500 * time.Millisecond,
			description:      "Fast blocks every 500ms",
		},
		{
			name:             "4 blocks per second",
			period:           1,
			blocksPerSecond:  4,
			expectedInterval: 250 * time.Millisecond,
			description:      "Very fast blocks every 250ms",
		},
		{
			name:             "10 blocks per second",
			period:           1,
			blocksPerSecond:  10,
			expectedInterval: 100 * time.Millisecond,
			description:      "Ultra fast blocks every 100ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			periodMs := CalcPeriodMs(tt.blocksPerSecond)
			actualInterval := time.Duration(periodMs) * time.Millisecond

			if actualInterval != tt.expectedInterval {
				t.Errorf("%s: Block interval = %v, want %v\nDescription: %s",
					tt.name, actualInterval, tt.expectedInterval, tt.description)
			}

			// Also verify the math: blocksPerSecond * interval should equal 1 second
			totalTimePerSecond := time.Duration(tt.blocksPerSecond) * actualInterval
			expectedTotalTime := 1 * time.Second

			if totalTimePerSecond != expectedTotalTime {
				t.Errorf("%s: %d blocks * %v interval = %v, want %v",
					tt.name, tt.blocksPerSecond, actualInterval, totalTimePerSecond, expectedTotalTime)
			}
		})
	}
}

func TestComplexTimingScenarios(t *testing.T) {
	// Test a more complex scenario: tracking timestamp progression over multiple blocks
	config := &params.SystemContractConfig{
		Period:          1,
		BlocksPerSecond: 4, // 250ms per block
		RelaxedPeriod:   false,
	}

	sc := &SystemContract{
		config: config,
	}

	baseTime := uint64(time.Now().Unix()) + 3600 // 1 hour in the future
	expectedTimestamps := []uint64{
		baseTime,     // block 0: no increment yet
		baseTime,     // block 1: no increment yet
		baseTime,     // block 2: no increment yet
		baseTime + 1, // block 3: complete period, increment by 1 second
		baseTime + 1, // block 4: start new period
		baseTime + 1, // block 5: continue period
		baseTime + 1, // block 6: continue period
		baseTime + 2, // block 7: complete second period, increment by 1 second
	}

	for i, expectedTime := range expectedTimestamps {
		parent := &types.Header{
			Time:   baseTime,
			Number: new(big.Int).SetUint64(uint64(i)),
		}

		result := sc.CalcTimestamp(parent)
		if result != expectedTime {
			t.Errorf("Block %d: CalcTimestamp() = %d, want %d", i, result, expectedTime)
		}

		// Update baseTime for next iteration to simulate time progression
		baseTime = result
	}
}

func TestDefaultValues(t *testing.T) {
	// Test with default/zero values
	config := &params.SystemContractConfig{
		Period:          0, // Should default to 1
		BlocksPerSecond: 0, // Should default to 1
		RelaxedPeriod:   false,
	}

	sc := &SystemContract{
		config: config,
	}

	futureTime := uint64(time.Now().Unix()) + 3600 // 1 hour in the future
	parent := &types.Header{
		Time:   futureTime,
		Number: new(big.Int).SetUint64(0),
	}

	result := sc.CalcTimestamp(parent)
	// With defaults (period=1, blocksPerSecond=1), first block should increment timestamp
	expected := futureTime + 1

	if result != expected {
		t.Errorf("With default values: CalcTimestamp() = %d, want %d", result, expected)
	}

	// Verify the period calculation
	periodMs := CalcPeriodMs(0) // Should default to 1000ms
	if periodMs != 1000 {
		t.Errorf("Default period calculation: got %d ms, want 1000 ms", periodMs)
	}
}

// TestTimestampIncrementLogic specifically tests the timestamp increment logic
func TestTimestampIncrementLogic(t *testing.T) {
	config := &params.SystemContractConfig{
		Period:          1,
		BlocksPerSecond: 2, // 2 blocks per second
		RelaxedPeriod:   false,
	}

	sc := &SystemContract{
		config: config,
	}

	baseTime := uint64(time.Now().Unix()) + 3600 // 1 hour in the future
	currentTime := baseTime

	// Test for several blocks
	for i := 0; i < 6; i++ {
		parent := &types.Header{
			Time:   currentTime,
			Number: new(big.Int).SetUint64(uint64(i)),
		}

		newTimestamp := sc.CalcTimestamp(parent)
		nextBlockNumber := uint64(i + 1)

		// Calculate expected timestamp
		var expectedTimestamp uint64
		if nextBlockNumber%2 == 0 {
			// This is a period boundary (blocks 2, 4, 6, ...)
			expectedTimestamp = currentTime + 1
		} else {
			// This is within a period (blocks 1, 3, 5, ...)
			expectedTimestamp = currentTime
		}

		if newTimestamp != expectedTimestamp {
			t.Errorf("Block %d: got timestamp %d, want %d", nextBlockNumber, newTimestamp, expectedTimestamp)
		}

		// Update currentTime for next iteration (simulate blockchain progression)
		currentTime = newTimestamp
	}
}
