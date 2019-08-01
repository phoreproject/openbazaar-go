package phore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/gorilla/websocket"
	"github.com/op/go-logging"
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

	err = n.conn.WriteMessage(websocket.TextMessage, toSend)
	if err != nil {
		log.Errorf("Bloom filter subscription failed %s", err)
	}
}

func connectToWebsocket(n *NotificationListener, dialer *websocket.Dialer, url url.URL) error {
	conn, _, err := dialer.Dial(url.String(), nil)
	if err != nil {
		return err
	}

	conn.SetPingHandler(nil)
	conn.SetPongHandler(nil)

	closeHandlerFunc := func(code int, text string) error {
		if code == 4000 { // current node was marked as dead and connection was closed
			return connectToWebsocket(n, dialer, url)
		}
		return nil
	}
	conn.SetCloseHandler(closeHandlerFunc)

	if n.conn != nil {
		n.conn.Close()
	}
	n.conn = conn
	return nil
}

func startNotificationListener(wallet *RPCWallet) (*NotificationListener, error) {
	notificationListener := &NotificationListener{wallet: wallet}
	websocketURL := url.URL{Scheme: "wss", Host: "rpc2.phore.io", Path: "/ws"}

	dialerWithCookies := websocket.DefaultDialer
	dialerWithCookies.Jar = *new(http.CookieJar)

	log.Infof("Connecting to %s", websocketURL.String())
	err := connectToWebsocket(notificationListener, dialerWithCookies, websocketURL)
	if err != nil {
		return nil, err
	}
	log.Info("Connected to websockets!")

	ticker := time.NewTicker(15 * time.Second)
	go func() {
		for range ticker.C {
			// log.Debugf("<- ping")
			err := notificationListener.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second))
			if err != nil {
				log.Errorf("Error when pinging websocket. %s", err)
				log.Info("Trying to reconnect")
				err = connectToWebsocket(notificationListener, dialerWithCookies, websocketURL)
				if err != nil {
					log.Errorf("Reconnection failed. %s", err)
				}
				// if disconnected try to download transactions again and subscribe to bloom
				wallet.RetrieveTransactions()
				notificationListener.updateFilterAndSend()
			}
		}
	}()

	go func() {
		for {
			_, message, err := notificationListener.conn.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) {
					log.Warning(err)
					log.Infof("Reconnecting to %s", websocketURL.String())
					err = connectToWebsocket(notificationListener, dialerWithCookies, websocketURL)
					if err != nil {
						log.Errorf("Reconnection failed. %s", err)
					}

				} else {
					log.Error(err)
					return
				}
			}

			if string(message) == "success" || len(message) == 0 {
				continue
			}

			if strings.HasPrefix(string(message), "error") {
				log.Warningf("Websocket returned - %s", string(message))
				continue
			}

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
