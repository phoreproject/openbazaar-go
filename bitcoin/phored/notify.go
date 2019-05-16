package phored

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
	"github.com/phoreproject/btcd/btcjson"
	"github.com/phoreproject/btcd/chaincfg/chainhash"
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
	filt, err := n.wallet.txstore.GimmeFilter()

	if err != nil {
		log.Error(err)
		return
	}

	message := filt.MsgFilterLoad()

	toSend := []byte(fmt.Sprintf("subscribeBloom %s %d %d 0", hex.EncodeToString(message.Filter), message.HashFuncs, message.Tweak))

	//log.Debugf("<- toSend %s", toSend)

	err = n.conn.WriteMessage(websocket.TextMessage, toSend)
	if err != nil{
		log.Errorf("Bloom filter subscription failed %s", err)
	}
}

func startNotificationListener(wallet *RPCWallet) (*NotificationListener, error) {
	notificationListener := &NotificationListener{wallet: wallet}
	u := url.URL{Scheme: "wss", Host: wallet.rpcBasePath, Path: "/ws"}

	log.Infof("Connecting to %s", u.String())

	dialerWithCookies := websocket.DefaultDialer
	dialerWithCookies.Jar = *new(http.CookieJar)
	conn, _, err := dialerWithCookies.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	log.Info("Connected to websockets!")

	notificationListener.conn = conn

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			// log.Debugf("<- ping")
			err := notificationListener.conn.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				log.Errorf("Error when pinging websocket. %s", err)
			}
		}
	}()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) {
					log.Infof("Reconnecting to %s", u.String())
					conn, _, err := dialerWithCookies.Dial(u.String(), nil)
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

			if string(message) == "pong" || string(message) == "ping" || string(message) == "success" {
				continue
			}

			if strings.HasPrefix(string(message), "error") {
				log.Warningf("Websocket returned - %s", string(message))
				continue
			}

			log.Debugf("-> %s", message)

			var getTx btcjson.GetTransactionResult
			err = json.Unmarshal(message, &getTx)

			if err == nil {
				err = ingestTransaction(wallet, getTx)
				if err != nil {
					continue
				}
				notificationListener.updateFilterAndSend()
			} else {
				log.Errorf("msg: %s, err: %s", string(message), err)
			}
		}
	}()
	return notificationListener, nil
}

func ingestTransaction(wallet *RPCWallet, getTx btcjson.GetTransactionResult) error {
	txBytes, err := hex.DecodeString(getTx.Hex)
	if err != nil {
		log.Error(err)
	}

	transaction := wire.NewMsgTx(1)
	transaction.BtcDecode(bytes.NewReader(txBytes), 1, wire.BaseEncoding)

	var blockHeight int32

	if getTx.BlockHash != "" {
		blockhash, err := chainhash.NewHashFromStr(getTx.BlockHash)
		if err != nil {
			log.Error(err)
		}

		block, err := wallet.rpcClient.GetBlockVerbose(blockhash)
		if err != nil {
			log.Error(err)
		}
		blockHeight = int32(block.Height)
	}

	hits, err := wallet.txstore.Ingest(transaction, blockHeight, time.Unix(getTx.BlockTime, 0))
	if err != nil {
		log.Errorf("Error ingesting tx: %s\n", err.Error())
	}
	if hits == 0 {
		log.Debugf("Tx %s from Peer%d had no hits, filter false positive.", transaction.TxHash().String())
	}
	log.Infof("Tx %s ingested", transaction.TxHash().String())
	return nil
}