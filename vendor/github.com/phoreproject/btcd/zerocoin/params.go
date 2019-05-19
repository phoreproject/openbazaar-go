package zerocoin

import (
	"encoding/binary"
	"errors"
	"math"
	"math/big"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
)

var (
	// bigOne is 1 represented as a big.Int.  It is defined here to avoid
	// the overhead of creating it multiple times.
	bigOne = big.NewInt(1)

	// oneLsh256 is 1 shifted left 256 bits.  It is defined here to avoid
	// the overhead of creating it multiple times.
	oneLsh256 = new(big.Int).Lsh(bigOne, 256)

	// oneLsh256Minus1 should be 0x2100ffff in compact form
	oneLsh256Minus1 = new(big.Int).Sub(oneLsh256, bigOne)
)

// IntegerGroupParams is a cryptographic integer group.
// Note log_G(H) and log_H(G) must be unknown.
type IntegerGroupParams struct {
	// G is the generator for the group
	G *big.Int

	// H is another generator for the group
	H *big.Int

	Modulus     *big.Int
	GroupOrder  *big.Int
	Initialized bool
}

// AccumulatorAndProofParams are the parameters used for Zerocoin
// on the network.
type AccumulatorAndProofParams struct {
	Initialized                   bool
	AccumulatorModulus            *big.Int
	AccumulatorBase               *big.Int
	MinCoinValue                  *big.Int
	MaxCoinValue                  *big.Int
	AccumulatorPoKCommitmentGroup *IntegerGroupParams
	AccumulatorQRNCommitmentGroup *IntegerGroupParams

	// KPrime is the bit length of the challenges used in the accumulator
	// proof-of-knowledge.
	KPrime uint32

	// KDPrime is the statistical zero-knowledgeness of the accumulator
	// proof.
	KDPrime uint32
}

// Params are parameters for Zerocoin with a trusted modulus N.
type Params struct {
	Initialized       bool
	AccumulatorParams *AccumulatorAndProofParams

	// CoinCommitmentGroup is the quadratic residue group from which
	// we form a coin as a commitment to a serial number.
	CoinCommitmentGroup *IntegerGroupParams

	// One of two groups used to form a commitment to a coin. This
	// is used in the serial number proof. The order must equal the
	// modulus of CoinCommitmentGroup.
	SerialNumberSoKCommitmentGroup *IntegerGroupParams

	// Number of iterations used in the serial number proof.
	ZKPIterations uint32

	// Amount of the hash function we use for proofs.
	ZKPHashLength uint32
}

// appendAll appends all items in a list together
func appendAll(items ...[]byte) []byte {
	out := []byte{}
	for _, i := range items {
		out = append(out, i...)
	}
	return out
}

func calculateSeed(modulus *big.Int, auxString string, securityLevel uint32, groupName string) chainhash.Hash {
	var secLevelBytes [4]byte
	binary.LittleEndian.PutUint32(secLevelBytes[:], securityLevel)

	separator := []byte("||")

	modulusBytes := SerializeBigNum(modulus)

	hashInput := appendAll(modulusBytes[:], separator, secLevelBytes[:], separator, []byte(auxString), separator, []byte(groupName))

	return chainhash.HashH(hashInput)
}

// calculateGroupParamLengths will calculate the group parameter sizes
// based on a security level. Returns pLen, qLen
func calculateGroupParamLengths(maxPLen uint32, securityLevel uint32) (uint32, uint32, error) {
	pLen := uint32(0)
	qLen := uint32(0)

	if securityLevel < 80 {
		return 0, 0, errors.New("Security level must be at least 80 bits")
	} else if securityLevel == 80 {
		pLen = 1024
		qLen = 256
	} else if securityLevel <= 112 {
		pLen = 2048
		qLen = 256
	} else if securityLevel <= 128 {
		pLen = 3072
		qLen = 320
	} else {
		return 0, 0, errors.New("Security level not supported")
	}
	if pLen > maxPLen {
		return 0, 0, errors.New("Modulus size is too small for this security level")
	}
	return pLen, qLen, nil
}

