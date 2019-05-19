// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
	"time"
)

// MsgElectionEntryPing implements the Message interface and represents a bitcoin
// dseep message.  It is used to enter a masternode into the election
// for rewards.
type MsgElectionEntryPing struct {
	Input         TxIn
	Signature     []byte
	SignatureTime time.Time
	Stop          bool
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgElectionEntryPing) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readTxIn(r, pver, 0, &msg.Input)
	if err != nil {
		return err
	}

	sig, err := ReadVarBytes(r, pver, 72, "signature")
	if err != nil {
		return err
	}
	msg.Signature = sig
	return readElements(r, (*uint32Time)(&msg.SignatureTime), &msg.Stop)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgElectionEntryPing) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeTxIn(w, pver, 0, &msg.Input)
	if err != nil {
		return err
	}
	if err := WriteVarBytes(w, pver, msg.Signature); err != nil {
		return err
	}
	return writeElements(w, (uint32Time)(msg.SignatureTime), msg.SignatureTime)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgElectionEntryPing) Command() string {
	return CmdElectionEntryPing
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgElectionEntryPing) MaxPayloadLength(pver uint32) uint32 {
	return 500
}

// NewMsgElectionEntryPing returns a new bitcoin feefilter message that conforms to
// the Message interface.  See MsgFeeFilter for details.
func NewMsgElectionEntryPing() *MsgElectionEntryPing {
	return &MsgElectionEntryPing{}
}
