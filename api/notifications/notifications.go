package notifications

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	mh "gx/ipfs/QmU9a9NV9RdPNwZQDYd5uKsm6N6LJLSvLbywDDYFbaaC6P/go-multihash"
	"time"
)

// Notification is a single notification to send a user.
type Notification struct {
	Data      Data      `json:"notification"`
	Timestamp time.Time `json:"timestamp"`
	Read      bool      `json:"read"`
}

// Thumbnail is URLs to a thumbnail for some notifications
type Thumbnail struct {
	Tiny  string `json:"tiny"`
	Small string `json:"small"`
}

// Data is the actual data used for a notification.
type Data interface {
	// TODO maybe should be made 'real interface', which will allow
	// to use typed channels, type checking and semantic dispatching
	// instead of typecase:

	// Serialize() []byte
	// Describe() (string, string)
}

type notificationWrapper struct {
	Notification Data `json:"notification"`
}

type messageWrapper struct {
	Message Data `json:"message"`
}

type walletWrapper struct {
	Message Data `json:"wallet"`
}

type messageReadWrapper struct {
	MessageRead Data `json:"messageRead"`
}

type messageTypingWrapper struct {
	MessageRead Data `json:"messageTyping"`
}

// OrderNotification is sent when a user orders an item.
type OrderNotification struct {
	ID          string    `json:"notificationId"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	BuyerID     string    `json:"buyerId"`
	BuyerHandle string    `json:"buyerHandle"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	OrderID     string    `json:"orderId"`
	Slug        string    `json:"slug"`
}

// PaymentNotification is sent when a user pays for an item.
type PaymentNotification struct {
	ID           string `json:"notificationId"`
	Type         string `json:"type"`
	OrderID      string `json:"orderId"`
	FundingTotal uint64 `json:"fundingTotal"`
}

