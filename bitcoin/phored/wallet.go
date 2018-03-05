package phored

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
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
	repoPath         string
	masterPrivateKey *hd.ExtendedKey
	masterPublicKey  *hd.ExtendedKey
	rpcClient        *rpcclient.Client
	started          bool
	keyManager       *spvwallet.KeyManager
	mnemonic         string
	DB               *TxStore
	connCfg          *rpcclient.ConnConfig
	notifications    *NotificationListener
	rpcBasePath      string
}

// NewRPCWallet creates a new wallet given
func NewRPCWallet(mnemonic string, params *chaincfg.Params, repoPath string, DB wallet.Datastore, host string) *RPCWallet {
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

	seed := b39.NewSeed(mnemonic, "")

	mPrivKey, _ := hd.NewMaster(seed, params)
	mPubKey, _ := mPrivKey.Neuter()

	keyManager, _ := spvwallet.NewKeyManager(DB.Keys(), params, mPrivKey)

	txstore, _ := NewTxStore(params, DB, keyManager)

	w := RPCWallet{
		params:           params,
		repoPath:         repoPath,
		masterPrivateKey: mPrivKey,
		masterPublicKey:  mPubKey,
		keyManager:       keyManager,
		DB:               txstore,
		connCfg:          connCfg,
		rpcBasePath:      host,
	}
	return &w
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

	log.Info("Connected to phored")
	w.started = true
}

// CurrencyCode returns the currency code of the wallet
func (w *RPCWallet) CurrencyCode() string {
	return "phr"
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
	i, _ := w.DB.Keys().GetUnused(purpose)
	key, _ := w.keyManager.GenerateChildKey(purpose, uint32(i[1]))
	addr, _ := key.Address(w.params)
	w.DB.Keys().MarkKeyAsUsed(addr.ScriptAddress())
	w.DB.PopulateAdrs()
	return btc.Address(addr)
}

// DecodeAddress decodes an address string to an address using the wallet's chain parameters
func (w *RPCWallet) DecodeAddress(addr string) (btc.Address, error) {
	return btc.DecodeAddress(addr, w.params)
}

// ScriptToAddress converts a script to an address
func (w *RPCWallet) ScriptToAddress(script []byte) (btc.Address, error) {
	_, addrs, _, err := txscript.ExtractPkScriptAddrs(script, w.params)
	if err != nil {
		return nil, err
	}
	if len(addrs) == 0 {
		return nil, errors.New("unknown script")
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
	utxos, _ := w.DB.Utxos().GetAll()
	stxos, _ := w.DB.Stxos().GetAll()
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
	return w.DB.Txns().GetAll(false)
}

// GetTransaction returns the transaction given by a transaction hash
func (w *RPCWallet) GetTransaction(txid chainhash.Hash) (wallet.Txn, error) {
	_, txn, err := w.DB.Txns().Get(txid)
	return txn, err
}

// GetConfirmations returns the number of confirmations and the block number where the transaction was confirmed
func (w *RPCWallet) GetConfirmations(txid chainhash.Hash) (uint32, uint32, error) {
	_, txn, err := w.DB.Txns().Get(txid)
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
				}
				return w.checkIfStxoIsConfirmed(stxo.Utxo, stxos)
			} else if stxo.Utxo.IsEqual(&utxo) {
				if stxo.Utxo.AtHeight > 0 {
					return true
				}
				return false
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
	w.DB.listeners = append(w.DB.listeners, callback)
}

// ChainTip returns the tip of the active blockchain
func (w *RPCWallet) ChainTip() (uint32, chainhash.Hash) {
	ch, err := w.rpcClient.GetBestBlockHash()
	if err != nil {
		return 0, chainhash.Hash{}
	}

	height, err := w.rpcClient.GetBlockCount()
	return uint32(height), *ch
}

// AddWatchedScript adds a script to be watched
func (w *RPCWallet) AddWatchedScript(script []byte) error {
	err := w.DB.WatchedScripts().Put(script)
	w.DB.PopulateAdrs()

	w.notifications.updateFilterAndSend()
	return err
}

