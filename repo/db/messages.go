package db

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/phoreproject/pm-go/pb"
	"github.com/phoreproject/pm-go/repo"
)

// MessagesDB represents the messages table
type MessagesDB struct {
	modelStore
}

// NewMessageStore return new MessagesDB
func NewMessageStore(db *sql.DB, lock *sync.Mutex) repo.MessageStore {
	return &MessagesDB{modelStore{db, lock}}
}

// Put will insert a record into the messages
func (o *MessagesDB) Put(messageID, orderID string, mType pb.Message_MessageType, peerID string, msg repo.Message, rErr string, receivedAt int64, pubkey []byte) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	stm := `insert or replace into messages(messageID, orderID, message_type, message, peerID, err, received_at, pubkey, created_at) values(?,?,?,?,?,?,?,?,?)`
	stmt, err := o.PrepareQuery(stm)
	if err != nil {
		return fmt.Errorf("prepare message sql: %s", err.Error())
	}
	defer stmt.Close()

	msg0, err := msg.MarshalJSON()
	if err != nil {
		return fmt.Errorf("marshal message: %s", err.Error())
	}

	_, err = stmt.Exec(
		messageID,
		orderID,
		int(mType),
		msg0,
		peerID,
		rErr,
		receivedAt,
		pubkey,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("err inserting message: %s", err.Error())
	}

	return nil
}

// GetByOrderIDType returns the message for the specified order and message type
func (o *MessagesDB) GetByOrderIDType(orderID string, mType pb.Message_MessageType) (*repo.Message, string, error) {
	o.lock.Lock()
	defer o.lock.Unlock()
	var (
		msg0   []byte
		peerID string
	)

	stmt, err := o.db.Prepare("select message, peerID from messages where orderID=? and message_type=?")
	if err != nil {
		return nil, "", err
	}
	err = stmt.QueryRow(orderID, mType).Scan(&msg0, &peerID)
	if err != nil {
		return nil, "", err
	}

	msg := new(repo.Message)

	if len(msg0) > 0 {
		err = msg.UnmarshalJSON(msg0)
		if err != nil {
			return nil, "", err
		}
	}

	return msg, peerID, nil
}

// GetAllErrored returns all messages which have an error state
func (o *MessagesDB) GetAllErrored() ([]repo.OrderMessage, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	stmt := `select messageID, orderID, message_type, message, peerID, err, pubkey from messages where err != ""`
	var ret []repo.OrderMessage
	rows, err := o.db.Query(stmt)
	if err != nil {
		return ret, err
	}
	defer rows.Close()

	for rows.Next() {
		var messageID, orderID, peerID, rErr string
		var msg0, pkey []byte
		var mType int32
		err = rows.Scan(&messageID, &orderID, &mType, &msg0, &peerID, &rErr, &pkey)
		if err != nil {
			log.Error(err)
		}
		ret = append(ret, repo.OrderMessage{
			PeerID:      peerID,
			MessageID:   messageID,
			OrderID:     orderID,
			MessageType: mType,
			Message:     msg0,
			MsgErr:      rErr,
			PeerPubkey:  pkey,
		})
	}
	return ret, nil
}

// MarkAsResolved sets a provided message as resolved
func (o *MessagesDB) MarkAsResolved(m repo.OrderMessage) error {
	var (
		stmt = `update messages set err = "" where messageID == ?`
		msg  = new(repo.Message)
	)

	if len(m.Message) > 0 {
		err := msg.UnmarshalJSON(m.Message)
		if err != nil {
			log.Errorf("failed extracting message (%+v): %s", m, err.Error())
			return err
		}
	}
	_, err := o.db.Exec(stmt, m.MessageID)
	if err != nil {
		return fmt.Errorf("marking msg (%s) as resolved: %s", m.MessageID, err.Error())
	}
	return nil
}
