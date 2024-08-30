Due to using `ethereum/go-ethereum` (required for simulating L1) and `l2geth` (`scroll-tech/go-ethereum`, required to simulate L2), 
compiling the testsuite results in [duplicated symbols when linking](https://github.com/cosmos/cosmos-sdk/issues/18232#issuecomment-1782657851).

Therefore, we need to compile with `-ldflags=all="-extldflags=-Wl,--allow-multiple-definition"`. Unfortunately, this only works on Linux. A Docker image is provided for convenience.


```bash
docker build -t testsuite -f Dockerfile.testing .
docker run --rm -it -v "$(dirname "$(dirname "$(pwd)")")":/go-ethereum testsuite /bin/bash -c "cd /go-ethereum/rollup/testsuite && exec bash"

go test -tags libsecp256k1_sdk -ldflags=all="-extldflags=-Wl,--allow-multiple-definition"
```