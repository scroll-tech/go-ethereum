package da_syncer

import "fmt"

// FetcherMode represents the mode of fetcher
type FetcherMode int

const (
	// L1RPC mode fetches DA from L1RPC
	L1RPC FetcherMode = iota
	// Snapshot mode loads DA from snapshot file
	Snapshot
)

func (mode FetcherMode) IsValid() bool {
	return mode >= L1RPC && mode <= Snapshot
}

// String implements the stringer interface.
func (mode FetcherMode) String() string {
	switch mode {
	case L1RPC:
		return "l1rpc"
	case Snapshot:
		return "snapshot"
	default:
		return "unknown"
	}
}

func (mode FetcherMode) MarshalText() ([]byte, error) {
	switch mode {
	case L1RPC:
		return []byte("l1rpc"), nil
	case Snapshot:
		return []byte("snapshot"), nil
	default:
		return nil, fmt.Errorf("unknown sync mode %d", mode)
	}
}

func (mode *FetcherMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "l1rpc":
		*mode = L1RPC
	case "snapshot":
		*mode = Snapshot
	default:
		return fmt.Errorf(`unknown sync mode %q, want "l1rpc" or "snapshot"`, text)
	}
	return nil
}
