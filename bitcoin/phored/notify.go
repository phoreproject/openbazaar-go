package phored

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	logging "github.com/op/go-logging"
	"github.com/phoreproject/btcd/btcjson"
	"github.com/phoreproject/btcd/wire"
)

var log = logging.MustGetLogger("phored")

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

	toSend := []byte(fmt.Sprintf("subscribeBloom %s %d %d 0", hex.EncodeToString(message.Filter), message.HashFuncs, message.Tweak))

	// log.Debugf("<- %s", toSend)

	n.conn.WriteMessage(websocket.TextMessage, toSend)
}

func startNotificationListener(wallet *RPCWallet) (*NotificationListener, error) {
	notificationListener := &NotificationListener{wallet: wallet}
	u := url.URL{Scheme: "wss", Host: wallet.rpcBasePath, Path: "/ws"}

	log.Infof("Connecting to %s", u.String())
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	log.Infof("Connected to websockets!")

	if err != nil {
		return nil, err
	}

	notificationListener.conn = conn

	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for range ticker.C {
			// log.Debugf("<- ping")
			notificationListener.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
		}
	}()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err) {
					log.Infof("Reconnecting to %s", u.String())
					conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
					if err != nil {
						log.Error(err)
						return
					}
					notificationListener.conn = conn
				} else {
					log.Error(err)
					return
				}
			}

			// log.Debugf("-> %s", message)

			if string(message) == "pong" {
				continue
			}

			var getTx btcjson.GetTransactionResult
			err = json.Unmarshal(message, &getTx)

			if err == nil {
				txBytes, err := hex.DecodeString(getTx.Hex)
				if err != nil {
					log.Error(err)
					continue
				}

				transaction := wire.NewMsgTx(1)
				transaction.BtcDecode(bytes.NewReader(txBytes), 1, wire.BaseEncoding)

				hits, err := wallet.DB.Ingest(transaction, 0)
				if err != nil {
					log.Errorf("Error ingesting tx: %s\n", err.Error())
					continue
				}
				if hits == 0 {
					log.Debugf("Tx %s from Peer%d had no hits, filter false positive.", transaction.TxHash().String())
					continue
				}
				notificationListener.updateFilterAndSend()
				log.Infof("Tx %s ingested", transaction.TxHash().String())
			} else {
				fmt.Println(err)
			}
		}
	}()
	return notificationListener, nil
}
