package phored

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/phoreproject/btcd/blockchain"
	"github.com/phoreproject/btcd/btcec"
	"github.com/phoreproject/btcd/chaincfg"
	"github.com/phoreproject/btcd/chaincfg/chainhash"
	"github.com/phoreproject/btcd/rpcclient"
	"github.com/phoreproject/btcd/txscript"
	"github.com/phoreproject/btcd/wire"
	btc "github.com/phoreproject/btcutil"
	"github.com/phoreproject/btcutil/coinset"
	hd "github.com/phoreproject/btcutil/hdkeychain"
	"github.com/phoreproject/btcutil/txsort"
	"github.com/phoreproject/btcwallet/wallet/txrules"
	"github.com/phoreproject/spvwallet"
	wallet "github.com/phoreproject/wallet-interface"
	b39 "github.com/tyler-smith/go-bip39"
)

const (
	// Account is the name for the Bitcoin wallet account
	Account = "OpenBazaar"
)

// RPCWallet represents a wallet based on JSON-RPC and Bitcoind
type RPCWallet struct {
	params           *chaincfg.Params

	masterPrivateKey *hd.ExtendedKey
	masterPublicKey  *hd.ExtendedKey

	mnemonic         string

	//feeProvider    *FeeProvider

	repoPath         string

	rpcClient        *rpcclient.Client

	keyManager       *spvwallet.KeyManager

	txstore          *TxStore
	connCfg          *rpcclient.ConnConfig
	notifications    *NotificationListener
	rpcBasePath      string
	rpcLock          *sync.Mutex

	started          bool
}

// NewRPCWallet creates a new wallet given
func NewRPCWallet(mnemonic string, params *chaincfg.Params, repoPath string, DB wallet.Datastore, host string) (*RPCWallet, error) {
	if mnemonic == "" {
		ent, _ := b39.NewEntropy(128)
		mnemonic, _ = b39.NewMnemonic(ent)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:                 path.Join(host, "rpc"),
		HTTPPostMode:         true,
		DisableTLS:           false,
		DisableAutoReconnect: false,
		DisableConnectOnNew:  false,
	}

	seed, err := b39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	mPrivKey, err := hd.NewMaster(seed, params)
	if err != nil {
		return nil, err
	}
	mPubKey, err := mPrivKey.Neuter()
	if err != nil {
		return nil, err
	}

	keyManager, err := spvwallet.NewKeyManager(DB.Keys(), params, mPrivKey)
	if err != nil {
		return nil, err
	}

	txstore, err := NewTxStore(params, DB, keyManager)
	if err != nil {
		return nil, err
	}

	w := RPCWallet{
		params:           params,
		repoPath:         repoPath,
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		keyManager:       keyManager,
		txstore:          txstore,
		connCfg:          connCfg,
		rpcBasePath:      host,
		rpcLock:          new(sync.Mutex),
	}
	return &w, nil
}

// Start sets up the rpc wallet
func (w *RPCWallet) Start() {
	client, _ := rpcclient.New(w.connCfg, nil)
	w.rpcClient = client

	ticker := time.NewTicker(time.Second * 30)
	go func() {
		for range ticker.C {
			log.Fatal("Failed to connect to phored")
		}
	}()
	for {
		_, err := client.GetBlockCount()
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}
	ticker.Stop()

	// notification listener must start after rpc connection is stabilized
	n, err := startNotificationListener(w)
	if err != nil {
		log.Error(err)
		return
	}
	w.notifications = n

	err = w.RetrieveTransactions()
	if err != nil {
		log.Error(err)
		return
	}

	w.notifications.updateFilterAndSend()

	unbroadcastedTransactions, err := w.txstore.GetPendingInv()
	if err != nil {
		log.Error(err)
		return
	}

	for tx := range unbroadcastedTransactions.InvList {
		hash := unbroadcastedTransactions.InvList[tx].Hash
		log.Debugf("Found transaction unbroadcasted: %s", hash.String())
		txn, err := w.txstore.Txns().Get(hash)
		if err != nil {
			log.Error(err)
			continue
		}

		r := bytes.NewReader(txn.Bytes)
		transaction := wire.NewMsgTx(wire.TxVersion)
		transaction.DeserializeNoWitness(r)
		err = w.Broadcast(transaction)
		if err != nil {
			if strings.HasPrefix(err.Error(), "-27") {
				// transaction is already in the blockchain, so go retrieve it
				res, err := w.rpcClient.GetRawTransactionVerbose(&hash)
				if err != nil {
					log.Error(err)
					continue
				}

				blockHash, err := chainhash.NewHashFromStr(res.BlockHash)
				if err != nil {
					log.Error(err)
					continue
				}

				w.rpcLock.Lock()
				block, err := w.rpcClient.GetBlockVerbose(blockHash)
				w.rpcLock.Unlock()

				if err != nil {
					log.Error(err)
					continue
				}

				transactionBytes, err := hex.DecodeString(res.Hex)
				if err != nil {
					log.Error(err)
					continue
				}

				transaction := wire.MsgTx{}
				err = transaction.BtcDecode(bytes.NewReader(transactionBytes), 1, wire.BaseEncoding)
				if err != nil {
					log.Error(err)
					continue
				}

				w.txstore.Ingest(&transaction, int32(block.Height), time.Unix(block.Time, 0))
			} else if strings.HasPrefix(err.Error(), "-26") {
				// transaction is spending inputs already spent, so we should just remove it
				w.txstore.Txns().Delete(&hash)
			} else {
				log.Error(err)
			}
		}
	}

	log.Info("Connected to phored")
	w.started = true
}

