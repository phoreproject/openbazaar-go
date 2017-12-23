package exchange

import (
	"encoding/json"
	"errors"
	"github.com/op/go-logging"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const SatoshiPerBTC = 100000000

var log = logging.MustGetLogger("exchangeRates")

type ExchangeRateProvider struct {
	fetchUrl string
	cache    map[string]float64
	client   *http.Client
	decoder  ExchangeRateDecoder
}

type ExchangeRateDecoder interface {
	decode(dat interface{}, cache map[string]float64) (err error)
}

// empty structs to tag the different ExchangeRateDecoder implementations
type CMCDecoder struct{}

type BitcoinPriceFetcher struct {
	sync.Mutex
	cache     map[string]float64
	providers []*ExchangeRateProvider
}

func NewBitcoinPriceFetcher(dialer proxy.Dialer) *BitcoinPriceFetcher {
	b := BitcoinPriceFetcher{
		cache: make(map[string]float64),
	}
	dial := net.Dial
	if dialer != nil {
		dial = dialer.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	client := &http.Client{Transport: tbTransport, Timeout: time.Minute}

	b.providers = []*ExchangeRateProvider{
		{"https://api.coinmarketcap.com/v1/ticker/phore", b.cache, client, CMCDecoder{}},
	}
	go b.run()
	return &b
}

func (b *BitcoinPriceFetcher) GetExchangeRate(currencyCode string) (float64, error) {
	b.Lock()
	defer b.Unlock()
	price, ok := b.cache[currencyCode]
	if !ok {
		return 0, errors.New("Currency not tracked")
	}
	return price, nil
}

func (b *BitcoinPriceFetcher) GetLatestRate(currencyCode string) (float64, error) {
	b.fetchCurrentRates()
	b.Lock()
	defer b.Unlock()
	price, ok := b.cache[currencyCode]
	if !ok {
		return 0, errors.New("Currency not tracked")
	}
	return price, nil
}

func (b *BitcoinPriceFetcher) GetAllRates() (map[string]float64, error) {
	b.Lock()
	defer b.Unlock()
	return b.cache, nil
}

func (b *BitcoinPriceFetcher) UnitsPerCoin() int {
	return SatoshiPerBTC
}

func (b *BitcoinPriceFetcher) fetchCurrentRates() error {
	b.Lock()
	defer b.Unlock()
	for _, provider := range b.providers {
		err := provider.fetch()
		if err == nil {
			return nil
		}
	}
	log.Error("Failed to fetch bitcoin exchange rates")
	return errors.New("All exchange rate API queries failed")
}

func (provider *ExchangeRateProvider) fetch() (err error) {
	if len(provider.fetchUrl) == 0 {
		err = errors.New("Provider has no fetchUrl")
		return err
	}
	resp, err := provider.client.Get(provider.fetchUrl)
	if err != nil {
		log.Error("Failed to fetch from "+provider.fetchUrl, err)
		return err
	}
	decoder := json.NewDecoder(resp.Body)
	var dataMap interface{}
	err = decoder.Decode(&dataMap)
	if err != nil {
		log.Error("Failed to decode JSON from "+provider.fetchUrl, err)
		return err
	}
	return provider.decoder.decode(dataMap, provider.cache)
}

func (b *BitcoinPriceFetcher) run() {
	b.fetchCurrentRates()
	ticker := time.NewTicker(time.Minute * 15)
	for range ticker.C {
		b.fetchCurrentRates()
	}
}

// Decoders
func (b CMCDecoder) decode(dat interface{}, cache map[string]float64) (err error) {
	data := dat.([]interface{})

	if len(data) < 1 {
		return errors.New("could not get currency")
	}

	priceInfo := data[0].(map[string]interface{})

	priceUSDObj, found := priceInfo["price_usd"]

	if !found {
		return errors.New("could not get USD price")
	}

	priceUSDString, ok := priceUSDObj.(string)

	if !ok {
		return errors.New("could not decode current USD price")
	}

	priceUSD, err := strconv.ParseFloat(priceUSDString, 64)

	if err != nil {
		return errors.New("usd price is not a number")
	}

	cache["PHR"] = priceUSD
	cache["USD"] = priceUSD

	return nil
}
