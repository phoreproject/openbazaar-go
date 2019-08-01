package migrations_test

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/phoreproject/openbazaar-go/repo/migrations"
	"github.com/phoreproject/openbazaar-go/schema"
)

const preMigration017Config = `{
	"DataSharing": {
		"AcceptStoreRequests": false,
		"PushTo": [
			"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
			"QmRh7fSZyFHesEL9aTmdxbrvMFxzyFxoaQGjYBotot6WLw",
			"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf"
		]
	},
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123"
}`

const postMigration017Config = `{
	"DataSharing": {
		"AcceptStoreRequests": false,
		"PushTo": [
			"QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh",
			"QmRh7fSZyFHesEL9aTmdxbrvMFxzyFxoaQGjYBotot6WLw",
			"QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf"
		]
	},
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123"
}`

func TestMigration017(t *testing.T) {
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
	if err = ioutil.WriteFile(configPath, []byte(preMigration017Config), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(repoverPath, []byte("15"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var m migrations.Migration017
	err = m.Up(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var re = regexp.MustCompile(`\s`)
	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(postMigration017Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "18")

	err = m.Down(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err = ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(preMigration017Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "17")
}
