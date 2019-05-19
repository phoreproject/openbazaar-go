// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
	"time"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
)

// MsgBudgetProposalBroadcast repesents a budget proposal
// on the Phore network
type MsgBudgetProposalBroadcast struct {
	ProposalName       string
	ProposalURL        string
	Time               time.Time
	BlockStart         int
	BlockEnd           int
	Amount             int64
	Payee              []byte
	FeeTransactionHash chainhash.Hash
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgBudgetProposalBroadcast) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	pn, err := ReadVarString(r, pver)
	if err != nil {
		return err
	}
	msg.ProposalName = pn
	pu, err := ReadVarString(r, pver)
	if err != nil {
		return err
	}
	msg.ProposalURL = pu
	err = readElements(r, (*uint32Time)(&msg.Time), &msg.BlockStart, &msg.BlockEnd, &msg.Amount)
	if err != nil {
		return err
	}
	pa, err := ReadVarBytes(r, pver, 500, "payee")
	if err != nil {
		return err
	}
	msg.Payee = pa
	return readElement(r, &msg.FeeTransactionHash)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgBudgetProposalBroadcast) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := WriteVarString(w, pver, msg.ProposalName)
	if err != nil {
		return err
	}
	err = WriteVarString(w, pver, msg.ProposalURL)
	if err != nil {
		return err
	}
	err = writeElements(w, msg.Time, msg.BlockStart, msg.BlockEnd, msg.Amount)
	if err != nil {
		return err
	}
	err = WriteVarBytes(w, pver, msg.Payee)
	if err != nil {
		return err
	}
	return writeElement(w, msg.FeeTransactionHash)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgBudgetProposalBroadcast) Command() string {
	return CmdMasternodeProposal
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgBudgetProposalBroadcast) MaxPayloadLength(pver uint32) uint32 {
	return 100000 // TODO
}

// NewMsgBudgetProposalBroadcast returns a new Phore message to advertise a
// budget proposal to peers.
func NewMsgBudgetProposalBroadcast() *MsgBudgetProposalBroadcast {
	return &MsgBudgetProposalBroadcast{}
}
