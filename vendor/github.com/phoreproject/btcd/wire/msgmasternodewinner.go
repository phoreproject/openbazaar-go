// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

// MsgMasternodeWinner repesents a masternode winner at a certain
// block height with a given payee and signature
type MsgMasternodeWinner struct {
	MasternodeInput TxIn // Type of data
	BlockHeight     int
	Payee           []byte
	Signature       []byte
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgMasternodeWinner) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readTxIn(r, pver, 0, &msg.MasternodeInput)
	if err != nil {
		return err
	}

	err = readElement(r, &msg.BlockHeight)
	if err != nil {
		return err
	}

	p, err := ReadVarBytes(r, pver, 100000, "payee")
	if err != nil {
		return err
	}
	msg.Payee = p

	s, err := ReadVarBytes(r, pver, 100000, "signature")
	if err != nil {
		return err
	}
	msg.Signature = s
	return nil
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgMasternodeWinner) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeTxIn(w, pver, 0, &msg.MasternodeInput)
	if err != nil {
		return err
	}

	err = writeElement(w, msg.BlockHeight)
	if err != nil {
		return err
	}

	err = WriteVarBytes(w, pver, msg.Payee)
	if err != nil {
		return err
	}

	return WriteVarBytes(w, pver, msg.Signature)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgMasternodeWinner) Command() string {
	return CmdMasternodeWinner
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgMasternodeWinner) MaxPayloadLength(pver uint32) uint32 {
	return 10000000 // TODO
}

// NewMsgMasternodeWinner returns a new bitcoin pong message that conforms to the Message
// interface.  See MsgPong for details.
func NewMsgMasternodeWinner() *MsgMasternodeWinner {
	return &MsgMasternodeWinner{}
}
