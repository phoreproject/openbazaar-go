package exchange

import (
	"bytes"
	"encoding/json"
	"io"
	gonet "net"
	"net/http"
	"testing"
	"time"
)

func setupBitcoinPriceFetcher() (b BitcoinPriceFetcher) {
	b = BitcoinPriceFetcher{
		cache: make(map[string]float64),
	}
	client := &http.Client{Transport: &http.Transport{Dial: gonet.Dial}, Timeout: time.Minute}
	b.providers = []*ExchangeRateProvider{
		{"https://api.coinmarketcap.com/v1/ticker/phore/", b.cache, client, CMCDecoder{}},
	}
	return b
}

func TestFetchCurrentRates(t *testing.T) {
	b := setupBitcoinPriceFetcher()
	err := b.fetchCurrentRates()
	if err != nil {
		t.Error("Failed to fetch bitcoin exchange rates")
	}
}

func TestGetLatestRate(t *testing.T) {
	b := setupBitcoinPriceFetcher()
	price, err := b.GetLatestRate("PHR")
	if err != nil || price == 650 {
		t.Error("Incorrect return at GetLatestRate (price, err)", price, err)
	}
	b.cache["PHR"] = 650.00
	price, ok := b.cache["PHR"]
	if !ok || price != 650 {
		t.Error("Failed to fetch exchange rates from cache")
	}
	price, err = b.GetLatestRate("PHR")
	if err != nil || price == 650.00 {
		t.Error("Incorrect return at GetLatestRate (price, err)", price, err)
	}
}

func TestGetAllRates(t *testing.T) {
	b := setupBitcoinPriceFetcher()
	b.cache["USD"] = 650.00
	b.cache["EUR"] = 600.00
	priceMap, err := b.GetAllRates(true)
	if err != nil {
		t.Error(err)
	}
	usd, ok := priceMap["USD"]
	if !ok || usd != 650.00 {
		t.Error("Failed to fetch exchange rates from cache")
	}
	eur, ok := priceMap["EUR"]
	if !ok || eur != 600.00 {
		t.Error("Failed to fetch exchange rates from cache")
	}
}

func TestGetExchangeRate(t *testing.T) {
	b := setupBitcoinPriceFetcher()
	b.cache["usd"] = 650.00
	r, err := b.GetExchangeRate("usd")
	if err != nil {
		t.Error("Failed to fetch exchange rate")
	}
	if r != 650.00 {
		t.Error("Returned exchange rate incorrect")
	}
	r, err = b.GetExchangeRate("EUR")
	if r != 0 || err == nil {
		t.Error("Return erroneous exchange rate")
	}
}

type req struct {
	io.Reader
}

func (r *req) Close() error {
	return nil
}

func TestDecodeCMCDecoder(t *testing.T) {
	cache := make(map[string]float64)
	cmcDecoder := CMCDecoder{}
	var dataMap interface{}

	response := `[
      {
          "id": "phore",
          "name": "Phore",
          "symbol": "PHR",
          "rank": "377",
          "price_usd": "0.542017",
          "price_btc": "0.00004258",
          "24h_volume_usd": "52747.3",
          "market_cap_usd": "5275455.0",
          "available_supply": "9733007.0",
          "total_supply": "11349574.0",
          "max_supply": null,
          "percent_change_1h": "4.49",
          "percent_change_24h": "21.59",
          "percent_change_7d": "11.32",
          "last_updated": "1512583195"
      }
  ]`
	// Test valid response
	r := &req{bytes.NewReader([]byte(response))}
	decoder := json.NewDecoder(r)
	err := decoder.Decode(&dataMap)
	if err != nil {
		t.Error(err)
	}
	err = cmcDecoder.decode(dataMap, cache)
	if err != nil {
		t.Error(err)
	}

	// Make sure it saved to cache
	if len(cache) == 0 {
		t.Error("Failed to response to cache")
	}

	resp := `[]`

	// Test missing JSON element
	r = &req{bytes.NewReader([]byte(resp))}
	decoder = json.NewDecoder(r)
	err = decoder.Decode(&dataMap)
	if err != nil {
		t.Error(err)
	}
	err = cmcDecoder.decode(dataMap, cache)
	if err == nil {
		t.Error(err)
	}
	resp = `[]`

	// Test invalid JSON
	r = &req{bytes.NewReader([]byte(resp))}
	decoder = json.NewDecoder(r)
	err = decoder.Decode(&dataMap)
	if err != nil {
		t.Error(err)
	}
	err = cmcDecoder.decode(dataMap, cache)
	if err == nil {
		t.Error(err)
	}

	// Test decode error
	r = &req{bytes.NewReader([]byte(""))}
	decoder = json.NewDecoder(r)
	decoder.Decode(&dataMap)
	err = cmcDecoder.decode(dataMap, cache)
	if err == nil {
		t.Error(err)
	}
}
