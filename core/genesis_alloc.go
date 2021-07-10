// Copyright 2017 The go-ethereum Authors
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

package core

// Constants containing the genesis allocation of built-in genesis blocks.
// Their content is an RLP-encoded list of (address, balance) tuples.
// Use mkalloc.go to create/update them.

// nolint: misspell
const mainnetAllocData = "\xe2\xe1\x94S\xd7x'\xbe\x16\x8a\xb2\xa9\x11\xb5\xa1M\x0f\x16\xd1\xc5eq\x96\x8b\bE\x95\x16\x14\x01H\x80\x00\x00\x00"
const testnetAllocData = "\xf8B\xe0\x94lJ\xb0i\xaf\xfd\x85k\xb9\x15\xee\x93\xcbY7\x05t\xf53\x1e\x8a\xd3\xc2\x1b\xce\xcc\xed\xa0\x00\x00\x00\xe0\x94n\x93^\f\x8c\xf8:\xeaA\xc8\a\xac\xfc\x00\xb8X\x8c\xb5g\x17\x8a\xd3\xc2\x1b\xce\xcc\xed\xa0\x00\x00\x00"
