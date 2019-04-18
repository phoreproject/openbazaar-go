package core_test

import (
	"testing"

	"github.com/phoreproject/openbazaar-go/core"
	"github.com/phoreproject/openbazaar-go/test/factory"
)

func TestFactoryCryptoListingCoinDivisibilityMatchesConst(t *testing.T) {
	if factory.NewCryptoListing("blu").Metadata.CoinDivisibility != core.DefaultCoinDivisibility {
		t.Fatal("DefaultCoinDivisibility constant has changed. Please update factory value.")
	}
}