// HashToBig converts a chainhash.Hash into a big.Int that can be used to
// perform math comparisons.
func HashToBig(hash *chainhash.Hash) *big.Int {
	// A Hash is in little-endian, but the big package wants the bytes in
	// big-endian, so reverse them.
	buf := *hash
	blen := len(buf)
	for i := 0; i < blen/2; i++ {
		buf[i], buf[blen-1-i] = buf[blen-1-i], buf[i]
	}

	return new(big.Int).SetBytes(buf[:])
}

// BigToHash converts a big int to a chainhash
func BigToHash(in *big.Int) (*chainhash.Hash, error) {
	if in.Cmp(oneLsh256Minus1) > 0 {
		return nil, errors.New("big int out of bounds")
	}

	buf := [chainhash.HashSize]byte{}
	bigEndianBytes := in.Bytes()
	blen := len(bigEndianBytes)

	for i := 0; i < blen/2; i++ {
		bigEndianBytes[i], bigEndianBytes[blen-1-i] = bigEndianBytes[blen-1-i], bigEndianBytes[i]
	}

	blen = len(buf)

	for i := 0; i < blen; i++ {
		if i < len(bigEndianBytes) {
			buf[i] = bigEndianBytes[i]
		} else {
			buf[i] = 0
		}
	}

	h, err := chainhash.NewHash(buf[:])
	if err != nil {
		return nil, err
	}
	return h, nil
}

// BigToLittleBytes converts a big int to a little endian bytes
func BigToLittleBytes(in *big.Int) []byte {
	bigEndianBytes := in.Bytes()
	blen := len(bigEndianBytes)

	for i := 0; i < blen/2; i++ {
		bigEndianBytes[i], bigEndianBytes[blen-1-i] = bigEndianBytes[blen-1-i], bigEndianBytes[i]
	}

	return bigEndianBytes
}

func calculateHash(in *big.Int) *big.Int {
	in.Mod(in, oneLsh256Minus1)
	b, _ := BigToHash(in)
	h := chainhash.HashH(b[:])
	return HashToBig(&h)
}

func generateIntegerFromSeed(numBits uint32, seed *chainhash.Hash) (*big.Int, uint32) {
	result := big.NewInt(0)

	iterations := int(math.Ceil((float64)(numBits) / (float64)(256)))

	for count := 0; count < iterations; count++ {
		seedCount := new(big.Int).Add(HashToBig(seed), big.NewInt(int64(count)))
		s := big.NewInt(1)
		s.Lsh(s, uint(count*256))
		r := new(big.Int)
		r.Mul(calculateHash(seedCount), s)
		result.Add(result, r)
	}

	modulo := new(big.Int).Lsh(big.NewInt(1), uint(numBits-1))

	realResult := big.NewInt(1)
	realResult.Lsh(realResult, uint(numBits-1))
	realResult.Add(realResult, result.Mod(result, modulo))

	return realResult, uint32(iterations)
}

const maxPrimeGenerationAttempts = 10000