// Close closes the rpc wallet connection
func (w *RPCWallet) Close() {
	if w.started {
		log.Info("Disconnecting from peers and shutting down")
		w.rpcClient.Shutdown()
		w.started = false
	}
}

// ReSyncBlockchain resyncs the addresses used by the SPV wallet
func (w *RPCWallet) ReSyncBlockchain(fromDate time.Time) {
	w.DB.PopulateAdrs()
}

// SweepAddress sweeps any UTXOs from an address in a single transaction
func (w *RPCWallet) SweepAddress(utxos []wallet.Utxo, address *btc.Address, key *hd.ExtendedKey, redeemScript *[]byte, feeLevel wallet.FeeLevel) (*chainhash.Hash, error) {
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
	for _, u := range utxos {
		val += u.Value
		in := wire.NewTxIn(&u.Op, []byte{}, [][]byte{})
		inputs = append(inputs, in)
		additionalPrevScripts[u.Op] = u.ScriptPubkey
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
	estimatedSize := spvwallet.EstimateSerializeSize(len(utxos), []*wire.TxOut{out}, false, txType)

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
			for _, txIn := range tx.TxIn {
				locktime, err := spvwallet.LockTimeFromRedeemScript(*redeemScript)
				if err != nil {
					return nil, err
				}
				txIn.Sequence = locktime
			}
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
			sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, utxos[i].Value, *redeemScript, txscript.SigHashAll, privKey)
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
	_, err := w.DB.Ingest(tx, 0)
	if err != nil {
		return err
	}

	// make an inv message instead of a tx message to be polite
	txid := tx.TxHash()
	iv1 := wire.NewInvVect(wire.InvTypeTx, &txid)
	invMsg := wire.NewMsgInv()
	err = invMsg.AddInvVect(iv1)
	if err != nil {
		return err
	}

	_, err = w.rpcClient.SendRawTransaction(tx, false)
	if err != nil {
		log.Error(err)
	}

	w.notifications.updateFilterAndSend()
	return nil
}

// ErrBumpFeeAlreadyConfirmed throws when a transaction is already confirmed and the fee cannot be bumped
var ErrBumpFeeAlreadyConfirmed = errors.New("Transaction is confirmed, cannot bump fee")

// ErrBumpFeeTransactionDead throws when a transaction is dead and the fee cannot be dumped
var ErrBumpFeeTransactionDead = errors.New("Cannot bump fee of dead transaction")

// ErrBumpFeeNotFound throws when a transaction has been spent or doesn't exist
var ErrBumpFeeNotFound = errors.New("Transaction either doesn't exist or has already been spent")

// BumpFee attempts to bump the fee for a transaction
func (w *RPCWallet) BumpFee(txid chainhash.Hash) (*chainhash.Hash, error) {
	_, txn, err := w.DB.Txns().Get(txid)
	if err != nil {
		return nil, err
	}
	if txn.Height > 0 {
		return nil, ErrBumpFeeAlreadyConfirmed
	}
	if txn.Height < 0 {
		return nil, ErrBumpFeeTransactionDead
	}
	utxos, _ := w.DB.Utxos().GetAll()
	for _, u := range utxos {
		if u.Op.Hash.IsEqual(&txid) && u.AtHeight == 0 {
			addr, err := w.ScriptToAddress(u.ScriptPubkey)
			if err != nil {
				return nil, err
			}
			key, err := w.keyManager.GetKeyForScript(addr.ScriptAddress())
			if err != nil {
				return nil, err
			}
			transactionID, err := w.SweepAddress([]wallet.Utxo{u}, nil, key, nil, wallet.FEE_BUMP)
			if err != nil {
				return nil, err
			}
			return transactionID, nil
		}
	}
	return nil, ErrBumpFeeNotFound
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
		output := wire.NewTxOut(out.Value, out.ScriptPubKey)
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

	hashes := txscript.NewTxSigHashes(tx)
	for i := range tx.TxIn {
		sig, err := txscript.RawTxInWitnessSignature(tx, hashes, i, ins[i].Value, redeemScript, txscript.SigHashAll, signingKey)
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
		output := wire.NewTxOut(out.Value, out.ScriptPubKey)
		tx.TxOut = append(tx.TxOut, output)
	}
	estimatedSize := spvwallet.EstimateSerializeSize(len(ins), tx.TxOut, false, spvwallet.P2PKH)
	fee := estimatedSize * int(feePerByte)
	return uint64(fee)
}

