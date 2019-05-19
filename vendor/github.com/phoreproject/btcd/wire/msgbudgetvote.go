// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
	"time"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
)

// MsgBudgetVote repesents a budget proposal
// on the Phore network
type MsgBudgetVote struct {
	Input        TxIn
	ProposalHash chainhash.Hash
	Vote         int
	Time         time.Time
	Signature    []byte
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgBudgetVote) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readTxIn(r, pver, 0, &msg.Input)
	if err != nil {
		return err
	}
	err = readElements(r, pver, &msg.ProposalHash, &msg.Vote, (*uint32Time)(&msg.Time))
	if err != nil {
		return err
	}
	s, err := ReadVarBytes(r, pver, 72, "signature")
	if err != nil {
		return err
	}
	msg.Signature = s
	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgBudgetVote) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeTxIn(w, pver, 0, &msg.Input)
	if err != nil {
		return err
	}
	err = writeElements(w, pver, msg.ProposalHash, msg.Vote, (uint32Time)(msg.Time))
	if err != nil {
		return err
	}
	return WriteVarBytes(w, pver, msg.Signature)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgBudgetVote) Command() string {
	return CmdMasternodeVote
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgBudgetVote) MaxPayloadLength(pver uint32) uint32 {
	return 1000000 // TODO
}

// NewMsgBudgetVote returns a new bitcoin pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgBudgetVote() *MsgBudgetVote {
	return &MsgBudgetVote{}
}
