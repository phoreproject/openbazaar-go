package zerocoin

import (
	"errors"
	"math/big"
)

// PublicCoin is the part of the coin that is published
// to the network and contains the value of the commitment
// to a serial number and the denomination of the coin.
type PublicCoin struct {
	params       *Params
	value        *big.Int
	denomination Denomination
}

// NewPublicCoin initializes a new public coin without a
// denomination.
func NewPublicCoin(p *Params) (*PublicCoin, error) {
	if !p.Initialized {
		return nil, errors.New("Params are not initialized")
	}

	return &PublicCoin{denomination: DenomError, params: p}, nil
}

// NewPublicCoinFromValue initializes a new public coin from an
// existing value and denomination.
func NewPublicCoinFromValue(p *Params, coin *big.Int, d Denomination) (*PublicCoin, error) {
	if !p.Initialized {
		return nil, errors.New("Params are not initialized")
	}

	pub := &PublicCoin{}
	pub.value = coin
	pub.denomination = d
	return pub, nil
}

// Validate checks the validity of a public coin.
func (p PublicCoin) Validate() bool {
	if p.params.AccumulatorParams.MinCoinValue.Cmp(p.value) >= 0 {
		return false
	}
	if p.params.AccumulatorParams.MaxCoinValue.Cmp(p.value) < 0 {
		return false
	}
	if !p.value.ProbablyPrime(int(p.params.ZKPIterations)) {
		return false
	}
	return true
}

// Equal returns two if the two public coins are equal.
func (p PublicCoin) Equal(pub2 PublicCoin) bool {
	return p.value == pub2.value && p.params == pub2.params && p.denomination == pub2.denomination
}

// Value gets the value of the pubcoin
func (p *PublicCoin) Value() *big.Int {
	return p.value
}

// Denomination gets the denomination of the pubcoin
func (p *PublicCoin) Denomination() Denomination {
	return p.denomination
}

// Params gets the parameters used by the pubcoin
func (p *PublicCoin) Params() *Params {
	return p.params
}