// EstimateSpendFee builds a spend transaction for the amount and return the transaction fee
func (w *RPCWallet) EstimateSpendFee(amount int64, feeLevel wallet.FeeLevel) (uint64, error) {
	// Since this is an estimate we can use a dummy output address. Let's use a long one so we don't under estimate.
	addr, err := btc.DecodeAddress("bc1qxtq7ha2l5qg70atpwp3fus84fx3w0v2w4r2my7gt89ll3w0vnlgspu349h", w.params)
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
	utxos, err := w.DB.Utxos().GetAll()
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
	height, _ := w.rpcClient.GetBlockCount()
	utxos, _ := w.DB.Utxos().GetAll()
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

	// Create input source
	coinMap := w.gatherCoins()
	coins := make([]coinset.Coin, 0, len(coinMap))
	for k := range coinMap {
		coins = append(coins, k)
	}
	inputSource := func(target btc.Amount) (total btc.Amount, inputs []*wire.TxIn, scripts [][]byte, err error) {
		coinSelector := coinset.MaxValueAgeCoinSelector{MaxInputs: 10000, MinChangeAmount: btc.Amount(0)}
		coins, err := coinSelector.CoinSelect(target, coins)
		if err != nil {
			return total, inputs, scripts, wallet.ErrorInsuffientFunds
		}
		additionalPrevScripts = make(map[wire.OutPoint][]byte)
		additionalKeysByAddress = make(map[string]*btc.WIF)
		for _, c := range coins.Coins() {
			total += c.Value()
			outpoint := wire.NewOutPoint(c.Hash(), c.Index())
			in := wire.NewTxIn(outpoint, []byte{}, [][]byte{})
			in.Sequence = 0 // Opt-in RBF so we can bump fees
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
		}
		return total, inputs, scripts, nil
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

	witnessProgram := sha256.Sum256(redeemScript)

	addr, err = btc.NewAddressWitnessScriptHash(witnessProgram[:], w.params)
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
		output := wire.NewTxOut(out.Value, out.ScriptPubKey)
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

		witness := wire.TxWitness{[]byte{}, sig1, sig2}

		if timeLocked {
			witness = append(witness, []byte{0x01})
		}
		witness = append(witness, redeemScript)
		input.Witness = witness
	}
	// broadcast
	if broadcast {
		w.Broadcast(tx)
	}
	var buf bytes.Buffer
	tx.BtcEncode(&buf, wire.ProtocolVersion, wire.WitnessEncoding)
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
	w.DB.addrMutex.Lock()

	addrs := make([]btc.Address, len(w.DB.adrs))

	copy(addrs, w.DB.adrs)

	w.DB.addrMutex.Unlock()

	numEmptyAddrs := 0

	for i := range addrs {
		log.Debugf("fetching transactions for address %s", addrs[i].String())
		txs, err := w.rpcClient.SearchRawTransactionsVerbose(addrs[i], 0, 1000000, false, false, []string{})
		if err != nil {
			return err
		}

		if len(txs) == 0 {
			numEmptyAddrs++
		}

		if numEmptyAddrs >= LookAheadDistance {
			return nil
		}

		for t := range txs {
			log.Debug(txs[t].BlockHash)

			hash, err := chainhash.NewHashFromStr(txs[t].BlockHash)

			if err != nil {
				return err
			}

			block, err := w.rpcClient.GetBlockVerbose(hash)

			if err != nil {
				return err
			}

			transactionBytes, err := hex.DecodeString(txs[t].Hex)
			if err != nil {
				return err
			}

			transaction := wire.MsgTx{}
			err = transaction.BtcDecode(bytes.NewReader(transactionBytes), 1, wire.BaseEncoding)
			if err != nil {
				return err
			}

			w.DB.Ingest(&transaction, int32(block.Height))

			log.Debugf("ingested tx hash %s", transaction.TxHash().String())
		}
	}
	return nil
}
