package test

import (
	peer "gx/ipfs/QmXYjuNuxVzXKJCfWasQk1RqkhVLDM9jtUKhqc2WPQmFSB/go-libp2p-peer"
	crypto "gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"

	"github.com/phoreproject/btcd/chaincfg"
	"github.com/phoreproject/openbazaar-go/bitcoin/phored"
	"github.com/phoreproject/openbazaar-go/core"
	"github.com/phoreproject/openbazaar-go/ipfs"
	"github.com/phoreproject/openbazaar-go/net"
	"github.com/phoreproject/openbazaar-go/net/service"
	bip39 "github.com/tyler-smith/go-bip39"
)

// NewNode creates a new *core.OpenBazaarNode prepared for testing
func NewNode() (*core.OpenBazaarNode, error) {
	// Create test repo
	repository, err := NewRepository()
	if err != nil {
		return nil, err
	}

	repository.Reset()
	if err != nil {
		return nil, err
	}

	// Create test ipfs node
	ipfsNode, err := ipfs.NewMockNode()
	if err != nil {
		return nil, err
	}

	seed := bip39.NewSeed(GetPassword(), "Secret Passphrase")
	privKey, err := ipfs.IdentityKeyFromSeed(seed, 256)
	if err != nil {
		return nil, err
	}

	sk, err := crypto.UnmarshalPrivateKey(privKey)
	if err != nil {
		return nil, err
	}

	id, err := peer.IDFromPublicKey(sk.GetPublic())
	if err != nil {
		return nil, err
	}

	ipfsNode.Identity = id

	// Create test context
	ctx, err := ipfs.MockCmdsCtx()
	if err != nil {
		return nil, err
	}

	// Create test wallet
	mnemonic, err := repository.DB.Config().GetMnemonic()
	if err != nil {
		return nil, err
	}

	wallet := phored.NewRPCWallet(mnemonic, &chaincfg.MainNetParams, repository.Path, repository.DB, "rpc.phore.io")

	// Put it all together in an OpenBazaarNode
	node := &core.OpenBazaarNode{
		Context:    ctx,
		RepoPath:   GetRepoPath(),
		IpfsNode:   ipfsNode,
		Datastore:  repository.DB,
		Wallet:     wallet,
		BanManager: net.NewBanManager([]peer.ID{}),
	}

	node.Service = service.New(node, ctx, repository.DB)

	return node, nil
}
