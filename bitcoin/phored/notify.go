package phored

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

func (n *NotificationListener) updateFilterAndSend() {
	filt, err := n.wallet.DB.GimmeFilter()

	if err != nil {
		log.Error(err)
		return
	}

	message := filt.MsgFilterLoad()

	n.conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("subscribeFilter %s %d %d 0", hex.EncodeToString(message.Filter), message.HashFuncs, message.Tweak)))
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

				hits, err := wallet.DB.Ingest(transaction, int32(block.Height))
				if err != nil {
					log.Errorf("Error ingesting tx: %s\n", err.Error())
					continue
				}
				if hits == 0 {
					log.Debugf("Tx %s from Peer%d had no hits, filter false positive.", transaction.TxHash().String())
					continue
				}
				notificationListener.updateFilterAndSend()
				log.Infof("Tx %s ingested at height %d", transaction.TxHash().String(), block.Height)
			}
		}
	}()
	return nil
}