// CurrencyCode returns the currency code of the wallet
func (w *RPCWallet) CurrencyCode() string {
	if w.params.Name == chaincfg.MainNetParams.Name {
		return "phr"
	} else {
		return "tphr"
	}
}

// IsDust determines if an amount is considered dust
func (w *RPCWallet) IsDust(amount int64) bool {
	return txrules.IsDustAmount(btc.Amount(amount), 25, txrules.DefaultRelayFeePerKb)
}

// MasterPrivateKey returns the wallet's master private key
func (w *RPCWallet) MasterPrivateKey() *hd.ExtendedKey {
	return w.masterPrivateKey
}

// MasterPublicKey returns the wallet's key used to derive public keys
func (w *RPCWallet) MasterPublicKey() *hd.ExtendedKey {
	return w.masterPublicKey
}

func (w *RPCWallet) ChildKey(keyBytes []byte, chaincode []byte, isPrivateKey bool) (*hd.ExtendedKey, error) {
	parentFP := []byte{0x00, 0x00, 0x00, 0x00}
	var id []byte
	if isPrivateKey {
		id = w.params.HDPrivateKeyID[:]
	} else {
		id = w.params.HDPublicKeyID[:]
	}
	hdKey := hd.NewExtendedKey(
		id,
		keyBytes,
		chaincode,
		parentFP,
		0,
		0,
		isPrivateKey)
	return hdKey.Child(0)
}

// Mnemonic returns the mnemonis used to generate the wallet
func (w *RPCWallet) Mnemonic() string {
	return w.mnemonic
}

// CurrentAddress returns an unused address
func (w *RPCWallet) CurrentAddress(purpose wallet.KeyPurpose) btc.Address {
	key, _ := w.keyManager.GetCurrentKey(purpose)
	addr, _ := key.Address(w.params)
	return btc.Address(addr)
}

// NewAddress creates a new address
func (w *RPCWallet) NewAddress(purpose wallet.KeyPurpose) btc.Address {
	i, _ := w.txstore.Keys().GetUnused(purpose)
	key, _ := w.keyManager.GenerateChildKey(purpose, uint32(i[1]))
	addr, _ := key.Address(w.params)
	w.txstore.Keys().MarkKeyAsUsed(addr.ScriptAddress())
	w.txstore.PopulateAdrs()
	return btc.Address(addr)
}

// DecodeAddress decodes an address string to an address using the wallet's chain parameters
func (w *RPCWallet) DecodeAddress(addr string) (btc.Address, error) {
	return btc.DecodeAddress(addr, w.params)
}

// ScriptToAddress converts a script to an address
func (w *RPCWallet) ScriptToAddress(script []byte) (btc.Address, error) {
	return scriptToAddress(script, w.params)
}

func scriptToAddress(script []byte, params *chaincfg.Params) (btc.Address, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, params)
	if err != nil {
		return &btc.AddressPubKeyHash{}, err
	}
	if len(addrs) == 0 {
		return &btc.AddressPubKeyHash{}, errors.New("unknown script")
	}
	return addrs[0], nil
}

// AddressToScript returns the script for a given address
func (w *RPCWallet) AddressToScript(addr btc.Address) ([]byte, error) {
	return txscript.PayToAddrScript(addr)
}

// HasKey returns true if we have the private key for a given address
func (w *RPCWallet) HasKey(addr btc.Address) bool {
	_, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	if err != nil {
		return false
	}
	return true
}

// GetKey gets the private key for a certain address
func (w *RPCWallet) GetKey(addr btc.Address) (*btcec.PrivateKey, error) {
	key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
	if err != nil {
		return nil, err
	}
	return key.ECPrivKey()
}

// ListAddresses lists our currently used addresses
func (w *RPCWallet) ListAddresses() []btc.Address {
	keys := w.keyManager.GetKeys()
	addrs := []btc.Address{}
	for _, k := range keys {
		addr, err := k.Address(w.params)
		if err != nil {
			continue
		}
		addrs = append(addrs, addr)
	}
	return addrs
}

