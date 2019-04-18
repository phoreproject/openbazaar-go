package net

import (
	peer "gx/ipfs/QmZoWKhxUmZ2seW4BzX6fJkNR8hh9PsGModr7q171yq2SS/go-libp2p-peer"
	"sync"
)

type BanManager struct {
	blockedIds map[string]bool
	*sync.RWMutex
}

func NewBanManager(blockedIds []peer.ID) *BanManager {
	blockedMap := make(map[string]bool)
	for _, pid := range blockedIds {
		blockedMap[pid.Pretty()] = true
	}
	return &BanManager{blockedMap, new(sync.RWMutex)}
}

func (bm *BanManager) AddBlockedId(peerID peer.ID) {
	bm.Lock()
	defer bm.Unlock()
	bm.blockedIds[peerID.Pretty()] = true
}

func (bm *BanManager) RemoveBlockedId(peerID peer.ID) {
	bm.Lock()
	defer bm.Unlock()
	if bm.blockedIds[peerID.Pretty()] {
		delete(bm.blockedIds, peerID.Pretty())
	}
}

func (bm *BanManager) SetBlockedIds(peerIDs []peer.ID) {
	bm.Lock()
	defer bm.Unlock()

	bm.blockedIds = make(map[string]bool)

	for _, pid := range peerIDs {
		bm.blockedIds[pid.Pretty()] = true
	}
}

func (bm *BanManager) GetBlockedIds() []peer.ID {
	bm.RLock()
	defer bm.RUnlock()
	var ret []peer.ID
	for pid := range bm.blockedIds {
		id, err := peer.IDB58Decode(pid)
		if err != nil {
			continue
		}
		ret = append(ret, id)
	}
	return ret
}

func (bm *BanManager) IsBanned(peerID peer.ID) bool {
	bm.RLock()
	defer bm.RUnlock()
	return bm.blockedIds[peerID.Pretty()]
}
