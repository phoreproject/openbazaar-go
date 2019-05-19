// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

// MsgSyncStatusCount implements the Message interface and represents a bitcoin pong
// message which is used to sync the number of different assets available.
type MsgSyncStatusCount struct {
	ItemID int32
	Count  int32
}

// BtcDecode decodes r using the bitcoin protocol encoding into the receiver.
// This is part of the Message interface implementation.
func (msg *MsgSyncStatusCount) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readElements(r, &msg.ItemID, &msg.Count)
}

// BtcEncode encodes the receiver to w using the bitcoin protocol encoding.
// This is part of the Message interface implementation.
func (msg *MsgSyncStatusCount) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeElements(w, msg.ItemID, msg.Count)
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgSyncStatusCount) Command() string {
	return CmdSyncStatusCount
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgSyncStatusCount) MaxPayloadLength(pver uint32) uint32 {
	return 16
}

// NewMsgSyncStatusCount returns a new phore ssc message that conforms to the Message
// interface.  See MsgSyncStatusCount for details.
func NewMsgSyncStatusCount() *MsgSyncStatusCount {
	return &MsgSyncStatusCount{}
}
