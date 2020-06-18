package migrations

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"
)

type Migration000 struct{}

func (Migration000) Up(repoPath string, dbPassword string, testnet bool) error {
	configFile, err := ioutil.ReadFile(path.Join(repoPath, "config"))
	if err != nil {
		return err
	}
	var cfgIface interface{}
	err = json.Unmarshal(configFile, &cfgIface)
	if err != nil {
		return err
	}
	cfg, ok := cfgIface.(map[string]interface{})
	if !ok {
		return errors.New("invalid config file")
	}

	walletIface, ok := cfg["Wallet"]
	if !ok {
		return errors.New("missing wallet config")
	}
	wallet, ok := walletIface.(map[string]interface{})
	if !ok {
		return errors.New("error parsing wallet config")
	}
	feeAPI, ok := wallet["FeeAPI"]
	if !ok {
		return errors.New("missing FeeAPI config")
	}
	feeAPIstr, ok := feeAPI.(string)
	if !ok {
		return errors.New("error parsing FeeAPI")
	}
	if feeAPIstr == "https://bitcoinfees.21.co/api/v1/fees/recommended" {
		wallet["FeeAPI"] = "https://btc.fees.openbazaar.org"
		cfg["Wallet"] = wallet
		out, err := json.MarshalIndent(cfg, "", "   ")
		if err != nil {
			return err
		}
		f, err := os.Create(path.Join(repoPath, "config"))
		if err != nil {
			return err
		}
		_, err = f.Write(out)
		if err != nil {
			return err
		}
		f.Close()
	}
	f, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("1"))
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

func (Migration000) Down(repoPath string, dbPassword string, testnet bool) error {
	configFile, err := ioutil.ReadFile(path.Join(repoPath, "config"))
	if err != nil {
		return err
	}
	var cfgIface interface{}
	err = json.Unmarshal(configFile, &cfgIface)
	if err != nil {
		return err
	}
	cfg, ok := cfgIface.(map[string]interface{})
	if !ok {
		return errors.New("invalid config file")
	}

	walletIface, ok := cfg["Wallet"]
	if !ok {
		return errors.New("missing wallet config")
	}
	wallet, ok := walletIface.(map[string]interface{})
	if !ok {
		return errors.New("error parsing wallet config")
	}
	feeAPI, ok := wallet["FeeAPI"]
	if !ok {
		return errors.New("missing FeeAPI config")
	}
	feeAPIstr, ok := feeAPI.(string)
	if !ok {
		return errors.New("error parsing FeeAPI")
	}
	if feeAPIstr == "https://btc.fees.openbazaar.org" {
		wallet["FeeAPI"] = "https://bitcoinfees.21.co/api/v1/fees/recommended"
		cfg["Wallet"] = wallet
		out, _ := json.MarshalIndent(cfg, "", "   ")
		f, err := os.Create(path.Join(repoPath, "config"))
		if err != nil {
			return err
		}
		_, err = f.Write(out)
		if err != nil {
			return err
		}
		f.Close()
	}
	f, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}
	_, err = f.Write([]byte("0"))
	if err != nil {
		return err
	}
	f.Close()
	return nil
}
