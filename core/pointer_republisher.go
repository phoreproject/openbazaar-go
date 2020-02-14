package core

import (
	net "github.com/phoreproject/openbazaar-go/net/repointer"
)

// StartPointerRepublisher - setup republisher for IPNS
func (n *OpenBazaarNode) StartPointerRepublisher() {
	n.PointerRepublisher = net.NewPointerRepublisher(n.DHT, n.Datastore, n.PushNodes, n.IsModerator)
	go n.PointerRepublisher.Run()
}