// ListKeys lists our currently used keys
func (w *RPCWallet) ListKeys() []btcec.PrivateKey {
	keys := w.keyManager.GetKeys()
	list := []btcec.PrivateKey{}
	for _, k := range keys {
		priv, err := k.ECPrivKey()
		if err != nil {
			continue
		}
		list = append(list, *priv)
	}
	return list
}

// Balance returns the total balance of our addresses
func (w *RPCWallet) Balance() (confirmed, unconfirmed int64) {
	utxos, _ := w.txstore.Utxos().GetAll()
	stxos, _ := w.txstore.Stxos().GetAll()
	for _, utxo := range utxos {
		if !utxo.WatchOnly {
			if utxo.AtHeight > 0 {
				confirmed += utxo.Value
			} else {
				if w.checkIfStxoIsConfirmed(utxo, stxos) {
					confirmed += utxo.Value
				} else {
					unconfirmed += utxo.Value
				}
			}
		}
	}
	return confirmed, unconfirmed
}

// Transactions returns all of the transactions relating to any of our addresses
func (w *RPCWallet) Transactions() ([]wallet.Txn, error) {
	height, _ := w.ChainTip()
	txns, err := w.txstore.Txns().GetAll(false)
	if err != nil {
		return txns, err
	}
	for i, tx := range txns {
		var confirmations int32
		var status wallet.StatusCode
		confs := int32(height) - tx.Height + 1
		if tx.Height <= 0 {
			confs = tx.Height
		}
		switch {
		case confs < 0:
			status = wallet.StatusDead
		case confs == 0 && time.Since(tx.Timestamp) <= time.Minute * 15:
			status = wallet.StatusUnconfirmed
		case confs == 0 && time.Since(tx.Timestamp) > time.Minute * 15:
			status = wallet.StatusDead
		case confs > 0 && confs < 6:
			status = wallet.StatusPending
			confirmations = confs
		case confs > 5:
			status = wallet.StatusConfirmed
			confirmations = confs
		}
		tx.Confirmations = int64(confirmations)
		tx.Status = status
		txns[i] = tx
	}
	return txns, nil
}

// GetTransaction returns the transaction given by a transaction hash
func (w *RPCWallet) GetTransaction(txid chainhash.Hash) (wallet.Txn, error) {
	return w.txstore.Txns().Get(txid)
}

// GetConfirmations returns the number of confirmations and the block number where the transaction was confirmed
func (w *RPCWallet) GetConfirmations(txid chainhash.Hash) (uint32, uint32, error) {
	txn, err := w.txstore.Txns().Get(txid)
	if err != nil {
		return 0, 0, err
	}
	if txn.Height == 0 {
		return 0, 0, nil
	}
	chainTip, _ := w.ChainTip()
	return chainTip - uint32(txn.Height) + 1, uint32(txn.Height), nil
}

func (w *RPCWallet) checkIfStxoIsConfirmed(utxo wallet.Utxo, stxos []wallet.Stxo) bool {
	for _, stxo := range stxos {
		if !stxo.Utxo.WatchOnly {
			if stxo.SpendTxid.IsEqual(&utxo.Op.Hash) {
				if stxo.SpendHeight > 0 {
					return true
				} else {
					return w.checkIfStxoIsConfirmed(stxo.Utxo, stxos)
				}
			} else if stxo.Utxo.IsEqual(&utxo) {
				if stxo.Utxo.AtHeight > 0 {
					return true
				} else {
					return false
				}
			}
		}
	}
	return false
}

// Params returns the current wallet's chain params
func (w *RPCWallet) Params() *chaincfg.Params {
	return w.params
}

// AddTransactionListener adds a listener for any wallet transactions
func (w *RPCWallet) AddTransactionListener(callback func(wallet.TransactionCallback)) {
	w.txstore.listeners = append(w.txstore.listeners, callback)
}

// ChainTip returns the tip of the active blockchain
func (w *RPCWallet) ChainTip() (uint32, chainhash.Hash) {
	w.rpcLock.Lock()
	ch, err := w.rpcClient.GetBestBlockHash()
	if err != nil {
		return 0, chainhash.Hash{}
	}

	height, err := w.rpcClient.GetBlockCount()
	w.rpcLock.Unlock()
	return uint32(height), *ch
}

func (w *RPCWallet) AddWatchedAddress(addr btc.Address) error {
	script, err := w.AddressToScript(addr)
	if err != nil {
		return err
	}
	err = w.txstore.WatchedScripts().Put(script)
	w.txstore.PopulateAdrs()
	if err != nil {
		return err
	}
	log.Debugf("addWatchedAddress %s\n", addr.String())
	return nil
}

func (w *RPCWallet) addWatchedScript(addr btc.Address) error {
	return w.rpcClient.ImportAddressRescan(addr.EncodeAddress(), "", false)
}

