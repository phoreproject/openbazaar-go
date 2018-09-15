package bchutil

import (
	"github.com/phoreproject/btcd/txscript"
	"github.com/phoreproject/btcutil"
	"errors"
)

func PayToAddrScript(addr btcutil.Address) ([]byte, error) {
	var script []byte
	var err error
	script, err = txscript.PayToAddrScript(addr)
	if err == nil {
		return script, nil
	}
	script, err = cashPayToAddrScript(addr)
	if err == nil {
		return script, nil
	}
	script, err = bitpayPayToAddrScript(addr)
	if err == nil {
		return script, nil
	}
	return script, errors.New("Unrecognized address format")
}