package migrations

import (
	"path"
	"io/ioutil"
	"os"
)

type Migration005 struct{}

var SwarmKeyData []byte = []byte("/key/swarm/psk/1.0.0/\n/base16/\n59468cfd4d4dc2a61395080513e853434d0313495f34be65c18d643d09eafe6f")

func (Migration005) Up(repoPath string, dbPassword string, testnet bool) error {

	f1, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(path.Join(repoPath, "swarm.key"), SwarmKeyData, 0644); err != nil {
		return err
	}

	_, err = f1.Write([]byte("5"))
	if err != nil {
		return err
	}
	f1.Close()
	return nil
}

func (Migration005) Down(repoPath string, dbPassword string, testnet bool) error {
	f1, err := os.Create(path.Join(repoPath, "repover"))
	if err != nil {
		return err
	}

	if err = os.Remove(path.Join(repoPath, "swarm.key")); err != nil {
		return err
	}

	_, err = f1.Write([]byte("4"))
	if err != nil {
		return err
	}
	f1.Close()
	return nil
}
