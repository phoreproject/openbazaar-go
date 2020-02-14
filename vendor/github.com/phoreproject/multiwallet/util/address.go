package util

import (
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/cpacia/bchutil"
	liteaddr "github.com/phoreproject/multiwallet/litecoin/address"
	zaddr "github.com/phoreproject/multiwallet/zcash/address"

	"errors"
)

func DecodeAddress(address string, params *chaincfg.Params) (btcutil.Address, error) {
	if len(address) == 0 {
		return nil, errors.New("unknown address")
	}
	if addr, err := btcutil.DecodeAddress(address, params); err == nil {
		return addr, nil
	}
	if addr, err := bchutil.DecodeAddress(address, params); err == nil {
		return addr, nil
	}
	if addr, err := liteaddr.DecodeAddress(address, params); err == nil {
		return addr, nil
	}
	if addr, err := zaddr.DecodeAddress(address, params); err == nil {
		return addr, nil
	}
	return nil, errors.New("unknown address")
}
