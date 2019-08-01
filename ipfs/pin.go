package ipfs

import (
	"context"
<<<<<<< HEAD
=======

	"github.com/ipfs/go-ipfs/core/coreapi"

>>>>>>> 1eba569e5bc08b0e8756887aa5838fee26022b3c
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corerepo"
)

/* Recursively un-pin a directory given its hash.
   This will allow it to be garbage collected. */
func UnPinDir(n *core.IpfsNode, rootHash string) error {
	_, err := corerepo.Unpin(n, coreapi.NewCoreAPI(n), context.Background(), []string{"/ipfs/" + rootHash}, true)
	return err
}
