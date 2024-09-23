# l2geth testsuite

The testsuite simulates a L1 backend with the necessary deployed rollup contracts as well as an L2 backend. It enables complete control over L1 and L2 interactions and thus allows to test L2 behavior based on any L1 circumstances (e.g. reorgs).


## Architecture
### Testsuite
The testsuite is the component that glues L1 and L2 together. It initializes both chains, deploys the contracts and sends the L2 genesis batch. Additionally, it provides convenience functions for asserting events and L1 and L2 states.

### L1
L1 is a fully-functioning, simulated L1 using upstream geth's `simulated.Backend` that has the following Scroll contracts deployed and provides convenience functions to interact with them: 
- `ScrollChainMockFinalize`: This is the main rollup contract with mocked finalization. 
- `L1MessageQueue`: This contract is needed to simulate L1->L2 message passing and `L1MessageTx` on L2.
- `L2GasPriceOracle`: This contract is needed as a dependency of `L1MessageQueue` contract, as the L2 gas price is read on L1.

### L2
L2 is a fully-functioning, simulated L2 using l2geth's `backends.SimulatedBackend`. It provides functions to 
- `SendDynamicFeeTransaction`: sends a dynamic fee transaction.
- `CommitBlock`: `commit` here means sealing the block in the simulated backend. 
- `CommitBatch`: commits a batch (with a single chunk) from the latest uncommitted block to the latest block.
- `FinalizeBatch`: finalizes a batch.
- `RevertBatch`: reverts a batch.

### KeyManager
A simple in-memory key-management tool that provides some convenience functions around `ecdsa.PrivateKey` for transaction signing. Keys are stored and retrievable with an alias string.

## How to use
TODO: we need to figure out how to exactly use the testsuite in the most convenient way. This probably becomes clear when writing the first in-depth tests.  

## How to run
Due to using `ethereum/go-ethereum` (required for simulating L1) and `l2geth` (`scroll-tech/go-ethereum`, required to simulate L2), 
compiling the testsuite results in [duplicated symbols when linking](https://github.com/cosmos/cosmos-sdk/issues/18232#issuecomment-1782657851).

Therefore, we need to compile with `-ldflags=all="-extldflags=-Wl,--allow-multiple-definition"`. Unfortunately, this only works on Linux. A Docker image is provided for convenience.


```bash
# build Docker image
docker build -t testsuite -f Dockerfile.testing .
# run Docker container with correct context (local l2geth is used so that modifications are compiled as well) 
docker run --rm -it -v "$(dirname "$(dirname "$(pwd)")")":/go-ethereum testsuite /bin/bash -c "cd /go-ethereum/rollup/testsuite && exec bash"

# run tests with
go test -tags libsecp256k1_sdk -ldflags=all="-extldflags=-Wl,--allow-multiple-definition"
```