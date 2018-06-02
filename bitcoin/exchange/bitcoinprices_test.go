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
		{"https://api.coinmarketcap.com/v2/ticker/2158/?convert=", b.cache, client, CMCDecoder{}},
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
	price, err := b.GetLatestRate("USD")
	if err != nil || price == 650 {
		t.Error("Incorrect return at GetLatestRate (price, err)", price, err)
	}
	b.cache["USD"] = 650.00
	price, ok := b.cache["USD"]
	if !ok || price != 650 {
		t.Error("Failed to fetch exchange rates from cache")
	}
	price, err = b.GetLatestRate("USD")
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
	b.cache["USD"] = 650.00
	r, err := b.GetExchangeRate("USD")
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

	// Test that currency symbols are normalized correctly
	r, err = b.GetExchangeRate("usd")
	if err != nil {
		t.Error("Failed to fetch exchange rate")
	}
	if r != 650.00 {
		t.Error("Returned exchange rate incorrect")
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

	correctResponse := `{
    	"data": {
        	"id": 2158, 
        	"name": "Phore", 
        	"symbol": "PHR", 
        	"website_slug": "phore", 
        	"rank": 313, 
        	"circulating_supply": 13318225.0, 
        	"total_supply": 13318225.0, 
        	"max_supply": null, 
        	"quotes": {
            	"USD": {
                	"price": 2.00949, 
                	"volume_24h": 232477.0, 
                	"market_cap": 26762840.0, 
                	"percent_change_1h": -0.58, 
                	"percent_change_24h": 4.97, 
                	"percent_change_7d": 26.54
            	}
        	}, 
        	"last_updated": 1527944368
    	}, 
    	"metadata": {
        	"timestamp": 1527944160, 
        	"error": null
    	}
	}`
	// Test valid correctResponse
	r := &req{bytes.NewReader([]byte(correctResponse))}
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

	emptyResponse := `{}`
	// Test missing JSON element
	r = &req{bytes.NewReader([]byte(emptyResponse))}
	decoder = json.NewDecoder(r)
	err = decoder.Decode(&dataMap)
	if err != nil {
		t.Error(err)
	}
	err = cmcDecoder.decode(dataMap, cache)
	if err == nil {
		t.Error(err)
	}

	errorResponse := `{
    	"data": null, 
    	"metadata": {
        	"timestamp": 1527944160, 
        	"error": "some error"
    	}
	}`
	// Test CMC error not empty
	r = &req{bytes.NewReader([]byte(errorResponse))}
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
