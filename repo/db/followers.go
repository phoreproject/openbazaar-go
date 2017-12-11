package db

import (
	"database/sql"
	"strconv"
	"sync"

	"github.com/phoreproject/openbazaar-go/repo"
)

type FollowerDB struct {
	db   *sql.DB
	lock *sync.Mutex
}

func (f *FollowerDB) Put(follower string, proof []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	tx, _ := f.db.Begin()
	stmt, _ := tx.Prepare("insert into followers(peerID, proof) values(?,?)")

	defer stmt.Close()
	_, err := stmt.Exec(follower, proof)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (f *FollowerDB) Get(offsetID string, limit int) ([]repo.Follower, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	var stm string
	if offsetID != "" {
		stm = "select peerID, proof from followers order by rowid desc limit " + strconv.Itoa(limit) + " offset ((select coalesce(max(rowid)+1, 0) from followers)-(select rowid from followers where peerID='" + offsetID + "'))"
	} else {
		stm = "select peerID, proof from followers order by rowid desc limit " + strconv.Itoa(limit) + " offset 0"
	}
	var ret []repo.Follower
	rows, err := f.db.Query(stm)
	if err != nil {
		return ret, err
	}
	defer rows.Close()

	for rows.Next() {
		var peerID string
		var proof []byte
		rows.Scan(&peerID, &proof)
		ret = append(ret, repo.Follower{peerID, proof})
	}
	return ret, nil
}

func (f *FollowerDB) Delete(follower string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.db.Exec("delete from followers where peerID=?", follower)
	return nil
}

func (f *FollowerDB) Count() int {
	f.lock.Lock()
	defer f.lock.Unlock()
	row := f.db.QueryRow("select Count(*) from followers")
	var count int
	row.Scan(&count)
	return count
}

func (f *FollowerDB) FollowsMe(peerID string) bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	stmt, err := f.db.Prepare("select peerID from followers where peerID=?")
	defer stmt.Close()
	var follower string
	err = stmt.QueryRow(peerID).Scan(&follower)
	if err != nil {
		return false
	}
	return true
}