// ReSyncBlockchain resyncs the addresses used by the SPV wallet
func (w *RPCWallet) ReSyncBlockchain(fromDate time.Time) {
	w.txstore.PopulateAdrs()
	w.RetrieveTransactions()
}

// Close closes the rpc wallet connection
func (w *RPCWallet) Close() {
	if w.started {
		log.Info("Disconnecting from peers and shutting down")
		w.rpcLock.Lock()
		defer w.rpcLock.Unlock()

		// add timer to shutdown execution
		ch := make(chan bool, 1)
		defer close(ch)

		go func() {
			w.rpcClient.Shutdown()
			ch <- true
		}()

		select {
		case <-ch:
			log.Debugf("RPC client shutdown normally")
		case <-time.After(60 * time.Second):
			log.Debugf("RPC client shutdown timeout")
		}
		w.started = false
	}
}

// SweepAddress sweeps any UTXOs from an address in a single transaction
func (w *RPCWallet) SweepAddress(ins []wallet.TransactionInput, address *btc.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	var internalAddr btc.Address
	if address != nil {
		internalAddr = *address
	} else {
		internalAddr = w.CurrentAddress(wallet.INTERNAL)
	}
	script, err := txscript.PayToAddrScript(internalAddr)
	if err != nil {
		return nil, err
	}

	var val int64
	var inputs []*wire.TxIn
	additionalPrevScripts := make(map[wire.OutPoint][]byte)
	for _, in := range ins {
		val += in.Value
		ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
		if err != nil {
			return nil, err
		}
		script, err := txscript.PayToAddrScript(in.LinkedAddress)
		if err != nil {
			return nil, err
		}
		outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
		input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
		inputs = append(inputs, input)
		additionalPrevScripts[*outpoint] = script
	}
	out := wire.NewTxOut(val, script)

	txType := spvwallet.P2PKH
	if redeemScript != nil {
		txType = spvwallet.P2SH_1of2_Multisig
		_, err := spvwallet.LockTimeFromRedeemScript(*redeemScript)
		if err == nil {
			txType = spvwallet.P2SH_Multisig_Timelock_1Sig
		}
	}
	estimatedSize := spvwallet.EstimateSerializeSize(len(ins), []*wire.TxOut{out}, false, txType)

	// Calculate the fee
	feePerByte := int(w.GetFeePerByte(feeLevel))
	fee := estimatedSize * feePerByte

	outVal := val - int64(fee)
	if outVal < 0 {
		outVal = 0
	}
	out.Value = outVal

	tx := &wire.MsgTx{
		Version:  wire.TxVersion,
		TxIn:     inputs,
		TxOut:    []*wire.TxOut{out},
		LockTime: 0,
	}

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	// Sign tx
	privKey, err := key.ECPrivKey()
	if err != nil {
		return nil, err
	}
	pk := privKey.PubKey().SerializeCompressed()
	addressPub, err := btc.NewAddressPubKey(pk, w.params)

	getKey := txscript.KeyClosure(func(addr btc.Address) (*btcec.PrivateKey, bool, error) {
		if addressPub.EncodeAddress() == addr.EncodeAddress() {
			wif, err := btc.NewWIF(privKey, w.params, true)
			if err != nil {
				return nil, false, err
			}
			return wif.PrivKey, wif.CompressPubKey, nil
		}
		return nil, false, errors.New("Not found")
	})
	getScript := txscript.ScriptClosure(func(addr btc.Address) ([]byte, error) {
		if redeemScript == nil {
			return []byte{}, nil
		}
		return *redeemScript, nil
	})

	// Check if time locked
	var timeLocked bool
	if redeemScript != nil {
		rs := *redeemScript
		if rs[0] == txscript.OP_IF {
			timeLocked = true
			tx.Version = 2
		}
		for _, txIn := range tx.TxIn {
			locktime, err := spvwallet.LockTimeFromRedeemScript(*redeemScript)
			if err != nil {
				return nil, err
			}
			txIn.Sequence = locktime
		}
	}

	hashes := txscript.NewTxSigHashes(tx)
	for i, txIn := range tx.TxIn {
		if redeemScript == nil {
			prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
			script, err := txscript.SignTxOutput(w.params,
				tx, i, prevOutScript, txscript.SigHashAll, getKey,
				getScript, txIn.SignatureScript)
			if err != nil {
				return nil, errors.New("Failed to sign transaction")
			}
			txIn.SignatureScript = script
		} else {
			sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, ins[i].Value, *redeemScript, txscript.SigHashAll, privKey)
			if err != nil {
				return nil, err
			}
			var witness wire.TxWitness
			if timeLocked {
				witness = wire.TxWitness{sig, []byte{}}
			} else {
				witness = wire.TxWitness{[]byte{}, sig}
			}
			witness = append(witness, *redeemScript)
			txIn.Witness = witness
		}
	}

	// broadcast
	//_, err = w.rpcClient.SendRawTransaction(tx, false)
	//if err != nil {
	//	return nil, err
	//}

	w.Broadcast(tx)
	txid := tx.TxHash()
	return &txid, nil
}

