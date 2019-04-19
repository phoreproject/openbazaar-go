package dropbox

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"

	mh "gx/ipfs/QmPnFwZ2JXKnXgMw8CdBPxn7FWh6LLdjUjxV1fKHuJnkr8/go-multihash"
	ma "gx/ipfs/QmT4U94DnD8FRfqr21obWY32HLM5VExccPKMjQHofeYqr9/go-multiaddr"
	peer "gx/ipfs/QmTRhk7cgjUf2gfQ3p2M9KPECNZEW9XUrmHcFCgog4cPgB/go-libp2p-peer"

	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/files"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/sharing"
	"github.com/dropbox/dropbox-sdk-go-unofficial/dropbox/users"
)

// DropBoxStorage is a pluggable dropbox storage for OB
type DropBoxStorage struct {
	apiToken string
}

// NewDropBoxStorage generates a new API client
func NewDropBoxStorage(apiToken string) (*DropBoxStorage, error) {
	config := dropbox.Config{Token: apiToken}
	api := users.New(config)
	if _, err := api.GetCurrentAccount(); err != nil {
		return nil, err
	}
	return &DropBoxStorage{
		apiToken: apiToken,
	}, nil
}

// Store stores a file on dropbox
func (s *DropBoxStorage) Store(peerID peer.ID, ciphertext []byte) (ma.Multiaddr, error) {
	config := dropbox.Config{Token: s.apiToken, LogLevel: dropbox.LogDebug}
	filesAPI := files.New(config)
	sharingAPI := sharing.New(config)
	hash := sha256.Sum256(ciphertext)
	hexStr := hex.EncodeToString(hash[:])

	// Upload ciphertext
	uploadArg := files.NewCommitInfo("/" + hexStr)
	r := bytes.NewReader(ciphertext)
	_, err := filesAPI.Upload(uploadArg, r)
	if err != nil {
		return nil, err
	}

	// Set public sharing
	sharingArg := sharing.NewCreateSharedLinkArg("/" + hexStr)
	res, err := api.CreateSharedLink(sharingArg)
	if err != nil {
		return nil, err
	}

	// Create encoded multiaddr
	url := res.Url[:len(res.Url)-1] + "1"
	b, err := mh.Encode([]byte(url), mh.SHA1)
	if err != nil {
		return nil, err
	}
	m, err := mh.Cast(b)
	if err != nil {
		return nil, err
	}

	addr, err := ma.NewMultiaddr("/ipfs/" + m.B58String() + "/https/")
	if err != nil {
		return nil, err
	}
	return addr, nil
}