// generateRandomPrime generates a random prime number given a bit length and input seed
// Uses the Shawe-Taylor algorithm as described in FIPS 186-3
// Appendix C.6. This is a recursive function.
func generateRandomPrime(pLen uint32, seed *chainhash.Hash) (*chainhash.Hash, uint32, *big.Int, error) {
	if pLen < 2 {
		return nil, 0, nil, errors.New("Prime length is too short")
	}

	primeGenCounter := uint32(0)

	if pLen < 33 {
		primeSeed := HashToBig(seed)

		for primeGenCounter < 4*pLen {
			h, err := BigToHash(primeSeed)
			if err != nil {
				return nil, 0, nil, err
			}
			c, iterationCount := generateIntegerFromSeed(pLen, h)

			primeSeed.Add(primeSeed, big.NewInt(int64(iterationCount+1)))
			primeSeed.Mod(primeSeed, oneLsh256Minus1)

			primeGenCounter++

			intc := c.Uint64()

			intc = (2 * uint64(math.Floor(float64(intc)/2.0))) + 1

			bigc := big.NewInt(int64(intc))

			if bigc.ProbablyPrime(40) {
				h, err := BigToHash(primeSeed)
				if err != nil {
					return nil, 0, nil, err
				}
				return h, primeGenCounter, bigc, nil
			}
		}

		return nil, 0, nil, errors.New("Unable to find prime in Shawe-Taylor algorithm")
	}
	newLength := uint32(math.Ceil(float64(pLen)/2.0)) + 1
	outSeed, primeGenCounter, c0, err := generateRandomPrime(newLength, seed)
	if err != nil {
		return nil, 0, nil, err
	}

	x, numIterations := generateIntegerFromSeed(pLen, outSeed)
	h := HashToBig(outSeed)
	h.Add(h, big.NewInt(int64(numIterations+1)))
	h.Mod(h, oneLsh256Minus1)
	outSeed, _ = BigToHash(h)

	t := x.Div(x, new(big.Int).Mul(big.NewInt(2), c0))

	for i := 0; i < maxPrimeGenerationAttempts; i++ {
		left := new(big.Int).Mul(new(big.Int).Mul(t, c0), big.NewInt(2))
		right := new(big.Int).Lsh(big.NewInt(1), uint(pLen))
		if left.Cmp(right) > 0 {
			c0doubled := new(big.Int).Mul(c0, big.NewInt(2))
			t = new(big.Int).Div(new(big.Int).Sub(new(big.Int).Lsh(bigOne, uint(pLen)), bigOne), c0doubled)
		}

		c := new(big.Int).Add(new(big.Int).Mul(new(big.Int).Mul(big.NewInt(2), t), c0), bigOne)

		primeGenCounter++

		a, numIterations := generateIntegerFromSeed(uint32(c.BitLen()), outSeed)

		a = new(big.Int).Mod(a, new(big.Int).Sub(c, big.NewInt(3)))
		a.Add(a, big.NewInt(2))

		h := HashToBig(outSeed)
		h.Add(h, big.NewInt(int64(numIterations+1)))
		h.Mod(h, oneLsh256Minus1)
		outSeed, _ = BigToHash(h)

		z := new(big.Int).Exp(a, new(big.Int).Mul(big.NewInt(2), t), c)

		gcd := new(big.Int).GCD(nil, nil, c, new(big.Int).Sub(z, bigOne))
		if gcd.Cmp(bigOne) == 0 && z.Exp(z, c0, c).Cmp(bigOne) == 0 {
			return outSeed, primeGenCounter, c, nil
		}

		t.Add(t, bigOne)
	}
	return nil, 0, nil, errors.New("unable to generate random prime")
}

func calculateGroupModulusAndOrder(seed *chainhash.Hash, pLen uint32, qLen uint32) (*big.Int, *big.Int, *chainhash.Hash, *chainhash.Hash, error) {
	if qLen > 256 {

	}

	qseed, _, resultGroupOrder, err := generateRandomPrime(qLen, seed)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	p0len := uint32(math.Ceil(float64(pLen)/2.0 + 1))
	pseed, pgencounter, p0, err := generateRandomPrime(p0len, qseed)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	oldCounter := pgencounter

	x, iterations := generateIntegerFromSeed(pLen, pseed)

	pseedBig := HashToBig(pseed)
	pseedBig.Add(pseedBig, big.NewInt(int64(iterations+1)))
	pseedBig.Mod(pseedBig, oneLsh256Minus1)
	pseed, err = BigToHash(pseedBig)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	powerOfTwo := new(big.Int).Lsh(bigOne, uint(pLen-1))
	x = new(big.Int).Add(powerOfTwo, new(big.Int).Mod(x, powerOfTwo))

	t := new(big.Int).Div(x, new(big.Int).Mul(new(big.Int).Mul(big.NewInt(2), resultGroupOrder), p0))

	for pgencounter <= ((4 * pLen) + oldCounter) {
		powerOfTwo = new(big.Int).Lsh(bigOne, uint(pLen))
		prod := new(big.Int).Mul(big.NewInt(2), t)
		prod.Mul(prod, resultGroupOrder)
		prod.Mul(prod, p0)
		prod.Add(prod, bigOne)
		if prod.Cmp(powerOfTwo) > 0 {
			t = new(big.Int).Lsh(bigOne, uint(pLen-1))
			divisor := big.NewInt(2)
			divisor.Mul(divisor, resultGroupOrder)
			divisor.Mul(divisor, p0)
			t.Div(t, divisor)
		}

		resultModulus := big.NewInt(2)
		resultModulus.Mul(resultModulus, t)
		resultModulus.Mul(resultModulus, resultGroupOrder)
		resultModulus.Mul(resultModulus, p0)
		resultModulus.Add(resultModulus, bigOne)

		a, iterations := generateIntegerFromSeed(pLen, pseed)

		pseedBig := HashToBig(pseed)
		pseedBig.Add(pseedBig, big.NewInt(int64(iterations+1)))
		pseedBig.Mod(pseedBig, oneLsh256Minus1)
		pseed, err = BigToHash(pseedBig)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		a = new(big.Int).Mod(a, new(big.Int).Sub(resultModulus, big.NewInt(3)))
		a.Add(a, big.NewInt(2))

		z1 := big.NewInt(2)
		z1.Mul(z1, t)
		z1.Mul(z1, resultGroupOrder)

		z := new(big.Int).Exp(a, z1, resultModulus)

		zMinusOne := new(big.Int).Sub(z, bigOne)

		if new(big.Int).GCD(nil, nil, resultModulus, zMinusOne).Cmp(bigOne) == 0 && new(big.Int).Exp(z, p0, resultModulus).Cmp(bigOne) == 0 {
			return resultModulus, resultGroupOrder, pseed, qseed, nil
		}

		t.Add(t, bigOne)
		pgencounter++
	}

	return nil, nil, nil, nil, errors.New("unable to generate a prime modulus for the group")
}