// GetFeePerByte gets the fee in pSAT per byte
func (w *RPCWallet) GetFeePerByte(feeLevel wallet.FeeLevel) uint64 {
	return 10
}

// Broadcast a transaction to the network
func (w *RPCWallet) Broadcast(tx *wire.MsgTx) error {
	// Our own tx; don't keep track of false positives
	_, err := w.txstore.Ingest(tx, 0, time.Now())
	if err != nil {
		return err
	}

	w.rpcLock.Lock()
	_, err = w.rpcClient.SendRawTransaction(tx, false)
	w.rpcLock.Unlock()
	if err != nil {
		return err
	}

	w.notifications.updateFilterAndSend()
	return nil
}

// BumpFee attempts to bump the fee for a transaction
func (w *RPCWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	includeWatchOnly := false
	tx, err := w.rpcClient.GetTransaction(&txid, &includeWatchOnly)
	if err != nil {
		return nil, err
	}
	if tx.Confirmations > 0 {
		return nil, spvwallet.BumpFeeAlreadyConfirmedError
	}
	unspent, err := w.rpcClient.ListUnspent()
	if err != nil {
		return nil, err
	}
	for _, u := range unspent {
		if u.TxID == txid.String() {
			if u.Confirmations > 0 {
				return nil, spvwallet.BumpFeeAlreadyConfirmedError
			}
			h, err := chainhash.NewHashFromStr(u.TxID)
			if err != nil {
				continue
			}
			addr, err := btc.DecodeAddress(u.Address, w.params)
			if err != nil {
				continue
			}
			key, err := w.rpcClient.DumpPrivKey(addr)
			if err != nil {
				continue
			}
			in := wallet.TransactionInput{
				LinkedAddress: addr,
				OutpointIndex: u.Vout,
				OutpointHash:  h.CloneBytes(),
				Value:         int64(u.Amount),
			}
			hdKey := hd.NewExtendedKey(w.params.HDPrivateKeyID[:], key.PrivKey.Serialize(), make([]byte, 32), make([]byte, 4), 0, 0, true)
			transactionID, err := w.SweepAddress([]wallet.TransactionInput{in}, nil, hdKey, nil, wallet.FEE_BUMP)
			if err != nil {
				return nil, err
			}
			return transactionID, nil
		}
	}
	return nil, spvwallet.BumpFeeNotFoundError
}

// CreateMultisigSignature creates a multisig signature given the transaction inputs and outputs and the keys
func (w *RPCWallet) CreateMultisigSignature(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, key *hd.ExtendedKey, redeemScript []byte, feePerByte uint64) ([]wallet.Signature, error) {
	var sigs []wallet.Signature
	tx := wire.NewMsgTx(1)
	for _, in := range ins {
		ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
		if err != nil {
			return sigs, err
		}
		outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
		input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
		tx.TxIn = append(tx.TxIn, input)
	}
	for _, out := range outs {
		scriptPubKey, err := w.AddressToScript(out.Address)
		if err != nil {
			return sigs, err
		}

		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}

	// Subtract fee
	txType := spvwallet.P2SH_2of3_Multisig
	_, err := spvwallet.LockTimeFromRedeemScript(redeemScript)
	if err == nil {
		txType = spvwallet.P2SH_Multisig_Timelock_2Sigs
	}
	estimatedSize := spvwallet.EstimateSerializeSize(len(ins), tx.TxOut, false, txType)
	fee := estimatedSize * int(feePerByte)
	if len(tx.TxOut) > 0 {
		feePerOutput := fee / len(tx.TxOut)
		for _, output := range tx.TxOut {
			output.Value -= int64(feePerOutput)
		}
	}

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	signingKey, err := key.ECPrivKey()
	if err != nil {
		return sigs, err
	}

	for i := range tx.TxIn {
		sig, err := txscript.RawTxInSignature(tx, i, redeemScript, txscript.SigHashAll, signingKey)
		if err != nil {
			continue
		}
		bs := wallet.Signature{InputIndex: uint32(i), Signature: sig}
		sigs = append(sigs, bs)
	}
	return sigs, nil
}

// EstimateFee estimates the fee of a transaction
func (w *RPCWallet) EstimateFee(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, feePerByte uint64) uint64 {
	tx := new(wire.MsgTx)
	for _, out := range outs {
		scriptPubKey, _ := w.AddressToScript(out.Address)
		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}
	estimatedSize := spvwallet.EstimateSerializeSize(len(ins), tx.TxOut, false, spvwallet.P2PKH)
	fee := estimatedSize * int(feePerByte)
	return uint64(fee)
}

