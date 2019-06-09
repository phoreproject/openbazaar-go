package bitcoin

import (
	"github.com/phoreproject/multiwallet/util"
	"github.com/phoreproject/openbazaar-go/repo"
	"github.com/OpenBazaar/wallet-interface"
)

type WalletListener struct {
	db        repo.Datastore
	broadcast chan repo.Notifier
	coinType  util.ExtCoinType
}

func NewWalletListener(db repo.Datastore, broadcast chan repo.Notifier, coinType util.ExtCoinType) *WalletListener {
	l := &WalletListener{db, broadcast, coinType}
	return l
}

func (l *WalletListener) OnTransactionReceived(cb wallet.TransactionCallback) {
	if !cb.WatchOnly {
		metadata, _ := l.db.TxMetadata().Get(cb.Txid)
		status := "UNCONFIRMED"
		confirmations := 0
		if cb.Height > 0 {
			status = "PENDING"
			confirmations = 1
		}
		n := repo.IncomingTransaction{
			Wallet:        l.coinType.CurrencyCode(),
			Txid:          cb.Txid,
			Value:         cb.Value,
			Address:       metadata.Address,
			Status:        status,
			Memo:          metadata.Memo,
			Timestamp:     cb.Timestamp,
			Confirmations: int32(confirmations),
			OrderId:       metadata.OrderID,
			Thumbnail:     metadata.Thumbnail,
			Height:        cb.Height,
			CanBumpFee:    cb.Value > 0,
		}
		l.broadcast <- n
	}
}
