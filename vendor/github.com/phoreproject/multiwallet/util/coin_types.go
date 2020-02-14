package util

import "github.com/OpenBazaar/wallet-interface"

type ExtCoinType wallet.CoinType

func ExtendCoinType(coinType wallet.CoinType) ExtCoinType {
	return ExtCoinType(uint32(coinType))
}

const (
	CoinTypePhore     ExtCoinType = 444
	CoinTypePhoreTest             = 1000444
)

func (c *ExtCoinType) String() string {
	ct := wallet.CoinType(uint32(*c))
	str := ct.String()
	if str != "" {
		return str
	}

	switch *c {
	case CoinTypePhore:
		return "Phore"
	case CoinTypePhoreTest:
		return "Testnet Phore"
	default:
		return ""
	}
}

func (c *ExtCoinType) CurrencyCode() string {
	ct := wallet.CoinType(uint32(*c))
	str := ct.CurrencyCode()
	if str != "" {
		return str
	}

	switch *c {
	case CoinTypePhore:
		return "PHR"
	case CoinTypePhoreTest:
		return "TPHR"
	default:
		return ""
	}
}

func (c ExtCoinType) ToCoinType() wallet.CoinType {
	return wallet.CoinType(uint32(c))
}
