package test

import (
	// "github.com/ipfs/go-ipfs/thirdparty/testutil"
	"github.com/phoreproject/openbazaar-go/core"
	"github.com/phoreproject/openbazaar-go/ipfs"
	"github.com/phoreproject/openbazaar-go/net"
	"github.com/phoreproject/openbazaar-go/net/service"
	"github.com/phoreproject/spvwallet"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ipfs/go-ipfs/core/mock"
	"github.com/tyler-smith/go-bip39"
	"gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	"gx/ipfs/QmaPbCnUMBohSGo3KnxEa2bHqyJVVeEEcwtqJAYxerieBo/go-libp2p-crypto"
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
	ipfsNode, err := coremock.NewMockNode()
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

	// Create test wallet
	mnemonic, err := repository.DB.Config().GetMnemonic()
	if err != nil {
		return nil, err
	}

	wallet := phored.NewRPCWallet(mnemonic, &chaincfg.MainNetParams, repository.Path, repository.DB, "rpc.phore.io")

	// Put it all together in an OpenBazaarNode
	node := &core.OpenBazaarNode{
		RepoPath:   GetRepoPath(),
		IpfsNode:   ipfsNode,
		Datastore:  repository.DB,
		Wallet:     wallet,
		BanManager: net.NewBanManager([]peer.ID{}),
	}

	node.Service = service.New(node, repository.DB)

	return node, nil
}