// EstimateSpendFee builds a spend transaction for the amount and return the transaction fee
func (w *RPCWallet) EstimateSpendFee(amount int64, feeLevel wallet.FeeLevel) (uint64, error) {
	// Since this is an estimate we can use a dummy output address. Let's use a long one so we don't under estimate.
	addr, err := btc.DecodeAddress("PARPpSkk5wpji6kE2y9YxHGZ9k96wZPfin", w.params)
	if err != nil {
		return 0, err
	}
	tx, err := w.buildTx(amount, addr, feeLevel, nil)
	if err != nil {
		return 0, err
	}
	var outval int64
	for _, output := range tx.TxOut {
		outval += output.Value
	}
	var inval int64
	utxos, err := w.txstore.Utxos().GetAll()
	if err != nil {
		return 0, err
	}
	for _, input := range tx.TxIn {
		for _, utxo := range utxos {
			if utxo.Op.Hash.IsEqual(&input.PreviousOutPoint.Hash) && utxo.Op.Index == input.PreviousOutPoint.Index {
				inval += utxo.Value
				break
			}
		}
	}
	if inval < outval {
		return 0, errors.New("Error building transaction: inputs less than outputs")
	}
	return uint64(inval - outval), err
}

func (w *RPCWallet) gatherCoins() map[coinset.Coin]*hd.ExtendedKey {
	w.rpcLock.Lock()
	height, _ := w.rpcClient.GetBlockCount()
	w.rpcLock.Unlock()
	utxos, _ := w.txstore.Utxos().GetAll()
	m := make(map[coinset.Coin]*hd.ExtendedKey)
	for _, u := range utxos {
		if u.WatchOnly {
			continue
		}
		var confirmations int32
		if u.AtHeight > 0 {
			confirmations = int32(height) - u.AtHeight
		}
		c := spvwallet.NewCoin(u.Op.Hash.CloneBytes(), u.Op.Index, btc.Amount(u.Value), int64(confirmations), u.ScriptPubkey)
		addr, err := w.ScriptToAddress(u.ScriptPubkey)
		if err != nil {
			continue
		}
		key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
		if err != nil {
			continue
		}
		m[c] = key
	}
	return m
}

func (w *RPCWallet) buildTx(amount int64, addr btc.Address, feeLevel wallet.FeeLevel, optionalOutput *wire.TxOut) (*wire.MsgTx, error) {
	// Check for dust
	script, _ := txscript.PayToAddrScript(addr)
	if txrules.IsDustAmount(btc.Amount(amount), len(script), txrules.DefaultRelayFeePerKb) {
		return nil, wallet.ErrorDustAmount
	}

	var additionalPrevScripts map[wire.OutPoint][]byte
	var additionalKeysByAddress map[string]*btc.WIF
	var inVals map[wire.OutPoint]int64

	// Create input source
	coinMap := w.gatherCoins()
	coins := make([]coinset.Coin, 0, len(coinMap))
	for k := range coinMap {
		coins = append(coins, k)
	}
	inputSource := func(target btc.Amount) (total btc.Amount, inputs []*wire.TxIn, amounts []btc.Amount, scripts [][]byte, err error) {
		coinSelector := coinset.MaxValueAgeCoinSelector{MaxInputs: 10000, MinChangeAmount: btc.Amount(0)}
		coins, err := coinSelector.CoinSelect(target, coins)
		if err != nil {
			return total, inputs, []btc.Amount{}, scripts, errors.New("insuffient funds")
		}
		additionalPrevScripts = make(map[wire.OutPoint][]byte)
		inVals = make(map[wire.OutPoint]int64)
		additionalKeysByAddress = make(map[string]*btc.WIF)
		for _, c := range coins.Coins() {
			total += c.Value()
			outpoint := wire.NewOutPoint(c.Hash(), c.Index())
			in := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
			inputs = append(inputs, in)
			additionalPrevScripts[*outpoint] = c.PkScript()
			key := coinMap[c]
			addr, err := key.Address(w.params)
			if err != nil {
				continue
			}
			privKey, err := key.ECPrivKey()
			if err != nil {
				continue
			}
			wif, _ := btc.NewWIF(privKey, w.params, true)
			additionalKeysByAddress[addr.EncodeAddress()] = wif
			val := c.Value()
			sat := val.ToUnit(btc.AmountSatoshi)
			inVals[*outpoint] = int64(sat)
		}
		return total, inputs, []btc.Amount{}, scripts, nil
	}

	// Get the fee per kilobyte
	feePerKB := int64(w.GetFeePerByte(feeLevel)) * 1000

	// outputs
	out := wire.NewTxOut(amount, script)

	// Create change source
	changeSource := func() ([]byte, error) {
		addr := w.CurrentAddress(wallet.INTERNAL)
		script, err := txscript.PayToAddrScript(addr)
		if err != nil {
			return []byte{}, err
		}
		return script, nil
	}

	outputs := []*wire.TxOut{out}
	if optionalOutput != nil {
		outputs = append(outputs, optionalOutput)
	}
	authoredTx, err := spvwallet.NewUnsignedTransaction(outputs, btc.Amount(feePerKB), inputSource, changeSource)
	if err != nil {
		return nil, err
	}

	// BIP 69 sorting
	txsort.InPlaceSort(authoredTx.Tx)

	// Sign tx
	getKey := txscript.KeyClosure(func(addr btc.Address) (*btcec.PrivateKey, bool, error) {
		addrStr := addr.EncodeAddress()
		wif := additionalKeysByAddress[addrStr]
		return wif.PrivKey, wif.CompressPubKey, nil
	})
	getScript := txscript.ScriptClosure(func(
		addr btc.Address) ([]byte, error) {
		return []byte{}, nil
	})
	for i, txIn := range authoredTx.Tx.TxIn {
		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
		script, err := txscript.SignTxOutput(w.params,
			authoredTx.Tx, i, prevOutScript, txscript.SigHashAll, getKey,
			getScript, txIn.SignatureScript)
		if err != nil {
			return nil, errors.New("Failed to sign transaction")
		}
		txIn.SignatureScript = script
	}
	return authoredTx.Tx, nil
}

