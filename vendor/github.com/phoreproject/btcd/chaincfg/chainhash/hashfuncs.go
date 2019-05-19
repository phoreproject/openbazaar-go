// Copyright (c) 2015 The Decred developers
// Copyright (c) 2016-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chainhash

import (
	"crypto/sha256"

	"github.com/phoreproject/go-x11/blake"
	"github.com/phoreproject/go-x11/bmw"
	"github.com/phoreproject/go-x11/groest"
	"github.com/phoreproject/go-x11/jhash"
	"github.com/phoreproject/go-x11/keccak"
	"github.com/phoreproject/go-x11/skein"
)

// HashB calculates hash(b) and returns the resulting bytes.
func HashB(b []byte) []byte {
	hash := sha256.Sum256(b)
	return hash[:]
}

// HashH calculates hash(b) and returns the resulting bytes as a Hash.
func HashH(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

// DoubleHashB calculates hash(hash(b)) and returns the resulting bytes.
func DoubleHashB(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// DoubleHashH calculates hash(hash(b)) and returns the resulting bytes as a
// Hash.
func DoubleHashH(b []byte) Hash {
	first := sha256.Sum256(b)
	return Hash(sha256.Sum256(first[:]))
}

// QuarkHash calculates the quarkcoin hash of a specific value
func QuarkHash(data []byte) Hash {
	bmw1 := bmw.New()
	blake1 := blake.New()
	groest1 := groest.New()
	jhash1 := jhash.New()
	keccak1 := keccak.New()
	skein1 := skein.New()

	var out1a [64]byte
	var out2a [64]byte
	out1 := out1a[:]
	out2 := out2a[:]

	blake1.Write(data)
	blake1.Close(out1, 0, 0)

	bmw1.Write(out1)
	bmw1.Close(out2, 0, 0)

	if out2[0]&8 != 0 {
		groest1.Write(out2)
		groest1.Close(out1, 0, 0)
	} else {
		skein1.Write(out2)
		skein1.Close(out1, 0, 0)
	}

	groest1.Reset()
	groest1.Write(out1)
	groest1.Close(out2, 0, 0)

	jhash1.Write(out2)
	jhash1.Close(out1, 0, 0)

	if out1[0]&8 != 0 {
		blake1.Reset()
		blake1.Write(out1)
		blake1.Close(out2, 0, 0)
	} else {
		bmw1.Reset()
		bmw1.Write(out1)
		bmw1.Close(out2, 0, 0)
	}

	keccak1.Write(out2)
	keccak1.Close(out1, 0, 0)

	skein1.Reset()
	skein1.Write(out1)
	skein1.Close(out2, 0, 0)

	if out2[0]&8 != 0 {
		keccak1.Reset()
		keccak1.Write(out2)
		keccak1.Close(out1, 0, 0)
	} else {
		jhash1.Reset()
		jhash1.Write(out2)
		jhash1.Close(out1, 0, 0)
	}

	var out [32]byte

	copy(out[:], out1)

	return Hash(out)
}