// OrderConfirmationNotification is sent when an order is confirmed.
type OrderConfirmationNotification struct {
	ID           string    `json:"notificationId"`
	Type         string    `json:"type"`
	OrderID      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

// OrderDeclinedNotification is sent when an order is declined.
type OrderDeclinedNotification struct {
	ID           string    `json:"notificationId"`
	Type         string    `json:"type"`
	OrderID      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

// OrderCancelNotification is sent when a user cancels their order.
type OrderCancelNotification struct {
	ID          string    `json:"notificationId"`
	Type        string    `json:"type"`
	OrderID     string    `json:"orderId"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	BuyerHandle string    `json:"buyerHandle"`
	BuyerID     string    `json:"buyerId"`
}

// RefundNotification is sent when an order is refunded.
type RefundNotification struct {
	ID           string    `json:"notificationId"`
	Type         string    `json:"type"`
	OrderID      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

// FulfillmentNotification is sent when an order is fulfilled
type FulfillmentNotification struct {
	ID           string    `json:"notificationId"`
	Type         string    `json:"type"`
	OrderID      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

type ProcessingErrorNotification struct {
	ID           string    `json:"notificationId"`
	Type         string    `json:"type"`
	OrderId      string    `json:"orderId"`
	Thumbnail    Thumbnail `json:"thumbnail"`
	VendorHandle string    `json:"vendorHandle"`
	VendorID     string    `json:"vendorId"`
}

// CompletionNotification is sent when an order is completed.
type CompletionNotification struct {
	ID          string    `json:"notificationId"`
	Type        string    `json:"type"`
	OrderID     string    `json:"orderId"`
	Thumbnail   Thumbnail `json:"thumbnail"`
	BuyerHandle string    `json:"buyerHandle"`
	BuyerID     string    `json:"buyerId"`
}

// DisputeOpenNotification is sent when an order is disputed.
type DisputeOpenNotification struct {
	ID             string    `json:"notificationId"`
	Type           string    `json:"type"`
	OrderID        string    `json:"orderId"`
	Thumbnail      Thumbnail `json:"thumbnail"`
	DisputerID     string    `json:"disputerId"`
	DisputerHandle string    `json:"disputerHandle"`
	DisputeeID     string    `json:"disputeeId"`
	DisputeeHandle string    `json:"disputeeHandle"`
	Buyer          string    `json:"buyer"`
}

// DisputeUpdateNotification is sent when a dispute is updated.
type DisputeUpdateNotification struct {
	ID             string    `json:"notificationId"`
	Type           string    `json:"type"`
	OrderID        string    `json:"orderId"`
	Thumbnail      Thumbnail `json:"thumbnail"`
	DisputerID     string    `json:"disputerId"`
	DisputerHandle string    `json:"disputerHandle"`
	DisputeeID     string    `json:"disputeeId"`
	DisputeeHandle string    `json:"disputeeHandle"`
	Buyer          string    `json:"buyer"`
}

// DisputeCloseNotification is sent when a dispute is closed.
type DisputeCloseNotification struct {
	ID               string    `json:"notificationId"`
	Type             string    `json:"type"`
	OrderID          string    `json:"orderId"`
	Thumbnail        Thumbnail `json:"thumbnail"`
	OtherPartyID     string    `json:"otherPartyId"`
	OtherPartyHandle string    `json:"otherPartyHandle"`
	Buyer            string    `json:"buyer"`
}

// DisputeAcceptedNotification is sent when a dispute is accepted.
type DisputeAcceptedNotification struct {
	ID               string    `json:"notificationId"`
	Type             string    `json:"type"`
	OrderID          string    `json:"orderId"`
	Thumbnail        Thumbnail `json:"thumbnail"`
	OtherPartyID     string    `json:"otherPartyId"`
	OtherPartyHandle string    `json:"otherPartyHandle"`
	Buyer            string    `json:"buyer"`
}

// FollowNotification is sent when a user follows you.
type FollowNotification struct {
	ID     string `json:"notificationId"`
	Type   string `json:"type"`
	PeerID string `json:"peerId"`
}

// UnfollowNotification is sent when a user unfollows you.
type UnfollowNotification struct {
	ID     string `json:"notificationId"`
	Type   string `json:"type"`
	PeerID string `json:"peerId"`
}

// ModeratorAddNotification is sent when a moderator is added.
type ModeratorAddNotification struct {
	ID     string `json:"notificationId"`
	Type   string `json:"type"`
	PeerID string `json:"peerId"`
}

// ModeratorRemoveNotification is sent when a moderator is removed.
type ModeratorRemoveNotification struct {
	ID     string `json:"notificationId"`
	Type   string `json:"type"`
	PeerID string `json:"peerId"`
}

// StatusNotification is sent when the status changes.
type StatusNotification struct {
	Status string `json:"status"`
}

// ChatMessage is sent when you receive a chat message.
type ChatMessage struct {
	MessageID string    `json:"messageId"`
	PeerID    string    `json:"peerId"`
	Subject   string    `json:"subject"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatRead is sent when someone reads a chat message.
type ChatRead struct {
	MessageID string `json:"messageId"`
	PeerID    string `json:"peerId"`
	Subject   string `json:"subject"`
}

// ChatTyping is sent when someone in a chat starts typing
type ChatTyping struct {
	PeerID  string `json:"peerId"`
	Subject string `json:"subject"`
}

// IncomingTransaction is sent when you receive a new transaction
type IncomingTransaction struct {
	Txid          string    `json:"txid"`
	Value         int64     `json:"value"`
	Address       string    `json:"address"`
	Status        string    `json:"status"`
	Memo          string    `json:"memo"`
	Timestamp     time.Time `json:"timestamp"`
	Confirmations int32     `json:"confirmations"`
	OrderID       string    `json:"orderId"`
	Thumbnail     string    `json:"thumbnail"`
	Height        int32     `json:"height"`
	CanBumpFee    bool      `json:"canBumpFee"`
}

// TestNotification does nothing.
type TestNotification struct{}

// NewID generates a new unique ID.
func NewID() string {
	b := make([]byte, 32)
	rand.Read(b)
	encoded, _ := mh.Encode(b, mh.SHA2_256)
	nID, _ := mh.Cast(encoded)
	return nID.B58String()
}

func wrap(i interface{}) interface{} {
	switch i.(type) {
	case OrderNotification:
		n := i.(OrderNotification)
		n.Type = "order"
		return notificationWrapper{n}
	case PaymentNotification:
		n := i.(PaymentNotification)
		n.Type = "payment"
		return notificationWrapper{n}
	case OrderConfirmationNotification:
		n := i.(OrderConfirmationNotification)
		n.Type = "orderConfirmation"
		return notificationWrapper{n}
	case OrderCancelNotification:
		n := i.(OrderCancelNotification)
		n.Type = "cancel"
		return notificationWrapper{n}
	case RefundNotification:
		n := i.(RefundNotification)
		n.Type = "refund"
		return notificationWrapper{n}
	case FulfillmentNotification:
		n := i.(FulfillmentNotification)
		n.Type = "fulfillment"
		return notificationWrapper{n}
	case ProcessingErrorNotification:
		n := i.(ProcessingErrorNotification)
		n.Type = "processingError"
		return notificationWrapper{n}
	case CompletionNotification:
		n := i.(CompletionNotification)
		n.Type = "orderComplete"
		return notificationWrapper{n}
	case DisputeOpenNotification:
		n := i.(DisputeOpenNotification)
		n.Type = "disputeOpen"
		return notificationWrapper{n}
	case DisputeUpdateNotification:
		n := i.(DisputeUpdateNotification)
		n.Type = "disputeUpdate"
		return notificationWrapper{n}
	case DisputeCloseNotification:
		n := i.(DisputeCloseNotification)
		n.Type = "disputeClose"
		return notificationWrapper{n}
	case DisputeAcceptedNotification:
		n := i.(DisputeAcceptedNotification)
		n.Type = "disputeAccepted"
		return notificationWrapper{n}
	case FollowNotification:
		n := i.(FollowNotification)
		n.Type = "follow"
		return notificationWrapper{n}
	case UnfollowNotification:
		n := i.(UnfollowNotification)
		n.Type = "unfollow"
		return notificationWrapper{n}
	case ModeratorAddNotification:
		n := i.(ModeratorAddNotification)
		n.Type = "moderatorAdd"
		return notificationWrapper{n}
	case ModeratorRemoveNotification:
		n := i.(ModeratorRemoveNotification)
		n.Type = "moderatorRemove"
		return notificationWrapper{n}
	case ChatMessage:
		return messageWrapper{i.(ChatMessage)}
	case ChatRead:
		return messageReadWrapper{i.(ChatRead)}
	case ChatTyping:
		return messageTypingWrapper{i.(ChatTyping)}
	case IncomingTransaction:
		return walletWrapper{i.(IncomingTransaction)}
	default:
		return i
	}
}

// Serialize serializes a notification.
func Serialize(i interface{}) []byte {
	w := wrap(i)
	if _, ok := w.([]byte); ok {
		return w.([]byte)
	}
	b, _ := json.MarshalIndent(w, "", "    ")
	return b
}

// Describe prints a notification
func Describe(i interface{}) (string, string) {
	var head, body string
	switch i.(type) {
	case OrderNotification:
		head = "Order received"

		n := i.(OrderNotification)
		var buyer string
		if n.BuyerHandle != "" {
			buyer = n.BuyerHandle
		} else {
			buyer = n.BuyerID
		}
		form := "You received an order \"%s\".\n\nOrder ID: %s\nBuyer: %s\nThumbnail: %s\n"
		body = fmt.Sprintf(form, n.Title, n.OrderID, buyer, n.Thumbnail.Small)

	case PaymentNotification:
		head = "Payment received"

		n := i.(PaymentNotification)
		form := "Payment for order \"%s\" received (total %d)."
		body = fmt.Sprintf(form, n.OrderID, n.FundingTotal)

	case OrderConfirmationNotification:
		head = "Order confirmed"

		n := i.(OrderConfirmationNotification)
		form := "Order \"%s\" has been confirmed."
		body = fmt.Sprintf(form, n.OrderID)

	case OrderCancelNotification:
		head = "Order cancelled"

		n := i.(OrderCancelNotification)
		form := "Order \"%s\" has been cancelled."
		body = fmt.Sprintf(form, n.OrderID)

	case RefundNotification:
		head = "Payment refunded"

		n := i.(RefundNotification)
		form := "Payment refund for order \"%s\" received."
		body = fmt.Sprintf(form, n.OrderID)

	case FulfillmentNotification:
		head = "Order fulfilled"

		n := i.(FulfillmentNotification)
		form := "Order \"%s\" was marked as fulfilled."
		body = fmt.Sprintf(form, n.OrderID)

	case CompletionNotification:
		head = "Order completed"

		n := i.(CompletionNotification)
		form := "Order \"%s\" was marked as completed."
		body = fmt.Sprintf(form, n.OrderID)

	case DisputeOpenNotification:
		head = "Dispute opened"

		n := i.(DisputeOpenNotification)
		form := "Dispute around order \"%s\" was opened."
		body = fmt.Sprintf(form, n.OrderID)

	case DisputeUpdateNotification:
		head = "Dispute updated"

		n := i.(DisputeUpdateNotification)
		form := "Dispute around order \"%s\" was updated."
		body = fmt.Sprintf(form, n.OrderID)

	case DisputeCloseNotification:
		head = "Dispute closed"

		n := i.(DisputeCloseNotification)
		form := "Dispute around order \"%s\" was closed."
		body = fmt.Sprintf(form, n.OrderID)

	case TestNotification:
		head = "SMTP Notification Test"
		body = "Hello World"
	}

	return head, body
}
