# Go Secret
Official Golang implementation of the Secret protocol.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/SecretBlockChain/go-secret)
[![License](https://img.shields.io/badge/license-GPL%20v3-blue.svg)](LICENSE)

## About Secret

Secret is base on [go-ethereum](https://github.com/ethereum/go-ethereum) implementation, We add a new delegated-proof-of-stake consensus algorithm named [senate](consensus/senate/) in it.

Senate use header.extra to record the all infomation of current block and keep signature of miner. The snapshot keep vote & confirm information of whole chain, which will be update by each Seal or VerifySeal. By the end of each loop, the miner will calculate the next loop miners base on the snapshot. Code annotation will show the details about how it works.

## Contact

email: dev@secret.dev
