package zerocoin

import (
	"github.com/phoreproject/btcutil"
)

// Denomination represents a Zerocoin denomination
type Denomination int64

const (
	// DenomError is the default error zerocoin denomination
	DenomError Denomination = 0

	// DenomOne represents a zerocoin with denomination 1
	DenomOne Denomination = 1

	// DenomFive represents a zerocoin with denomination 5
	DenomFive Denomination = 5

	// DenomTen represents a zerocoin with denomination 10
	DenomTen Denomination = 10

	// DenomFifty represents a zerocoin with denomination 50
	DenomFifty Denomination = 50

	// DenomOneHundred represents a zerocoin with denomination 100
	DenomOneHundred Denomination = 100

	// DenomFiveHundred represents a zerocoin with denomination 500
	DenomFiveHundred Denomination = 500

	// DenomOneThousand represents a zerocoin with denomination 1000
	DenomOneThousand Denomination = 1000

	// DenomFiveThousand represents a zerocoin with denomination 5000
	DenomFiveThousand Denomination = 5000
)

// ZerocoinDenominations are the valid denominations of Zerocoin
var ZerocoinDenominations = [...]Denomination{
	DenomOne,
	DenomFive,
	DenomTen,
	DenomFifty,
	DenomOneHundred,
	DenomFiveHundred,
	DenomOneThousand,
	DenomFiveThousand,
}

// DenominationToAmount gets the denomination of an amount
func DenominationToAmount(d Denomination) int64 {
	return int64(d) * btcutil.SatoshiPerBitcoin
}

// AmountToZerocoinDenomination converts an amount to a zerocoin denomination
func AmountToZerocoinDenomination(amount int64) Denomination {
	switch amount {
	case 1 * btcutil.SatoshiPerBitcoin:
		return DenomOne
	case 5 * btcutil.SatoshiPerBitcoin:
		return DenomFive
	case 10 * btcutil.SatoshiPerBitcoin:
		return DenomTen
	case 50 * btcutil.SatoshiPerBitcoin:
		return DenomFifty
	case 100 * btcutil.SatoshiPerBitcoin:
		return DenomOneHundred
	case 500 * btcutil.SatoshiPerBitcoin:
		return DenomFiveHundred
	case 1000 * btcutil.SatoshiPerBitcoin:
		return DenomOneThousand
	case 5000 * btcutil.SatoshiPerBitcoin:
		return DenomFiveThousand
	default:
		return DenomError
	}
}
