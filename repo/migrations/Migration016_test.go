package migrations_test

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/phoreproject/openbazaar-go/repo/migrations"
	"github.com/phoreproject/openbazaar-go/schema"
)

const preMigration016Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets":{
		"PHR": {
			"API": [
					"https://phr.blockbook.api.phore.io/api"
			],
			"APITestnet": [
					"https://tphr.blockbook.api.phore.io/api"
			]
		},
		"BTC": {
			"API": [
					"https://btc.api.openbazaar.org/api"
			],
			"APITestnet": [
					"https://tbtc.api.openbazaar.org/api"
			]
		}
	}
}`

const postMigration016Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets": {
		"PHR": {
			"Type": "API",
			"API": [
					"https://phr.blockbook.api.phore.io/api"
			],
			"APITestnet": [
					"https://tphr.blockbook.api.phore.io/api"
			],
			"MaxFee": 200,
			"FeeAPI": "",
			"HighFeeDefault": 50,
			"MediumFeeDefault": 10,
			"LowFeeDefault": 1,
			"TrustedPeer": "",
			"WalletOptions": null
		},
		"BTC": {
			"Type": "API",
			"API": [
					"https://btc.blockbook.api.openbazaar.org/api"
			],
			"APITestnet": [
					"https://tbtc.blockbook.api.openbazaar.org/api"
			],
			"MaxFee": 200,
			"FeeAPI": "https://btc.fees.openbazaar.org",
			"HighFeeDefault": 50,
			"MediumFeeDefault": 10,
			"LowFeeDefault": 1,
			"TrustedPeer": "",
			"WalletOptions": null
		},
		"BCH": null,
		"LTC": null,
		"ZEC": null,
		"ETH": null
	}
}`

func migration016AssertAPI(t *testing.T, actual interface{}, expected string) {
	actualSlice := actual.([]interface{})
	if len(actualSlice) != 1 || actualSlice[0] != expected {
		t.Fatalf("incorrect api endpoint.\n\twanted: %s\n\tgot: %s\n", expected, actual)
	}
}

func TestMigration016(t *testing.T) {
	var testRepo, err = schema.NewCustomSchemaManager(schema.SchemaContext{
		DataPath:        schema.GenerateTempPath(),
		TestModeEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = testRepo.BuildSchemaDirectories(); err != nil {
		t.Fatal(err)
	}
	defer testRepo.DestroySchemaDirectories()

	var (
		configPath  = testRepo.DataPathJoin("config")
		repoverPath = testRepo.DataPathJoin("repover")
	)
	if err = ioutil.WriteFile(configPath, []byte(preMigration016Config), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(repoverPath, []byte("15"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var m migrations.Migration016
	err = m.Up(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	config := map[string]interface{}{}
	if err = json.Unmarshal(configBytes, &config); err != nil {
		t.Fatal(err)
	}

	w := config["Wallets"].(map[string]interface{})
	btc := w["BTC"].(map[string]interface{})
	phr := w["PHR"].(map[string]interface{})

	migration016AssertAPI(t, btc["API"], "https://btc.blockbook.api.openbazaar.org/api")
	migration016AssertAPI(t, btc["APITestnet"], "https://tbtc.blockbook.api.openbazaar.org/api")
	migration016AssertAPI(t, phr["API"], "https://phr.blockbook.api.phore.io/api")
	migration016AssertAPI(t, phr["APITestnet"], "https://tphr.blockbook.api.phore.io/api")

	var re = regexp.MustCompile(`\s`)
	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(postMigration016Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "17")
}