// GenerateMultisigScript generates a script representing a multisig wallet
func (w *RPCWallet) GenerateMultisigScript(keys []hd.ExtendedKey, threshold int, timeout time.Duration, timeoutKey *hd.ExtendedKey) (addr btc.Address, redeemScript []byte, err error) {
	if uint32(timeout.Hours()) > 0 && timeoutKey == nil {
		return nil, nil, errors.New("Timeout key must be non nil when using an escrow timeout")
	}

	if len(keys) < threshold {
		return nil, nil, fmt.Errorf("unable to generate multisig script with "+
			"%d required signatures when there are only %d public "+
			"keys available", threshold, len(keys))
	}

	var ecKeys []*btcec.PublicKey
	for _, key := range keys {
		ecKey, err := key.ECPubKey()
		if err != nil {
			return nil, nil, err
		}
		ecKeys = append(ecKeys, ecKey)
	}

	builder := txscript.NewScriptBuilder()
	if uint32(timeout.Hours()) == 0 {

		builder.AddInt64(int64(threshold))
		for _, key := range ecKeys {
			builder.AddData(key.SerializeCompressed())
		}
		builder.AddInt64(int64(len(ecKeys)))
		builder.AddOp(txscript.OP_CHECKMULTISIG)

	} else {
		ecKey, err := timeoutKey.ECPubKey()
		if err != nil {
			return nil, nil, err
		}
		sequenceLock := blockchain.LockTimeToSequence(false, uint32(timeout.Hours()*6))
		builder.AddOp(txscript.OP_IF)
		builder.AddInt64(int64(threshold))
		for _, key := range ecKeys {
			builder.AddData(key.SerializeCompressed())
		}
		builder.AddInt64(int64(len(ecKeys)))
		builder.AddOp(txscript.OP_CHECKMULTISIG)
		builder.AddOp(txscript.OP_ELSE).
			AddInt64(int64(sequenceLock)).
			AddOp(txscript.OP_CHECKSEQUENCEVERIFY).
			AddOp(txscript.OP_DROP).
			AddData(ecKey.SerializeCompressed()).
			AddOp(txscript.OP_CHECKSIG).
			AddOp(txscript.OP_ENDIF)
	}
	redeemScript, err = builder.Script()
	if err != nil {
		return nil, nil, err
	}

	addr, err = btc.NewAddressScriptHash(redeemScript, w.params)
	if err != nil {
		return nil, nil, err
	}
	return addr, redeemScript, nil
}

