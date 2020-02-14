package phore

import (
	"errors"
	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	btc "github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/txsort"
	"github.com/btcsuite/btcwallet/wallet/txrules"

	"github.com/phoreproject/multiwallet/util"
)

func (w *RPCWallet) buildSpendAllTx(addr btc.Address, feeLevel wi.FeeLevel) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(1)

	height, _ := w.ws.ChainTip()
	utxos, err := w.db.Utxos().GetAll()
	if err != nil {
		return nil, err
	}
	coinMap := util.GatherCoins(height, utxos, w.ScriptToAddress, w.km.GetKeyForScript)

	totalIn, _, additionalPrevScripts, additionalKeysByAddress := util.LoadAllInputs(tx, coinMap, w.params)

	// outputs
	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, err
	}

	// Get the fee
	feePerByte := int64(w.GetFeePerByte(feeLevel))
	estimatedSize := EstimateSerializeSize(1, []*wire.TxOut{wire.NewTxOut(0, script)}, false, P2PKH)
	fee := int64(estimatedSize) * feePerByte

	// Check for dust output
	if txrules.IsDustAmount(btc.Amount(totalIn-fee), len(script), txrules.DefaultRelayFeePerKb) {
		return nil, wi.ErrorDustAmount
	}

	// Build the output
	out := wire.NewTxOut(totalIn-fee, script)
	tx.TxOut = append(tx.TxOut, out)

	// BIP 69 sorting
	txsort.InPlaceSort(tx)

	// Sign
	getKey := txscript.KeyClosure(func(addr btc.Address) (*btcec.PrivateKey, bool, error) {
		addrStr := addr.EncodeAddress()
		wif, ok := additionalKeysByAddress[addrStr]
		if !ok {
			return nil, false, errors.New("key not found")
		}
		return wif.PrivKey, wif.CompressPubKey, nil
	})
	getScript := txscript.ScriptClosure(func(
		addr btc.Address) ([]byte, error) {
		return []byte{}, nil
	})
	for i, txIn := range tx.TxIn {
		prevOutScript := additionalPrevScripts[txIn.PreviousOutPoint]
		script, err := txscript.SignTxOutput(w.params,
			tx, i, prevOutScript, txscript.SigHashAll, getKey,
			getScript, txIn.SignatureScript)
		if err != nil {
			return nil, errors.New("failed to sign transaction")
		}
		txIn.SignatureScript = script
	}
	return tx, nil
}
