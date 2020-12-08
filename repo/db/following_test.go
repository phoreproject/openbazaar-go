package db_test

import (
	"strconv"
	"sync"
	"testing"

	"github.com/phoreproject/pm-go/repo"
	"github.com/phoreproject/pm-go/repo/db"
	"github.com/phoreproject/pm-go/schema"
)

func buildNewFollowingStore() (repo.FollowingStore, func(), error) {
	appSchema := schema.MustNewCustomSchemaManager(schema.SchemaContext{
		DataPath:        schema.GenerateTempPath(),
		TestModeEnabled: true,
	})
	if err := appSchema.BuildSchemaDirectories(); err != nil {
		return nil, nil, err
	}
	if err := appSchema.InitializeDatabase(); err != nil {
		return nil, nil, err
	}
	database, err := appSchema.OpenDatabase()
	if err != nil {
		return nil, nil, err
	}
	return db.NewFollowingStore(database, new(sync.Mutex)), appSchema.DestroySchemaDirectories, nil
}

func TestPutFollowing(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	err = fldb.Put("abc")
	if err != nil {
		t.Error(err)
	}
	stmt, err := fldb.PrepareQuery("select peerID from following where peerID=?")
	if err != nil {
		t.Error(err)
	}
	defer stmt.Close()
	var following string
	err = stmt.QueryRow("abc").Scan(&following)
	if err != nil {
		t.Error(err)
	}
	if following != "abc" {
		t.Errorf(`Expected "abc" got %s`, following)
	}
}

func TestPutDuplicateFollowing(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	err = fldb.Put("abc")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Put("abc")
	if err == nil {
		t.Error("Expected unquire constriant error to be thrown")
	}
}

func TestCountFollowing(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	err = fldb.Put("abc")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Put("123")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Put("xyz")
	if err != nil {
		t.Error(err)
	}
	x := fldb.Count()
	if x != 3 {
		t.Errorf("Expected 3 got %d", x)
	}
	err = fldb.Delete("abc")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Delete("123")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Delete("xyz")
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteFollowing(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	err = fldb.Put("abc")
	if err != nil {
		t.Error(err)
	}
	err = fldb.Delete("abc")
	if err != nil {
		t.Error(err)
	}
	stmt, _ := fldb.PrepareQuery("select peerID from followers where peerID=?")
	defer stmt.Close()
	var follower string
	err = stmt.QueryRow("abc").Scan(&follower)
	if err != nil {
		t.Log(err)
	}
	if follower != "" {
		t.Error("Failed to delete follower")
	}
}

func TestGetFollowing(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	for i := 0; i < 100; i++ {
		err = fldb.Put(strconv.Itoa(i))
		if err != nil {
			t.Error(err)
		}
	}
	followers, err := fldb.Get("", 100)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 100; i++ {
		f, _ := strconv.Atoi(followers[i])
		if f != 99-i {
			t.Errorf("Returned %d expected %d", f, 99-i)
		}
	}

	followers, err = fldb.Get(strconv.Itoa(30), 100)
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 30; i++ {
		f, _ := strconv.Atoi(followers[i])
		if f != 29-i {
			t.Errorf("Returned %d expected %d", f, 29-i)
		}
	}
	if len(followers) != 30 {
		t.Error("Incorrect number of followers returned")
	}

	followers, err = fldb.Get(strconv.Itoa(30), 5)
	if err != nil {
		t.Error(err)
	}
	if len(followers) != 5 {
		t.Error("Incorrect number of followers returned")
	}
	for i := 0; i < 5; i++ {
		f, _ := strconv.Atoi(followers[i])
		if f != 29-i {
			t.Errorf("Returned %d expected %d", f, 29-i)
		}
	}
}

func TestIFollow(t *testing.T) {
	fldb, teardown, err := buildNewFollowingStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	err = fldb.Put("abc")
	if err != nil {
		t.Error(err)
	}
	if !fldb.IsFollowing("abc") {
		t.Error("I follow failed to return correctly")
	}
}