// Multisign signs a multisig transaction
func (w *RPCWallet) Multisign(ins []wallet.TransactionInput, outs []wallet.TransactionOutput, sigs1 []wallet.Signature, sigs2 []wallet.Signature, redeemScript []byte, feePerByte uint64, broadcast bool) ([]byte, error) {
	tx := wire.NewMsgTx(1)
	for _, in := range ins {
		ch, err := chainhash.NewHashFromStr(hex.EncodeToString(in.OutpointHash))
		if err != nil {
			return nil, err
		}
		outpoint := wire.NewOutPoint(ch, in.OutpointIndex)
		input := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
		tx.TxIn = append(tx.TxIn, input)
	}
	for _, out := range outs {
		scriptPubKey, err := w.AddressToScript(out.Address)
		if err != nil {
			return nil, err
		}
		output := wire.NewTxOut(out.Value, scriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}

	// Subtract fee
	txType := spvwallet.P2SH_2of3_Multisig
	_, err := spvwallet.LockTimeFromRedeemScript(redeemScript)
	if err == nil {
		txType = spvwallet.P2SH_Multisig_Timelock_2Sigs
	}
	estimatedSize := spvwallet.EstimateSerializeSize(len(ins), tx.TxOut, false, txType)
	fee := estimatedSize * int(feePerByte)
	if len(tx.TxOut) > 0 {
		feePerOutput := fee / len(tx.TxOut)
		for _, output := range tx.TxOut {
			output.Value -= int64(feePerOutput)
		}
	}

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	// Check if time locked
	var timeLocked bool
	if redeemScript[0] == txscript.OP_IF {
		timeLocked = true
	}

	for i, input := range tx.TxIn {
		var sig1 []byte
		var sig2 []byte
		for _, sig := range sigs1 {
			if int(sig.InputIndex) == i {
				sig1 = sig.Signature
				break
			}
		}
		for _, sig := range sigs2 {
			if int(sig.InputIndex) == i {
				sig2 = sig.Signature
				break
			}
		}
		builder := txscript.NewScriptBuilder()
		builder.AddOp(txscript.OP_0)
		builder.AddData(sig1)
		builder.AddData(sig2)

		if timeLocked {
			builder.AddOp(txscript.OP_1)
		}

		builder.AddData(redeemScript)
		scriptSig, err := builder.Script()
		if err != nil {
			return nil, err
		}
		input.SignatureScript = scriptSig
	}
	// broadcast
	if broadcast {
		w.Broadcast(tx)
	}
	var buf bytes.Buffer
	tx.BtcEncode(&buf, wire.ProtocolVersion, wire.BaseEncoding)
	return buf.Bytes(), nil
}

// Spend spends an amount from an address with a given fee level
func (w *RPCWallet) Spend(amount int64, addr btc.Address, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
	tx, err := w.buildTx(amount, addr, feeLevel, nil)
	if err != nil {
		return nil, err
	}
	// Broadcast
	err = w.Broadcast(tx)
	if err != nil {
		return nil, err
	}
	ch := tx.TxHash()
	return &ch, nil
}

// LookAheadDistance is the number of addresses to look for transactions before assuming the rest are unused
var LookAheadDistance = 5

// RetrieveTransactions fetches transactions from the rpc server and stores them into the database
func (w *RPCWallet) RetrieveTransactions() error {
	w.txstore.addrMutex.Lock()

	addrs := make([]btc.Address, len(w.txstore.adrs))

	copy(addrs, w.txstore.adrs)

	w.txstore.addrMutex.Unlock()

	// receive transactions for P2PKH and P2PK
	w.receiveTransactions(addrs, false)

	// receive transactions for P2SH
	log.Debugf("extracting P2SH script addresses")
	scriptAddresses := make([]btc.Address, len(w.txstore.watchedScripts))
	for idx, scriptBytes := range w.txstore.watchedScripts {
		_, localScriptAddress, _, err := txscript.ExtractPkScriptAddrs(scriptBytes, w.txstore.params)
		if err != nil {
			log.Debugf("adding script address (%s) to watch error (%s)", localScriptAddress, err)
			continue
		}
		if len(localScriptAddress) > 1 {
			log.Warningf("many addresses %s were exported from script", localScriptAddress)
		}
		scriptAddresses[idx] = localScriptAddress[0]
	}

	w.receiveTransactions(scriptAddresses, false)
	return nil
}

func (w *RPCWallet) receiveTransactions(addrs []btc.Address, lookAhead bool) {
	numEmptyAddrs := 0

	for i := range addrs {
		log.Debugf("fetching transactions for address %s", addrs[i].String())
		w.rpcLock.Lock()
		txs, err := w.rpcClient.SearchRawTransactionsVerbose(addrs[i], 0, 1000000, false, false, []string{})
		w.rpcLock.Unlock()
		if err != nil {
			log.Errorf("fetching transactions for address %s failed with error: %s", addrs[i].String(), err)
			continue
		}

		if lookAhead {
			if len(txs) == 0 {
				numEmptyAddrs++
			}

			if numEmptyAddrs >= LookAheadDistance {
				return
			}
		}

		for t := range txs {
			log.Debugf("block hash %s\n", txs[t].BlockHash)

			hash, err := chainhash.NewHashFromStr(txs[t].BlockHash)
			if err != nil {
				log.Error(err)
				continue
			}

			w.rpcLock.Lock()
			block, err := w.rpcClient.GetBlockVerbose(hash)
			w.rpcLock.Unlock()

			if err != nil {
				log.Error(err)
				continue
			}

			transactionBytes, err := hex.DecodeString(txs[t].Hex)
			if err != nil {
				log.Error(err)
				continue
			}

			transaction := wire.MsgTx{}
			err = transaction.BtcDecode(bytes.NewReader(transactionBytes), 1, wire.BaseEncoding)
			if err != nil {
				log.Error(err)
				continue
			}

			w.txstore.Ingest(&transaction, int32(block.Height), time.Unix(block.Time, 0))

			log.Debugf("ingested tx hash %s", transaction.TxHash().String())
		}
	}
}