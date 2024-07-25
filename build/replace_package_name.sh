#!/bin/sh

find . -name "*.go" -type f -exec sed -i '' 's|"github.com/ethereum/go-ethereum|"github.com/scroll-tech/go-ethereum|g' {} +
find . -name "*.txt" -type f -exec sed -i '' 's|"github.com/ethereum/go-ethereum|"github.com/scroll-tech/go-ethereum|g' {} +
find . -name "go.mod" -type f -exec sed -i '' 's|github.com/ethereum/go-ethereum|github.com/scroll-tech/go-ethereum|g' {} +