func calculateGeneratorSeed(seed, pSeed, qSeed *chainhash.Hash, label string, index uint32, count uint32) chainhash.Hash {
	var indexBytes [4]byte
	var countBytes [4]byte
	binary.LittleEndian.PutUint32(indexBytes[:], index)
	binary.LittleEndian.PutUint32(countBytes[:], count)

	separator := []byte("||")

	outBytes := appendAll(seed[:], separator, pSeed[:], separator, qSeed[:], separator, []byte(label), separator, indexBytes[:], separator, countBytes[:])

	return chainhash.HashH(outBytes)
}

func calculateGroupGenerator(seed, pSeed, qSeed *chainhash.Hash, modulus, groupOrder *big.Int, index uint32) (*big.Int, error) {
	if index > 255 {
		return nil, errors.New("invalid index for group generation")
	}

	e := new(big.Int).Div(new(big.Int).Sub(modulus, bigOne), groupOrder)

	for count := uint32(1); count < maxPrimeGenerationAttempts; count++ {
		hash := calculateGeneratorSeed(seed, pSeed, qSeed, "ggen", index, count)
		hashBig := HashToBig(&hash)

		hashBig.Exp(hashBig, e, modulus)

		if hashBig.Cmp(bigOne) > 0 {
			return hashBig, nil
		}
	}

	return nil, errors.New("unable to find generator, too many attempts")
}

func deriveIntegerGroupParams(seed *chainhash.Hash, pLen uint32, qLen uint32) (*IntegerGroupParams, error) {
	// Calculate "p" and "q" and "domain_parameter_seed" from the
	// "seed" buf[:]fer above, using the procedure described in NIST
	// FIPS 186-3, Appendix A.1.2.
	modulus, groupOrder, pSeed, qSeed, err := calculateGroupModulusAndOrder(seed, pLen, qLen)
	if err != nil {
		return nil, err
	}

	generator1, err := calculateGroupGenerator(seed, pSeed, qSeed, modulus, groupOrder, 1)
	if err != nil {
		return nil, err
	}
	generator2, err := calculateGroupGenerator(seed, pSeed, qSeed, modulus, groupOrder, 2)
	if err != nil {
		return nil, err
	}

	result := &IntegerGroupParams{
		G:          generator1,
		H:          generator2,
		Modulus:    modulus,
		GroupOrder: groupOrder,
	}

	if result.Modulus.BitLen() < int(pLen) {
		return nil, errors.New("generated smaller than expected modulus")
	}

	if result.GroupOrder.BitLen() < int(qLen) {
		return nil, errors.New("generated smaller than expected group order")
	}

	if !result.Modulus.ProbablyPrime(40) {
		return nil, errors.New("generated non-prime modulus")
	}

	if !result.GroupOrder.ProbablyPrime(40) {
		return nil, errors.New("generated non-prime group order")
	}

	if new(big.Int).Exp(result.G, result.GroupOrder, result.Modulus).Cmp(bigOne) != 0 {
		return nil, errors.New("G^order mod modulus != 1")
	}

	if new(big.Int).Exp(result.H, result.GroupOrder, result.Modulus).Cmp(bigOne) != 0 {
		return nil, errors.New("H^order mod modulus != 1")
	}

	if new(big.Int).Exp(result.G, big.NewInt(100), result.Modulus).Cmp(bigOne) == 0 {
		return nil, errors.New("G^100 mod modulus != 1")
	}

	if new(big.Int).Exp(result.H, big.NewInt(100), result.Modulus).Cmp(bigOne) == 0 {
		return nil, errors.New("H^100 mod modulus != 1")
	}

	if result.G.Cmp(result.H) == 0 {
		return nil, errors.New("H == G")
	}

	if result.G.Cmp(bigOne) == 0 {
		return nil, errors.New("G == 1")
	}

	return result, nil
}

