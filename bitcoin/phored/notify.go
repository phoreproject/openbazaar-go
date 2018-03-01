package phored

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/url"

	"github.com/gorilla/websocket"
	logging "github.com/op/go-logging"
	"github.com/phoreproject/btcd/btcjson"
	"github.com/phoreproject/btcd/chaincfg/chainhash"
	"github.com/phoreproject/btcd/wire"
)

var log = logging.MustGetLogger("bitcoind")
var baseURL = "rpc.phore.io"

// NotificationListener listens for any transactions
type NotificationListener struct {
	wallet *RPCWallet

	// websocket connection
	conn *websocket.Conn
}

func startNotificationListener(wallet *RPCWallet) error {
	notificationListener := NotificationListener{wallet: wallet}
	u := url.URL{Scheme: "wss", Host: baseURL, Path: "/ws"}

	log.Infof("Connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	notificationListener.conn = conn

	if err != nil {
		return err
	}

	defer notificationListener.conn.Close()

	wallet.notifications = &notificationListener

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Error(err)
				return
			}

			if string(message) == "pong" {
				continue
			}

			var getTx btcjson.GetTransactionResult
			err = json.Unmarshal(message, getTx)

			if err == nil {
				txBytes, err := hex.DecodeString(getTx.Hex)
				if err != nil {
					log.Error(err)
					continue
				}

				transaction := wire.NewMsgTx(1)
				transaction.BtcDecode(bytes.NewReader(txBytes), 1, wire.BaseEncoding)

				hash, err := chainhash.NewHash([]byte(getTx.BlockHash))
				if err != nil {
					log.Error(err)
					continue
				}

				block, err := wallet.rpcClient.GetBlockVerbose(hash)
				if err != nil {
					log.Error(err)
					continue
				}

				wallet.DB.Ingest(transaction, int32(block.Height))
			}
		}
	}()
	return nil
}

// if err != nil {
// 	log.Error(err)
// 	return
// }
// tx, err := l.client.GetRawTransaction(hash)
// if err != nil {
// 	log.Error(err)
// 	return
// }
// includeWatchOnly := true
// txInfo, err := l.client.GetTransaction(hash, &includeWatchOnly)
// var outputs []wallet.TransactionOutput
// for i, txout := range tx.MsgTx().TxOut {
// 	out := wallet.TransactionOutput{ScriptPubKey: txout.PkScript, Value: txout.Value, Index: uint32(i)}
// 	outputs = append(outputs, out)
// }
// var inputs []wallet.TransactionInput
// for _, txin := range tx.MsgTx().TxIn {
// 	in := wallet.TransactionInput{OutpointHash: txin.PreviousOutPoint.Hash.CloneBytes(), OutpointIndex: txin.PreviousOutPoint.Index}
// 	prev, err := l.client.GetRawTransaction(&txin.PreviousOutPoint.Hash)
// 	if err != nil {
// 		inputs = append(inputs, in)
// 		continue
// 	}
// 	in.LinkedScriptPubKey = prev.MsgTx().TxOut[txin.PreviousOutPoint.Index].PkScript
// 	in.Value = prev.MsgTx().TxOut[txin.PreviousOutPoint.Index].Value
// 	inputs = append(inputs, in)
// }

// height := int32(0)
// if txInfo.Confirmations > 0 {
// 	h, err := chainhash.NewHashFromStr(txInfo.BlockHash)
// 	if err != nil {
// 		log.Error(err)
// 		return
// 	}
// 	blockinfo, err := l.client.GetBlockHeaderVerbose(h)
// 	if err != nil {
// 		log.Error(err)
// 		return
// 	}
// 	height = blockinfo.Height
// }
// cb := wallet.TransactionCallback{
// 	Txid:      tx.Hash().CloneBytes(),
// 	Inputs:    inputs,
// 	Outputs:   outputs,
// 	Value:     int64(txInfo.Amount * 100000000),
// 	Timestamp: time.Unix(txInfo.TimeReceived, 0),
// 	Height:    height,
// }
// for _, lis := range l.listeners {
// 	lis(cb)
// }
// }
