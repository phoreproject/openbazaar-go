package migrations

import (
	"errors"
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
)

type Migration005 struct{}

var SwarmKeyData = []byte("/key/swarm/psk/1.0.0/\n/base16/\n59468cfd4d4dc2a61395080513e853434d0313495f34be65c18d643d09eafe6f")

func (Migration005) Up(repoPath string, dbPassword string, testnet bool) error {
	configFile, err := ioutil.ReadFile(path.Join(repoPath, "config"))
	if err != nil {
		return err
	}
	var cfgIface interface{}
	json.Unmarshal(configFile, &cfgIface)
	cfg, ok := cfgIface.(map[string]interface{})
	if !ok {
		return errors.New("Invalid config file")
	}

	ipns, ok := cfg["Ipns"]
	if !ok {
		return errors.New("Ipns config not found")
	}
	ipnsCfg, ok := ipns.(map[string]interface{})
	if !ok {
		return errors.New("Ipns config not found")
	}
	ipnsCfg["QuerySize"] = 1
	ipnsCfg["BackUpAPI"] = "https://gateway.ob1.io/ob/ipns/"

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

	if err = ioutil.WriteFile(path.Join(repoPath, "swarm.key"), SwarmKeyData, 0644); err != nil {
		return err
	}

	f1, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}
	_, err = f1.Write([]byte("6"))
	if err != nil {
		return err
	}
	f1.Close()
	return nil
}

func (Migration005) Down(repoPath string, dbPassword string, testnet bool) error {
	configFile, err := ioutil.ReadFile(path.Join(repoPath, "config"))
	if err != nil {
		return err
	}
	var cfgIface interface{}
	json.Unmarshal(configFile, &cfgIface)
	cfg, ok := cfgIface.(map[string]interface{})
	if !ok {
		return errors.New("Invalid config file")
	}

	ipns, ok := cfg["Ipns"]
	if !ok {
		return errors.New("Ipns config not found")
	}
	ipnsCfg, ok := ipns.(map[string]interface{})
	if !ok {
		return errors.New("Ipns config not found")
	}
	ipnsCfg["QuerySize"] = 5
	ipnsCfg["BackUpAPI"] = ""

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

	if err = os.Remove(path.Join(repoPath, "swarm.key")); err != nil {
		return err
	}

	f1, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}
	_, err = f1.Write([]byte("5"))
	if err != nil {
		return err
	}
	f1.Close()
	return nil
}