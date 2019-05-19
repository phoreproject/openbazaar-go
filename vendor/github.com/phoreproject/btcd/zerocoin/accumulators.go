package zerocoin

import (
	"encoding/binary"
	"errors"
	"math"
	"math/big"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
)

// Accumulator represents an RSA-based accumulator.
type Accumulator struct {
	params       *AccumulatorAndProofParams
	value        *big.Int
	denomination Denomination
}

// NewAccumulator initializes a new empty accumulator with given parameters
// and denomination.
func NewAccumulator(params *AccumulatorAndProofParams, d Denomination) (*Accumulator, error) {
	if !params.Initialized {
		return nil, errors.New("accumulator and proof params must be initialized")
	}
	return &Accumulator{
		denomination: d,
		value:        params.AccumulatorBase,
	}, nil
}

// NewAccumulatorWithValue initializes an accumulator with given zerocoin params,
// denomination, and with a preset value.
func NewAccumulatorWithValue(params *AccumulatorAndProofParams, d Denomination, value *big.Int) (*Accumulator, error) {
	a := &Accumulator{}

	a.params = params
	a.denomination = d

	if !a.params.Initialized {
		return nil, errors.New("zerocoin parameters must be initialized")
	}

	if value.Cmp(big.NewInt(0)) != 0 {
		a.value = value
	} else {
		a.value = params.AccumulatorBase
	}
	return a, nil
}

// Increment adds a value to the accumulator
func (a Accumulator) Increment(value *big.Int) {
	a.value = new(big.Int).Exp(a.value, value, a.params.AccumulatorModulus)
}

// Accumulate a given coin if it is valid and the denomination matches.
func (a Accumulator) Accumulate(coin *PublicCoin) error {
	if a.value.Cmp(big.NewInt(0)) == 0 {
		return errors.New("accumulator is not initialized")
	}

	if a.denomination != coin.denomination {
		return errors.New("accumulator does not match the coin being accumulated")
	}

	if coin.Validate() {
		a.Increment(coin.value)
	} else {
		return errors.New("coin is not valid")
	}
	return nil
}

// AccumulatorWitness is a witness that a public coin is
// in the accumulation of a set of coins.
type AccumulatorWitness struct {
	witness Accumulator
	element PublicCoin
}

// ResetValue resets the value of the accumulator witness to a
// given checkpoint and public coin.
func (a AccumulatorWitness) ResetValue(checkpoint *Accumulator, coin PublicCoin) {
	a.witness.value = checkpoint.value
	a.element = coin
}

// AddElement adds a public coin to the accumulator.
func (a AccumulatorWitness) AddElement(coin PublicCoin) {
	if a.element.value != coin.value {
		a.witness.Accumulate(&coin)
	}
}

// VerifyWitness verifies that a witness matches the accumulator.
func (a AccumulatorWitness) VerifyWitness(acc *Accumulator, p *PublicCoin) (bool, error) {
	temp, err := NewAccumulatorWithValue(a.witness.params, a.witness.denomination, a.witness.value)
	if err != nil {
		return false, err
	}

	temp.Accumulate(&a.element)

	return temp.value == acc.value && a.element.Equal(*p), nil
}

// AccumulatorMap each denomination of accumulator.
type AccumulatorMap struct {
	accs   map[Denomination]*Accumulator
	params *Params
}

// NewAccumulatorMap creates a new map of
func NewAccumulatorMap(params *Params) (*AccumulatorMap, error) {
	acc := &AccumulatorMap{}
	acc.params = params
	acc.accs = make(map[Denomination]*Accumulator)
	for _, d := range ZerocoinDenominations {
		a, err := NewAccumulator(params.AccumulatorParams, d)
		if err != nil {
			return nil, err
		}
		acc.accs[d] = a
	}
	return acc, nil
}

// Reset resets the accumulators to their default values.
func (a *AccumulatorMap) Reset() error {
	// construct a blank acc map given the parameters
	a1, err := NewAccumulatorMap(a.params)
	if err != nil {
		return err
	}

	a = a1
	return nil
}

// Read reads the accumulator value for a certain denomination
func (a *AccumulatorMap) Read(d Denomination) *big.Int {
	return a.accs[d].value
}

// Accumulate adds a zerocoin to the accumulator of its denomination
func (a *AccumulatorMap) Accumulate(p PublicCoin) error {
	denom := p.denomination

	if denom == DenomError {
		return errors.New("attempted to accumulate an invalid pubcoin")
	}

	return a.accs[denom].Accumulate(&p)
}

// SetValue sets the value of one of the accumulators
func (a *AccumulatorMap) SetValue(d Denomination, i *big.Int) {
	a.accs[d].value = i
}

// SerializeBigNum serializes a big integer like openssl would
func SerializeBigNum(b *big.Int) []byte {
	length := uint32(math.Ceil(float64(b.BitLen()) / 8))
	var lengthBytes [4]byte
	binary.BigEndian.PutUint32(lengthBytes[:], length)
	return append(lengthBytes[:], b.Bytes()...)
}

// DeserializeBigNum deserializes a big integer similar to OpenSSL
// with a big endian 32-bit length followed by a big endian representation
// of the number.
func DeserializeBigNum(b []byte) (*big.Int, error) {
	if len(b) < 4 {
		return nil, errors.New("invalid big num length")
	}

	lengthBytes := b[:4]
	length := binary.BigEndian.Uint32(lengthBytes)
	if len(b) != int(4+length) {
		return nil, errors.New("invalid big num length")
	}

	i := new(big.Int)
	i.SetBytes(b[4:])
	return i, nil
}

// GetChecksum calculates the checksum of a zerocoin accumulator
// value.
func GetChecksum(b *big.Int) uint32 {
	h := chainhash.HashH(SerializeBigNum(b))
	n := HashToBig(&h).Bytes()
	lower32 := n[len(n)-5:]
	return binary.BigEndian.Uint32(lower32)
}

// GetCheckpoint gets the checksum of all of the accumulators
// contained in the map.
func (a *AccumulatorMap) GetCheckpoint() *big.Int {
	checkpoint := big.NewInt(0)

	for _, d := range ZerocoinDenominations {
		value := a.Read(d)

		checkpoint.Lsh(checkpoint, 32)
		checkpoint.Or(checkpoint, big.NewInt(int64(GetChecksum(value))))
	}
	return checkpoint
}

// SetZerocoinParams sets new parameters and resets the accumulator
func (a *AccumulatorMap) SetZerocoinParams(params *Params) {
	a.params = params
	a.Reset()
}
