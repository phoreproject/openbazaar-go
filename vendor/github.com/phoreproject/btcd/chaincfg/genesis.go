// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"time"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
	"github.com/phoreproject/btcd/wire"
)

// genesisCoinbaseTx is the coinbase transaction for the genesis blocks for
// the main network, regression test network, and test network (version 3).
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  chainhash.Hash{},
				Index: 0xffffffff,
			},
			SignatureScript: []byte{
				0x04, 0xff, 0xff, 0x00, 0x1d, 0x01, 0x04, 0x11, /* |........| */
				0x31, 0x32, 0x20, 0x53, 0x65, 0x70, 0x74, 0x65, /* |12 Septe| */
				0x6d, 0x62, 0x65, 0x72, 0x20, 0x32, 0x30, 0x31, /* |mber 201| */
				0x37, /* |7| */
			},
			Sequence: 0xffffffff,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value:    0,
			PkScript: []byte{},
		},
	},
	LockTime: 0,
}

// genesisHash is the hash of the first block in the block chain for the main
// network (genesis block).
var genesisHash = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xbf, 0x85, 0x04, 0x38, 0x7e, 0x10, 0x19, 0x30,
	0xd7, 0x11, 0x95, 0xe8, 0x1c, 0x92, 0x25, 0x5a,
	0x41, 0x19, 0xb9, 0xd5, 0x62, 0x36, 0x28, 0xad,
	0x59, 0xad, 0x2a, 0x71, 0x66, 0x0f, 0x1a, 0x2b,
})

// genesisMerkleRoot is the hash of the first transaction in the genesis block
// for the main network.
var genesisMerkleRoot = chainhash.Hash([chainhash.HashSize]byte{ // Make go vet happy.
	0xa3, 0xb3, 0x36, 0x07, 0xcb, 0x85, 0x1a, 0x7c,
	0xfc, 0xde, 0x34, 0x8f, 0x54, 0x53, 0x8a, 0x20,
	0x85, 0xfc, 0xe7, 0x95, 0xd3, 0x9d, 0xd8, 0xfe,
	0x2c, 0x95, 0x45, 0x7a, 0x13, 0x77, 0x41, 0x89,
})

// genesisBlock defines the genesis block of the block chain which serves as the
// public transaction ledger for the main network.
var genesisBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash{},         // 0000000000000000000000000000000000000000000000000000000000000000
		MerkleRoot: genesisMerkleRoot,        // 4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b
		Timestamp:  time.Unix(1505224800, 0), // 2009-01-03 18:15:05 +0000 UTC
		Bits:       0x207fffff,               // 486604799 [00000000ffff0000000000000000000000000000000000000000000000000000]
		Nonce:      12345,                    // 2083236893
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}
