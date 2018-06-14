package bitcoin

import (
	"github.com/phoreproject/openbazaar-go/api/notifications"
	"github.com/phoreproject/openbazaar-go/repo"
	"github.com/phoreproject/wallet-interface"
)

type WalletListener struct {
	db        repo.Datastore
	broadcast chan interface{}
}

func NewWalletListener(db repo.Datastore, broadcast chan interface{}) *WalletListener {
	l := &WalletListener{db, broadcast}
	return l
}

func (l *WalletListener) OnTransactionReceived(cb wallet.TransactionCallback) {
	if !cb.WatchOnly {
		txid := hex.EncodeToString(cb.Txid)
		metadata, _ := l.db.TxMetadata().Get(txid)
		status := "UNCONFIRMED"
		confirmations := 0
		if cb.Height > 0 {
			status = "PENDING"
			confirmations = 1
		}
		n := notifications.IncomingTransaction{
			Txid:          cb.Txid,
			Value:         cb.Value,
			Address:       metadata.Address,
			Status:        status,
			Memo:          metadata.Memo,
			Timestamp:     cb.Timestamp,
			Confirmations: int32(confirmations),
			OrderId:       metadata.OrderId,
			Thumbnail:     metadata.Thumbnail,
			Height:        cb.Height,
			CanBumpFee:    cb.Value > 0,
		}
		l.broadcast <- n
	}
}
