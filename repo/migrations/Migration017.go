package migrations

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

var (
	Migration017PushToBefore = []string{
		"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"QmRh7fSZyFHesEL9aTmdxbrvMFxzyFxoaQGjYBotot6WLw",
		"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
	}

	Migration017PushToAfter = []string{
		"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"Qma2LRYB4xLaoxsMCL2kb93WKCW4EotUMhgvQUSqE6tCka",
		"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
		"QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv",
	}
)

type migration017DataSharing struct {
	AcceptStoreRequests bool
	PushTo              []string
}

type Migration017 struct{}

func (Migration017) Up(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	configMap["DataSharing"] = migration017DataSharing{PushTo: Migration017PushToAfter}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 18); err != nil {
		return fmt.Errorf("bumping repover to 18: %s", err.Error())
	}
	return nil
}

func (Migration017) Down(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	configMap["DataSharing"] = migration017DataSharing{PushTo: Migration017PushToBefore}

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 17); err != nil {
		return fmt.Errorf("dropping repover to 16: %s", err.Error())
	}
	return nil
}