// deriveIntegerGroupFromOrder calculates the description of a group G of a specific
// prime order embedded within a field.
func deriveIntegerGroupFromOrder(groupOrder *big.Int) (*IntegerGroupParams, error) {
	result := &IntegerGroupParams{
		GroupOrder: groupOrder,
	}

	for i := uint32(1); i < maxPrimeGenerationAttempts; i++ {
		result.Modulus = new(big.Int).Mul(result.GroupOrder, big.NewInt(int64(i*2)))
		result.Modulus.Add(result.Modulus, bigOne)

		if result.Modulus.ProbablyPrime(128) {
			seed := calculateSeed(groupOrder, "", 128, "")
			pSeed := calculateHash(HashToBig(&seed))
			qSeed := calculateHash(pSeed)
			pSeedHash, _ := BigToHash(pSeed)
			qSeedHash, _ := BigToHash(qSeed)
			generator1, err := calculateGroupGenerator(&seed, pSeedHash, qSeedHash, result.Modulus, result.GroupOrder, 1)
			if err != nil {
				return nil, err
			}
			generator2, err := calculateGroupGenerator(&seed, pSeedHash, qSeedHash, result.Modulus, result.GroupOrder, 2)
			if err != nil {
				return nil, err
			}

			result.G = generator1
			result.H = generator2

			if !result.Modulus.ProbablyPrime(40) {
				return nil, errors.New("generated non-prime modulus")
			}

			if !result.GroupOrder.ProbablyPrime(40) {
				return nil, errors.New("generated non-prime group order")
			}

			if new(big.Int).Exp(result.G, result.GroupOrder, result.Modulus).Cmp(bigOne) != 0 {
				return nil, errors.New("G^order mod modulus != 1")
			}

			if new(big.Int).Exp(result.H, result.GroupOrder, result.Modulus).Cmp(bigOne) != 0 {
				return nil, errors.New("H^order mod modulus != 1")
			}

			if new(big.Int).Exp(result.G, big.NewInt(100), result.Modulus).Cmp(bigOne) == 0 {
				return nil, errors.New("G^100 mod modulus != 1")
			}

			if new(big.Int).Exp(result.H, big.NewInt(100), result.Modulus).Cmp(bigOne) == 0 {
				return nil, errors.New("H^100 mod modulus != 1")
			}

			if result.G.Cmp(result.H) == 0 {
				return nil, errors.New("H == G")
			}

			if result.G.Cmp(bigOne) == 0 {
				return nil, errors.New("G == 1")
			}

			return result, nil
		}
	}

	return nil, errors.New("too many attempts to generate schnorr group")
}

