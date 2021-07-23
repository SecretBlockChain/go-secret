// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import "github.com/SecretBlockChain/go-secret/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Secret network.
var MainnetBootnodes = []string{
	// Secret Foundation Go Bootnodes
	"enode://342ad34ac6001a54b977a1673014ab5b6a080e08ba0cb899675694d6172ba82a1d63c3b5ee831d0674b48fbb20118f1c1f1cc33149f54a21dd0b6a48cd19a3cf@18.166.178.151:30310",
	"enode://40c107850e014b9a39392dc9a430b5bac78817658eacdfa109187070a1c40219c948f27d794a82dd555cfad39d2f7b7dfe2924a7b7d46d8075e38b94a80c967c@18.166.177.253:30310",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// the testnet network.
var TestnetBootnodes = []string{
	"enode://7ad67ce3b1d49c687bcb11aea995f4e7bb10a92a297890a4547f7b7f8ebe344d4a5c1bd63801178f77234b5c1f18f5223c8de20b22064470521f851ef3f37e19@207.46.145.61:30311",
}

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	return ""
}
