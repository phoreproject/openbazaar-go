package migrations

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

var (
	Migration017PushToBeforePushTo = []string{
		"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"QmRh7fSZyFHesEL9aTmdxbrvMFxzyFxoaQGjYBotot6WLw",
		"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
	}

	Migration017PushToAfterPushTo = []string{
		"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"Qma2LRYB4xLaoxsMCL2kb93WKCW4EotUMhgvQUSqE6tCka",
		"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
		"QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv",
	}

	Migration017PushToBeforeBootstrapNodes = []string{
		"/ip4/54.227.172.110/tcp/5001/ipfs/QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"/ip4/45.63.71.103/tcp/5001/ipfs/QmRh7fSZyFHesEL9aTmdxbrvMFxzyFxoaQGjYBotot6WLw",
		"/ip4/54.175.193.226/tcp/5001/ipfs/QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
		"/ip4/34.239.133.237/tcp/5001/ipfs/QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv",
		"/ip4/159.203.115.78/tcp/5001/ipfs/QmPJuP4Myo8pGL1k56b85Q4rpaoSnmn5L3wLjYHTzbBrk1",
		"/ip4/104.131.19.44/tcp/5001/ipfs/QmRvbZttqh6CPFiMKWa1jPfRR9JxagYRv4wsvMAG4ADUTj",
	}

	Migration017PushToAfterBootstrapNodes = []string{
		"/ip4/54.227.172.110/tcp/5001/ipfs/QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
		"/ip4/144.202.25.235/tcp/5001/ipfs/Qma2LRYB4xLaoxsMCL2kb93WKCW4EotUMhgvQUSqE6tCka",
		"/ip4/54.175.193.226/tcp/5001/ipfs/QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf",
		"/ip4/34.239.133.237/tcp/5001/ipfs/QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv",
		"/ip4/159.203.115.78/tcp/5001/ipfs/QmPJuP4Myo8pGL1k56b85Q4rpaoSnmn5L3wLjYHTzbBrk1",
		"/ip4/104.131.19.44/tcp/5001/ipfs/QmRvbZttqh6CPFiMKWa1jPfRR9JxagYRv4wsvMAG4ADUTj",
	}
)

type migration015DataSharing struct {
	AcceptStoreRequests bool
	PushTo              []string
}

type Migration015 struct{}

func (Migration015) Up(repoPath, dbPassword string, testnet bool) error {
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

	configMap["DataSharing"] = migration015DataSharing{PushTo: Migration017PushToAfterPushTo}
	configMap["Bootstrap"] = Migration017PushToAfterBootstrapNodes

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	return nil
}

func (Migration015) Down(repoPath, dbPassword string, testnet bool) error {
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

	configMap["DataSharing"] = migration015DataSharing{PushTo: Migration017PushToBeforePushTo}
	configMap["Bootstrap"] = Migration017PushToBeforeBootstrapNodes

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	return nil
}