// calculateParams fills a params object deterministically given an RSA
// modulus N.
func calculateParams(N *big.Int, aux string, securityLevel uint32, params *Params) error {
	params.Initialized = false
	params.AccumulatorParams.Initialized = false

	l := N.BitLen()
	if l < 1023 {
		return errors.New("Modulus must be at least 1023 bits")
	}

	if securityLevel < 80 {
		return errors.New("Security level must be at least 80 bits")
	}

	params.AccumulatorParams.AccumulatorModulus = N

	pLen, qLen, err := calculateGroupParamLengths(uint32(l-2), securityLevel)
	if err != nil {
		return err
	}

	seed := calculateSeed(N, aux, securityLevel, "COIN_COMMITMENT_GROUP")

	params.CoinCommitmentGroup, err = deriveIntegerGroupParams(&seed, pLen, qLen)
	if err != nil {
		return err
	}

	params.SerialNumberSoKCommitmentGroup, err = deriveIntegerGroupFromOrder(params.CoinCommitmentGroup.Modulus)
	if err != nil {
		return err
	}

	newSeed := calculateSeed(N, aux, securityLevel, "ACCUMULATOR_INTERNAL_COMMITMENT_GROUP")

	params.AccumulatorParams.AccumulatorPoKCommitmentGroup, err = deriveIntegerGroupParams(&newSeed, qLen+300, qLen+1)
	if err != nil {
		return err
	}

	qrnCommitmentGroupGSeed := calculateSeed(N, aux, securityLevel, "ACCUMULATOR_QRN_COMMITMENT_GROUPG")
	params.AccumulatorParams.AccumulatorQRNCommitmentGroup.G, _ = generateIntegerFromSeed(uint32(l-1), &qrnCommitmentGroupGSeed)
	params.AccumulatorParams.AccumulatorQRNCommitmentGroup.G.Exp(params.AccumulatorParams.AccumulatorQRNCommitmentGroup.G, big.NewInt(2), N)

	qrnCommitmentGroupHSeed := calculateSeed(N, aux, securityLevel, "ACCUMULATOR_QRN_COMMITMENT_GROUPH")
	params.AccumulatorParams.AccumulatorQRNCommitmentGroup.H, _ = generateIntegerFromSeed(uint32(l-1), &qrnCommitmentGroupHSeed)
	params.AccumulatorParams.AccumulatorQRNCommitmentGroup.H.Exp(params.AccumulatorParams.AccumulatorQRNCommitmentGroup.H, big.NewInt(2), N)

	constant := big.NewInt(31)
	params.AccumulatorParams.AccumulatorBase = new(big.Int).Exp(constant, big.NewInt(2), params.AccumulatorParams.AccumulatorModulus)

	params.AccumulatorParams.MaxCoinValue = params.CoinCommitmentGroup.Modulus
	params.AccumulatorParams.MinCoinValue = new(big.Int).Exp(big.NewInt(2), big.NewInt(int64((params.CoinCommitmentGroup.Modulus.BitLen()/2)+3)), nil)

	params.AccumulatorParams.Initialized = true
	params.Initialized = true

	return nil
}

const accumulatorProofKPrime = 160
const accumulatorProofKDPrime = 128
const zerocoinProtocolVersion = "1"

// NewZerocoinParams returns a new set of Zerocoin params given a certain modulus.
func NewZerocoinParams(N *big.Int, securityLevel uint32) (*Params, error) {
	params := &Params{
		AccumulatorParams: &AccumulatorAndProofParams{
			AccumulatorQRNCommitmentGroup: &IntegerGroupParams{},
			AccumulatorPoKCommitmentGroup: &IntegerGroupParams{},
		},
	}
	params.ZKPHashLength = securityLevel
	params.ZKPIterations = securityLevel

	params.AccumulatorParams.KPrime = accumulatorProofKPrime
	params.AccumulatorParams.KDPrime = accumulatorProofKDPrime

	err := calculateParams(N, zerocoinProtocolVersion, securityLevel, params)
	if err != nil {
		return nil, err
	}

	params.AccumulatorParams.Initialized = true
	params.Initialized = true

	return params, nil
}

// NewAccumulatorAndProofParams returns a new uninitialized version of
// accumulator and proof parameters.
func NewAccumulatorAndProofParams() *AccumulatorAndProofParams {
	return &AccumulatorAndProofParams{Initialized: false}
}

// NewIntegerGroupParams returns a new uninitialized version of
// integer group params.
func NewIntegerGroupParams() *IntegerGroupParams {
	return &IntegerGroupParams{Initialized: false}
}
