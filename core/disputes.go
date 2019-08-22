package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	libp2p "gx/ipfs/QmTW4SdgBWq9GjsBsHeUx8WuGxzhgzAf88UMH2w62PC8yK/go-libp2p-crypto"
	"gx/ipfs/QmYVXrKrKHDC9FobgmcmshCDyWwdrfwfanNQN4oxJ9Fk3h/go-libp2p-peer"

	"github.com/OpenBazaar/openbazaar-go/net"
	"github.com/OpenBazaar/openbazaar-go/pb"
	"github.com/OpenBazaar/openbazaar-go/repo"
	"github.com/OpenBazaar/openbazaar-go/repo/db"
	"github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	hd "github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
)

// ConfirmationsPerHour is temporary until the Wallet interface has Attributes() to provide this value
const ConfirmationsPerHour = 6

// DisputeWg - waitgroup for disputes
var DisputeWg = new(sync.WaitGroup)

// ErrCaseNotFound - case not found err
var ErrCaseNotFound = errors.New("case not found")

// ErrCloseFailureCaseExpired - tried closing expired case err
var ErrCloseFailureCaseExpired = errors.New("unable to close expired case")

// ErrCloseFailureNoOutpoints indicates when a dispute cannot be closed due to neither party
// including outpoints with their dispute
var ErrCloseFailureNoOutpoints = errors.New("unable to close case with missing outpoints")

// ErrOpenFailureOrderExpired - tried disputing expired order err
var ErrOpenFailureOrderExpired = errors.New("unable to open case because order is too old to dispute")

// OpenDispute - open a dispute
func (n *OpenBazaarNode) OpenDispute(orderID string, contract *pb.RicardianContract, records []*wallet.TransactionRecord, claim string) error {
	if !n.verifyEscrowFundsAreDisputeable(contract, records) {
		return ErrOpenFailureOrderExpired
	}
	var isPurchase bool
	if n.IpfsNode.Identity.Pretty() == contract.BuyerOrder.BuyerID.PeerID {
		isPurchase = true
	}

	dispute := new(pb.Dispute)

	// Create timestamp
	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return err
	}
	dispute.Timestamp = ts

	// Add claim
	dispute.Claim = claim

	// Create outpoints
	var outpoints []*pb.Outpoint
	for _, r := range records {
		o := new(pb.Outpoint)
		o.Hash = strings.TrimPrefix(r.Txid, "0x")
		o.Index = r.Index
		o.NewValue = &pb.CurrencyValue{
			Currency: contract.BuyerOrder.Payment.AmountValue.Currency,
			Amount:   r.Value.String(),
		}
		outpoints = append(outpoints, o)
	}
	dispute.Outpoints = outpoints

	wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		return err
	}

	// Add payout address
	dispute.PayoutAddress = wal.CurrentAddress(wallet.EXTERNAL).EncodeAddress()

	// Serialize contract
	ser, err := proto.Marshal(contract)
	if err != nil {
		return err
	}
	dispute.SerializedContract = ser

	// Sign dispute
	rc := new(pb.RicardianContract)
	rc.Dispute = dispute
	rc, err = n.SignDispute(rc)
	if err != nil {
		return err
	}
	contract.Dispute = dispute
	contract.Signatures = append(contract.Signatures, rc.Signatures[0])

	// Send to moderator
	err = n.SendDisputeOpen(contract.BuyerOrder.Payment.Moderator, nil, rc)
	if err != nil {
		return err
	}

	// Send to counterparty
	var counterparty string
	var counterkey libp2p.PubKey
	if isPurchase {
		counterparty = contract.VendorListings[0].VendorID.PeerID
		counterkey, err = libp2p.UnmarshalPublicKey(contract.VendorListings[0].VendorID.Pubkeys.Identity)
		if err != nil {
			return nil
		}
	} else {
		counterparty = contract.BuyerOrder.BuyerID.PeerID
		counterkey, err = libp2p.UnmarshalPublicKey(contract.BuyerOrder.BuyerID.Pubkeys.Identity)
		if err != nil {
			return nil
		}
	}
	err = n.SendDisputeOpen(counterparty, &counterkey, rc)
	if err != nil {
		return err
	}

	// Update database
	if isPurchase {
		n.Datastore.Purchases().Put(orderID, *contract, pb.OrderState_DISPUTED, true)
	} else {
		n.Datastore.Sales().Put(orderID, *contract, pb.OrderState_DISPUTED, true)
	}
	return nil
}

func (n *OpenBazaarNode) verifyEscrowFundsAreDisputeable(contract *pb.RicardianContract, records []*wallet.TransactionRecord) bool {
	confirmationsForTimeout := contract.VendorListings[0].Metadata.EscrowTimeoutHours * ConfirmationsPerHour
	wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		log.Errorf("Failed verifyEscrowFundsAreDisputeable(): %s", err.Error())
		return false
	}
	for _, r := range records {
		hash, err := chainhash.NewHashFromStr(strings.TrimPrefix(r.Txid, "0x"))
		if err != nil {
			log.Errorf("Failed NewHashFromStr(%s): %s", r.Txid, err.Error())
			return false
		}
		actualConfirmations, _, err := wal.GetConfirmations(*hash)
		if err != nil {
			log.Errorf("Failed GetConfirmations(%s): %s", hash.String(), err.Error())
			return false
		}
		if actualConfirmations >= confirmationsForTimeout {
			return false
		}
	}
	return true
}

// SignDispute - sign the dispute
func (n *OpenBazaarNode) SignDispute(contract *pb.RicardianContract) (*pb.RicardianContract, error) {
	serializedDispute, err := proto.Marshal(contract.Dispute)
	if err != nil {
		return contract, err
	}
	s := new(pb.Signature)
	s.Section = pb.Signature_DISPUTE
	guidSig, err := n.IpfsNode.PrivateKey.Sign(serializedDispute)
	if err != nil {
		return contract, err
	}
	s.SignatureBytes = guidSig
	contract.Signatures = append(contract.Signatures, s)
	return contract, nil
}

// VerifySignatureOnDisputeOpen - verify signatures in an open dispute
func (n *OpenBazaarNode) VerifySignatureOnDisputeOpen(contract *pb.RicardianContract, peerID string) error {
	var pubkey []byte
	deser := new(pb.RicardianContract)
	err := proto.Unmarshal(contract.Dispute.SerializedContract, deser)
	if err != nil {
		return err
	}
	if len(deser.VendorListings) == 0 || deser.BuyerOrder == nil {
		return errors.New("invalid serialized contract")
	}
	if peerID == deser.BuyerOrder.BuyerID.PeerID {
		pubkey = deser.BuyerOrder.BuyerID.Pubkeys.Identity
	} else if peerID == deser.VendorListings[0].VendorID.PeerID {
		pubkey = deser.VendorListings[0].VendorID.Pubkeys.Identity
	} else {
		return errors.New("peer ID doesn't match either buyer or vendor")
	}

	if err := verifyMessageSignature(
		contract.Dispute,
		pubkey,
		contract.Signatures,
		pb.Signature_DISPUTE,
		peerID,
	); err != nil {
		switch err.(type) {
		case noSigError:
			return errors.New("contract does not contain a signature for the dispute")
		case invalidSigError:
			return errors.New("guid signature on contact failed to verify")
		case matchKeyError:
			return errors.New("public key in dispute does not match reported ID")
		default:
			return err
		}
	}
	return nil
}

// ProcessDisputeOpen - process an open dispute
func (n *OpenBazaarNode) ProcessDisputeOpen(rc *pb.RicardianContract, peerID string) error {
	DisputeWg.Add(1)
	defer DisputeWg.Done()

	if rc.Dispute == nil {
		return errors.New("dispute message is nil")
	}

	// Deserialize contract
	contract := new(pb.RicardianContract)
	err := proto.Unmarshal(rc.Dispute.SerializedContract, contract)
	if err != nil {
		return err
	}
	if len(contract.VendorListings) == 0 || contract.BuyerOrder == nil || contract.BuyerOrder.Payment == nil {
		return errors.New("serialized contract is malformatted")
	}

	orderID, err := n.CalcOrderID(contract.BuyerOrder)
	if err != nil {
		return err
	}

	wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		return err
	}

	var thumbnailTiny string
	var thumbnailSmall string
	var buyer string
	if len(contract.VendorListings) > 0 && contract.VendorListings[0].Item != nil && len(contract.VendorListings[0].Item.Images) > 0 {
		thumbnailTiny = contract.VendorListings[0].Item.Images[0].Tiny
		thumbnailSmall = contract.VendorListings[0].Item.Images[0].Small
		if contract.BuyerOrder != nil && contract.BuyerOrder.BuyerID != nil {
			buyer = contract.BuyerOrder.BuyerID.PeerID
		}
	}

	// Figure out what role we have in this dispute and process it
	var DisputerID string
	var DisputerHandle string
	var DisputeeID string
	var DisputeeHandle string
	if contract.BuyerOrder.Payment.Moderator == n.IpfsNode.Identity.Pretty() { // Moderator
		validationErrors := n.ValidateCaseContract(contract)
		var err error
		if contract.VendorListings[0].VendorID.PeerID == peerID {
			DisputerID = contract.VendorListings[0].VendorID.PeerID
			DisputerHandle = contract.VendorListings[0].VendorID.Handle
			DisputeeID = contract.BuyerOrder.BuyerID.PeerID
			DisputeeHandle = contract.BuyerOrder.BuyerID.Handle
			err = n.Datastore.Cases().Put(orderID, pb.OrderState_DISPUTED, false, rc.Dispute.Claim, db.PaymentCoinForContract(contract), db.CoinTypeForContract(contract))
			if err != nil {
				return err
			}
			err = n.Datastore.Cases().UpdateVendorInfo(orderID, contract, validationErrors, rc.Dispute.PayoutAddress, rc.Dispute.Outpoints)
			if err != nil {
				return err
			}
		} else if contract.BuyerOrder.BuyerID.PeerID == peerID {
			DisputerID = contract.BuyerOrder.BuyerID.PeerID
			DisputerHandle = contract.BuyerOrder.BuyerID.Handle
			DisputeeID = contract.VendorListings[0].VendorID.PeerID
			DisputeeHandle = contract.VendorListings[0].VendorID.Handle
			err = n.Datastore.Cases().Put(orderID, pb.OrderState_DISPUTED, true, rc.Dispute.Claim, db.PaymentCoinForContract(contract), db.CoinTypeForContract(contract))
			if err != nil {
				return err
			}
			err = n.Datastore.Cases().UpdateBuyerInfo(orderID, contract, validationErrors, rc.Dispute.PayoutAddress, rc.Dispute.Outpoints)
			if err != nil {
				return err
			}
		} else {
			return errors.New("peer ID doesn't match either buyer or vendor")
		}
		if err != nil {
			return err
		}
	} else if contract.VendorListings[0].VendorID.PeerID == n.IpfsNode.Identity.Pretty() { // Vendor
		DisputerID = contract.BuyerOrder.BuyerID.PeerID
		DisputerHandle = contract.BuyerOrder.BuyerID.Handle
		DisputeeID = contract.VendorListings[0].VendorID.PeerID
		DisputeeHandle = contract.VendorListings[0].VendorID.Handle
		// Load our version of the contract from the db
		myContract, state, _, records, _, _, err := n.Datastore.Sales().GetByOrderId(orderID)
		if err != nil {
			if err := n.SendProcessingError(DisputerID, orderID, pb.Message_DISPUTE_OPEN, nil); err != nil {
				log.Errorf("failed sending ORDER_PROCESSING_FAILURE to peer (%s): %s", DisputerID, err)
			}
			return net.OutOfOrderMessage
		}
		// Check this order is currently in a state which can be disputed
		if state == pb.OrderState_COMPLETED || state == pb.OrderState_DISPUTED || state == pb.OrderState_DECIDED || state == pb.OrderState_RESOLVED || state == pb.OrderState_REFUNDED || state == pb.OrderState_CANCELED || state == pb.OrderState_DECLINED || state == pb.OrderState_PROCESSING_ERROR {
			return errors.New("contract can no longer be disputed")
		}

		// Build dispute update message
		update := new(pb.DisputeUpdate)
		ser, err := proto.Marshal(myContract)
		if err != nil {
			return err
		}
		update.SerializedContract = ser
		update.OrderId = orderID
		update.PayoutAddress = wal.CurrentAddress(wallet.EXTERNAL).EncodeAddress()

		var outpoints []*pb.Outpoint
		for _, r := range records {
			o := new(pb.Outpoint)
			o.Hash = strings.TrimPrefix(r.Txid, "0x")
			o.Index = r.Index
			o.NewValue = &pb.CurrencyValue{
				Currency: myContract.BuyerOrder.Payment.AmountValue.Currency,
				Amount:   r.Value.String(),
			}
			outpoints = append(outpoints, o)
		}
		update.Outpoints = outpoints

		// Send the message
		err = n.SendDisputeUpdate(myContract.BuyerOrder.Payment.Moderator, update)
		if err != nil {
			return err
		}

		// Append the dispute and signature
		myContract.Dispute = rc.Dispute
		for _, sig := range rc.Signatures {
			if sig.Section == pb.Signature_DISPUTE {
				myContract.Signatures = append(myContract.Signatures, sig)
			}
		}
		// Save it back to the db with the new state
		err = n.Datastore.Sales().Put(orderID, *myContract, pb.OrderState_DISPUTED, false)
		if err != nil {
			return err
		}
	} else if contract.BuyerOrder.BuyerID.PeerID == n.IpfsNode.Identity.Pretty() { // Buyer
		DisputerID = contract.VendorListings[0].VendorID.PeerID
		DisputerHandle = contract.VendorListings[0].VendorID.Handle
		DisputeeID = contract.BuyerOrder.BuyerID.PeerID
		DisputeeHandle = contract.BuyerOrder.BuyerID.Handle

		// Load out version of the contract from the db
		myContract, state, _, records, _, _, err := n.Datastore.Purchases().GetByOrderId(orderID)
		if err != nil {
			return err
		}
		if state == pb.OrderState_AWAITING_PAYMENT || state == pb.OrderState_AWAITING_FULFILLMENT || state == pb.OrderState_PARTIALLY_FULFILLED || state == pb.OrderState_PENDING {
			if err := n.SendProcessingError(DisputerID, orderID, pb.Message_DISPUTE_OPEN, myContract); err != nil {
				log.Errorf("failed sending ORDER_PROCESSING_FAILURE to peer (%s): %s", DisputerID, err)
			}
			return net.OutOfOrderMessage
		}
		// Check this order is currently in a state which can be disputed
		if state == pb.OrderState_COMPLETED || state == pb.OrderState_DISPUTED || state == pb.OrderState_DECIDED || state == pb.OrderState_RESOLVED || state == pb.OrderState_REFUNDED || state == pb.OrderState_CANCELED || state == pb.OrderState_DECLINED {
			return errors.New("contract can no longer be disputed")
		}

		// Build dispute update message
		update := new(pb.DisputeUpdate)
		ser, err := proto.Marshal(myContract)
		if err != nil {
			return err
		}
		update.SerializedContract = ser
		update.OrderId = orderID
		update.PayoutAddress = wal.CurrentAddress(wallet.EXTERNAL).EncodeAddress()

		var outpoints []*pb.Outpoint
		for _, r := range records {
			o := new(pb.Outpoint)
			o.Hash = strings.TrimPrefix(r.Txid, "0x")
			o.Index = r.Index
			o.NewValue = &pb.CurrencyValue{
				Currency: myContract.BuyerOrder.Payment.AmountValue.Currency,
				Amount:   r.Value.String(),
			}
			outpoints = append(outpoints, o)
		}
		update.Outpoints = outpoints

		// Send the message
		err = n.SendDisputeUpdate(myContract.BuyerOrder.Payment.Moderator, update)
		if err != nil {
			return err
		}

		// Append the dispute and signature
		myContract.Dispute = rc.Dispute
		for _, sig := range rc.Signatures {
			if sig.Section == pb.Signature_DISPUTE {
				myContract.Signatures = append(myContract.Signatures, sig)
			}
		}
		// Save it back to the db with the new state
		err = n.Datastore.Purchases().Put(orderID, *myContract, pb.OrderState_DISPUTED, false)
		if err != nil {
			return err
		}
	} else {
		return errors.New("we are not involved in this dispute")
	}

	notif := repo.DisputeOpenNotification{
		ID:             repo.NewNotificationID(),
		Type:           "disputeOpen",
		OrderId:        orderID,
		Thumbnail:      repo.Thumbnail{Tiny: thumbnailTiny, Small: thumbnailSmall},
		DisputerID:     DisputerID,
		DisputerHandle: DisputerHandle,
		DisputeeID:     DisputeeID,
		DisputeeHandle: DisputeeHandle,
		Buyer:          buyer,
	}
	n.Broadcast <- notif
	n.Datastore.Notifications().PutRecord(repo.NewNotification(notif, time.Now(), false))
	return nil
}

// CloseDispute - close a dispute
func (n *OpenBazaarNode) CloseDispute(orderID string, buyerPercentage, vendorPercentage float32, resolution string, paymentCoinHint *repo.CurrencyCode) error {
	var payDivision = repo.PayoutRatio{Buyer: buyerPercentage, Vendor: vendorPercentage}
	if err := payDivision.Validate(); err != nil {
		return err
	}

	dispute, err := n.Datastore.Cases().GetByCaseID(orderID)
	if err != nil {
		return ErrCaseNotFound
	}

	if dispute.OrderState != pb.OrderState_DISPUTED {
		log.Errorf("unable to resolve expired dispute for order %s", orderID)
		return errors.New("A dispute for this order is not open")
	}
	if dispute.IsExpiredNow() {
		log.Errorf("unable to resolve expired dispute for order %s", orderID)
		return ErrCloseFailureCaseExpired
	}

	var outpoints = dispute.ResolutionPaymentOutpoints(payDivision)
	if outpoints == nil {
		log.Errorf("no outpoints to resolve in dispute for order %s", orderID)
		return ErrCloseFailureNoOutpoints
	}

	if dispute.VendorContract == nil && vendorPercentage > 0 {
		return errors.New("vendor must provide his copy of the contract before you can release funds to the vendor")
	}

	if dispute.BuyerContract == nil {
		dispute.BuyerContract = dispute.VendorContract
	}
	preferredContract := dispute.ResolutionPaymentContract(payDivision)

	// TODO: Remove once broken contracts are migrated
	paymentCoin := preferredContract.BuyerOrder.Payment.AmountValue.Currency.Code
	_, err = repo.LoadCurrencyDefinitions().Lookup(paymentCoin)
	if err != nil {
		log.Warningf("invalid BuyerOrder.Payment.Coin (%s) on order (%s)", paymentCoin, orderID)
		//preferredContract.BuyerOrder.Payment.Coin = paymentCoinHint.String()
	}

	var d = new(pb.DisputeResolution)

	// Add timestamp
	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return err
	}
	d.Timestamp = ts

	// Add orderId
	d.OrderId = orderID

	// Set self (moderator) as the party that made the resolution proposal
	d.ProposedBy = n.IpfsNode.Identity.Pretty()

	// Set resolution
	d.Resolution = resolution

	var (
		vendorID = preferredContract.VendorListings[0].VendorID.PeerID
		buyerID  = preferredContract.BuyerOrder.BuyerID.PeerID
	)
	buyerKey, err := libp2p.UnmarshalPublicKey(preferredContract.BuyerOrder.BuyerID.Pubkeys.Identity)
	if err != nil {
		return err
	}
	vendorKey, err := libp2p.UnmarshalPublicKey(preferredContract.VendorListings[0].VendorID.Pubkeys.Identity)
	if err != nil {
		return err
	}

	// Calculate total out value
	totalOut := big.NewInt(0)
	for _, o := range outpoints {
		n, ok := new(big.Int).SetString(o.NewValue.Amount, 10)
		if !ok {
			return errors.New("invalid total out amount")
		}
		totalOut.Add(totalOut, n)
	}

	wal, err := n.Multiwallet.WalletForCurrencyCode(preferredContract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		return err
	}

	// Create outputs using full value. We will subtract the fee off each output later.
	outMap := make(map[string]wallet.TransactionOutput)
	var outputs []wallet.TransactionOutput
	var modAddr btcutil.Address
	var modValue big.Int
	modAddr = wal.CurrentAddress(wallet.EXTERNAL)
	modValue, err = n.GetModeratorFee(*totalOut, preferredContract.BuyerOrder.Payment.AmountValue.Currency.Code, wal.CurrencyCode())
	if err != nil {
		return err
	}
	if modValue.Cmp(big.NewInt(0)) > 0 {
		out := wallet.TransactionOutput{
			Address: modAddr,
			Value:   modValue,
		}
		outputs = append(outputs, out)
		outMap["moderator"] = out
	}

	var buyerAddr btcutil.Address
	buyerValue := big.NewInt(0)
	effectiveVal := new(big.Int).Sub(totalOut, &modValue)
	if payDivision.BuyerAny() {
		buyerAddr, err = wal.DecodeAddress(dispute.BuyerPayoutAddress)
		if err != nil {
			return err
		}
		buyerValue = new(big.Int).Mul(effectiveVal, big.NewInt(int64(buyerPercentage)))
		buyerValue = buyerValue.Div(buyerValue, big.NewInt(100))
		out := wallet.TransactionOutput{
			Address: buyerAddr,
			Value:   *buyerValue,
		}
		outputs = append(outputs, out)
		outMap["buyer"] = out
	}
	var vendorAddr btcutil.Address
	vendorValue := big.NewInt(0)
	if payDivision.VendorAny() {
		vendorAddr, err = wal.DecodeAddress(dispute.VendorPayoutAddress)
		if err != nil {
			return err
		}
		vendorValue = new(big.Int).Mul(effectiveVal, big.NewInt(int64(vendorPercentage)))
		vendorValue = vendorValue.Div(vendorValue, big.NewInt(100))
		out := wallet.TransactionOutput{
			Address: vendorAddr,
			Value:   *vendorValue,
		}
		outputs = append(outputs, out)
		outMap["vendor"] = out
	}

	if len(outputs) == 0 {
		return errors.New("transaction has no outputs")
	}

	// Create inputs
	var inputs []wallet.TransactionInput
	for _, o := range outpoints {
		decodedHash, err := hex.DecodeString(o.Hash)
		if err != nil {
			return err
		}
		n, ok := new(big.Int).SetString(o.NewValue.Amount, 10)
		if !ok {
			return errors.New("invalid amount")
		}
		input := wallet.TransactionInput{
			OutpointHash:  decodedHash,
			OutpointIndex: o.Index,
			Value:         *n,
		}
		inputs = append(inputs, input)
	}

	if len(inputs) == 0 {
		return errors.New("transaction has no inputs")
	}

	// Calculate total fee
	defaultFee := wal.GetFeePerByte(wallet.NORMAL)
	txFee := wal.EstimateFee(inputs, outputs, dispute.ResolutionPaymentFeePerByte(payDivision, defaultFee))

	// Subtract fee from each output in proportion to output value
	var outs []wallet.TransactionOutput
	for role, output := range outMap {
		outPercentage := new(big.Float).Quo(new(big.Float).SetInt(&output.Value), new(big.Float).SetInt(totalOut))
		outputShareOfFee := new(big.Float).Mul(outPercentage, new(big.Float).SetInt(&txFee))
		valF := new(big.Float).Sub(new(big.Float).SetInt(&output.Value), outputShareOfFee)
		val, _ := valF.Int(nil)
		if !wal.IsDust(*val) {
			o := wallet.TransactionOutput{
				Value:   *val,
				Address: output.Address,
				Index:   output.Index,
			}
			outs = append(outs, o)
		} else {
			delete(outMap, role)
		}
	}

	// Create moderator key
	chaincode := preferredContract.BuyerOrder.Payment.Chaincode
	chaincodeBytes, err := hex.DecodeString(chaincode)
	if err != nil {
		return err
	}
	mECKey, err := n.MasterPrivateKey.ECPrivKey()
	if err != nil {
		return err
	}
	moderatorKey, err := wal.ChildKey(mECKey.Serialize(), chaincodeBytes, true)
	if err != nil {
		return err
	}

	// Sign buyer rating key
	if dispute.BuyerContract != nil {
		ecPriv, err := moderatorKey.ECPrivKey()
		if err != nil {
			return err
		}
		for _, key := range dispute.BuyerContract.BuyerOrder.RatingKeys {
			hashed := sha256.Sum256(key)
			sig, err := ecPriv.Sign(hashed[:])
			if err != nil {
				return err
			}
			d.ModeratorRatingSigs = append(d.ModeratorRatingSigs, sig.Serialize())
		}
	}

	// Create signatures
	redeemScript := preferredContract.BuyerOrder.Payment.RedeemScript
	redeemScriptBytes, err := hex.DecodeString(redeemScript)
	if err != nil {
		return err
	}
	sigs, err := wal.CreateMultisigSignature(inputs, outs, moderatorKey, redeemScriptBytes, *big.NewInt(0))
	if err != nil {
		return err
	}
	var bitcoinSigs []*pb.BitcoinSignature
	for _, sig := range sigs {
		s := new(pb.BitcoinSignature)
		s.InputIndex = sig.InputIndex
		s.Signature = sig.Signature
		bitcoinSigs = append(bitcoinSigs, s)
	}

	// Create payout object
	payout := new(pb.DisputeResolution_Payout)
	payout.Inputs = outpoints
	payout.Sigs = bitcoinSigs
	if _, ok := outMap["buyer"]; ok {
		f := new(big.Float).Quo(new(big.Float).SetInt(buyerValue), new(big.Float).SetInt(totalOut))
		outputShareOfFeeF := new(big.Float).Mul(f, new(big.Float).SetInt(&txFee))
		outputShareOfFeeInt, _ := outputShareOfFeeF.Int(nil)
		amt := new(big.Int).Sub(buyerValue, outputShareOfFeeInt)
		if amt.Cmp(big.NewInt(0)) < 0 {
			amt = big.NewInt(0)
		}
		payout.BuyerOutput = &pb.DisputeResolution_Payout_Output{
			ScriptOrAddress: &pb.DisputeResolution_Payout_Output_Address{Address: buyerAddr.String()},
			AmountValue: &pb.CurrencyValue{
				Currency: preferredContract.BuyerOrder.Payment.AmountValue.Currency,
				Amount:   amt.String(),
			},
		}
	}
	if _, ok := outMap["vendor"]; ok {
		f := new(big.Float).Quo(new(big.Float).SetInt(vendorValue), new(big.Float).SetInt(totalOut))
		outputShareOfFeeF := new(big.Float).Mul(f, new(big.Float).SetInt(&txFee))
		outputShareOfFeeInt, _ := outputShareOfFeeF.Int(nil)
		amt := new(big.Int).Sub(vendorValue, outputShareOfFeeInt)
		if amt.Cmp(big.NewInt(0)) < 0 {
			amt = big.NewInt(0)
		}
		payout.VendorOutput = &pb.DisputeResolution_Payout_Output{
			ScriptOrAddress: &pb.DisputeResolution_Payout_Output_Address{Address: vendorAddr.String()},
			AmountValue: &pb.CurrencyValue{
				Currency: preferredContract.BuyerOrder.Payment.AmountValue.Currency,
				Amount:   amt.String(),
			},
		}
	}
	if _, ok := outMap["moderator"]; ok {
		f := new(big.Float).Quo(new(big.Float).SetInt(&modValue), new(big.Float).SetInt(totalOut))
		outputShareOfFeeF := new(big.Float).Mul(f, new(big.Float).SetInt(&txFee))
		outputShareOfFeeInt, _ := outputShareOfFeeF.Int(nil)
		amt := new(big.Int).Sub(&modValue, outputShareOfFeeInt)
		if amt.Cmp(big.NewInt(0)) < 0 {
			amt = big.NewInt(0)
		}
		payout.ModeratorOutput = &pb.DisputeResolution_Payout_Output{
			ScriptOrAddress: &pb.DisputeResolution_Payout_Output_Address{Address: modAddr.String()},
			AmountValue: &pb.CurrencyValue{
				Currency: preferredContract.BuyerOrder.Payment.AmountValue.Currency,
				Amount:   amt.String(),
			},
		}
	}

	d.Payout = payout

	rc := new(pb.RicardianContract)
	rc.DisputeResolution = d
	rc, err = n.SignDisputeResolution(rc)
	if err != nil {
		return err
	}

	err = n.SendDisputeClose(buyerID, &buyerKey, rc)
	if err != nil {
		return err
	}
	err = n.SendDisputeClose(vendorID, &vendorKey, rc)
	if err != nil {
		return err
	}

	err = n.Datastore.Cases().MarkAsClosed(orderID, d)
	if err != nil {
		return err
	}
	return nil
}

// SignDisputeResolution - add signature to DisputeResolution
func (n *OpenBazaarNode) SignDisputeResolution(contract *pb.RicardianContract) (*pb.RicardianContract, error) {
	serializedDR, err := proto.Marshal(contract.DisputeResolution)
	if err != nil {
		return contract, err
	}
	s := new(pb.Signature)
	s.Section = pb.Signature_DISPUTE_RESOLUTION
	guidSig, err := n.IpfsNode.PrivateKey.Sign(serializedDR)
	if err != nil {
		return contract, err
	}
	s.SignatureBytes = guidSig
	contract.Signatures = append(contract.Signatures, s)
	return contract, nil
}

// ValidateCaseContract - validate contract details
func (n *OpenBazaarNode) ValidateCaseContract(contract *pb.RicardianContract) []string {
	var validationErrors []string

	// Contract should have a listing and order to make it to this point
	if len(contract.VendorListings) == 0 {
		validationErrors = append(validationErrors, "Contract contains no listings")
	}
	if contract.BuyerOrder == nil {
		validationErrors = append(validationErrors, "Contract is missing the buyer's order")
	}

	if contract.VendorListings[0].VendorID == nil || contract.VendorListings[0].VendorID.Pubkeys == nil {
		validationErrors = append(validationErrors, "The listing is missing the vendor ID information. Unable to validate any signatures.")
		return validationErrors
	}
	if contract.BuyerOrder.BuyerID == nil || contract.BuyerOrder.BuyerID.Pubkeys == nil {
		validationErrors = append(validationErrors, "The listing is missing the buyer ID information. Unable to validate any signatures.")
		return validationErrors
	}

	vendorPubkey := contract.VendorListings[0].VendorID.Pubkeys.Identity
	vendorGUID := contract.VendorListings[0].VendorID.PeerID

	buyerPubkey := contract.BuyerOrder.BuyerID.Pubkeys.Identity
	buyerGUID := contract.BuyerOrder.BuyerID.PeerID

	// Make sure the order contains a payment object
	if contract.BuyerOrder.Payment == nil {
		validationErrors = append(validationErrors, "The buyer's order is missing the payment section")
	}

	// There needs to be one listing for each unique item in the order
	var listingHashes []string
	for _, item := range contract.BuyerOrder.Items {
		listingHashes = append(listingHashes, item.ListingHash)
	}
	for _, listing := range contract.VendorListings {
		ser, err := proto.Marshal(listing)
		if err != nil {
			continue
		}
		listingMH, err := EncodeCID(ser)
		if err != nil {
			continue
		}
		for i, l := range listingHashes {
			if l == listingMH.String() {
				// Delete from listingHases
				listingHashes = append(listingHashes[:i], listingHashes[i+1:]...)
				break
			}
		}
	}
	// This should have a length of zero if there is one vendorListing for each item in buyerOrder
	if len(listingHashes) > 0 {
		validationErrors = append(validationErrors, "Not all items in the order have a matching vendor listing")
	}

	// There needs to be one listing signature for each listing
	var listingSigs []*pb.Signature
	for _, sig := range contract.Signatures {
		if sig.Section == pb.Signature_LISTING {
			listingSigs = append(listingSigs, sig)
		}
	}
	if len(listingSigs) < len(contract.VendorListings) {
		validationErrors = append(validationErrors, "Not all listings are signed by the vendor")
	}

	// Verify the listing signatures
	for i, listing := range contract.VendorListings {
		if err := verifyMessageSignature(listing, vendorPubkey, []*pb.Signature{listingSigs[i]}, pb.Signature_LISTING, vendorGUID); err != nil {
			validationErrors = append(validationErrors, "Invalid vendor signature on listing "+strconv.Itoa(i)+err.Error())
		}
		if i == len(listingSigs)-1 {
			break
		}
	}

	// Verify the order signature
	if err := verifyMessageSignature(contract.BuyerOrder, buyerPubkey, contract.Signatures, pb.Signature_ORDER, buyerGUID); err != nil {
		validationErrors = append(validationErrors, "Invalid buyer signature on order")
	}

	// Verify the order confirmation signature
	if contract.VendorOrderConfirmation != nil {
		if err := verifyMessageSignature(contract.VendorOrderConfirmation, vendorPubkey, contract.Signatures, pb.Signature_ORDER_CONFIRMATION, vendorGUID); err != nil {
			validationErrors = append(validationErrors, "Invalid vendor signature on order confirmation")
		}
	}

	// There should be one fulfillment signature for each vendorOrderFulfilment object
	var fulfilmentSigs []*pb.Signature
	for _, sig := range contract.Signatures {
		if sig.Section == pb.Signature_ORDER_FULFILLMENT {
			fulfilmentSigs = append(fulfilmentSigs, sig)
		}
	}
	if len(fulfilmentSigs) < len(contract.VendorOrderFulfillment) {
		validationErrors = append(validationErrors, "Not all order fulfilments are signed by the vendor")
	}

	// Verify the signature of the order fulfilments
	for i, f := range contract.VendorOrderFulfillment {
		if err := verifyMessageSignature(f, vendorPubkey, []*pb.Signature{fulfilmentSigs[i]}, pb.Signature_ORDER_FULFILLMENT, vendorGUID); err != nil {
			validationErrors = append(validationErrors, "Invalid vendor signature on fulfilment "+strconv.Itoa(i))
		}
		if i == len(fulfilmentSigs)-1 {
			break
		}
	}

	// Verify the buyer's bitcoin signature on his guid
	if err := verifyBitcoinSignature(
		contract.BuyerOrder.BuyerID.Pubkeys.Bitcoin,
		contract.BuyerOrder.BuyerID.BitcoinSig,
		contract.BuyerOrder.BuyerID.PeerID,
	); err != nil {
		validationErrors = append(validationErrors, "The buyer's bitcoin signature which covers his guid is invalid. This could be an attempt to forge the buyer's identity.")
	}

	// Verify the vendor's bitcoin signature on his guid
	if err := verifyBitcoinSignature(
		contract.VendorListings[0].VendorID.Pubkeys.Bitcoin,
		contract.VendorListings[0].VendorID.BitcoinSig,
		contract.VendorListings[0].VendorID.PeerID,
	); err != nil {
		validationErrors = append(validationErrors, "The vendor's bitcoin signature which covers his guid is invalid. This could be an attempt to forge the vendor's identity.")
	}

	// Verify the redeem script matches all the bitcoin keys
	if contract.BuyerOrder.Payment != nil {
		wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
		if err != nil {
			validationErrors = append(validationErrors, "Contract uses a coin not found in wallet")
			return validationErrors
		}
		chaincode, err := hex.DecodeString(contract.BuyerOrder.Payment.Chaincode)
		if err != nil {
			validationErrors = append(validationErrors, "Error validating bitcoin address and redeem script")
			return validationErrors
		}
		mECKey, err := n.MasterPrivateKey.ECPubKey()
		if err != nil {
			validationErrors = append(validationErrors, "Error validating bitcoin address and redeem script")
			return validationErrors
		}
		moderatorKey, err := wal.ChildKey(mECKey.SerializeCompressed(), chaincode, false)
		if err != nil {
			validationErrors = append(validationErrors, "Error validating bitcoin address and redeem script")
			return validationErrors
		}
		buyerKey, err := wal.ChildKey(contract.BuyerOrder.BuyerID.Pubkeys.Bitcoin, chaincode, false)
		if err != nil {
			validationErrors = append(validationErrors, "Error validating bitcoin address and redeem script")
			return validationErrors
		}
		vendorKey, err := wal.ChildKey(contract.VendorListings[0].VendorID.Pubkeys.Bitcoin, chaincode, false)
		if err != nil {
			validationErrors = append(validationErrors, "Error validating bitcoin address and redeem script")
			return validationErrors
		}
		timeout, _ := time.ParseDuration(strconv.Itoa(int(contract.VendorListings[0].Metadata.EscrowTimeoutHours)) + "h")
		addr, redeemScript, err := wal.GenerateMultisigScript([]hd.ExtendedKey{*buyerKey, *vendorKey, *moderatorKey}, 2, timeout, vendorKey)
		if err != nil {
			validationErrors = append(validationErrors, "Error generating multisig script")
			return validationErrors
		}

		if strings.TrimPrefix(contract.BuyerOrder.Payment.Address, "0x") != strings.TrimPrefix(addr.String(), "0x") {
			validationErrors = append(validationErrors, "The calculated bitcoin address doesn't match the address in the order")
		}

		if hex.EncodeToString(redeemScript) != contract.BuyerOrder.Payment.RedeemScript {
			validationErrors = append(validationErrors, "The calculated redeem script doesn't match the redeem script in the order")
		}
	}

	return validationErrors
}

// ValidateDisputeResolution - validate dispute resolution
func (n *OpenBazaarNode) ValidateDisputeResolution(contract *pb.RicardianContract) error {
	err := n.verifySignatureOnDisputeResolution(contract)
	if err != nil {
		return err
	}
	if contract.DisputeResolution.Payout == nil || len(contract.DisputeResolution.Payout.Sigs) == 0 {
		return errors.New("DisputeResolution contains invalid payout")
	}
	wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		return err
	}

	if contract.VendorListings[0].VendorID.PeerID == n.IpfsNode.Identity.Pretty() && contract.DisputeResolution.Payout.VendorOutput != nil {
		return n.verifyPaymentDestinationIsInWallet(contract.DisputeResolution.Payout.VendorOutput, wal)
	} else if contract.BuyerOrder.BuyerID.PeerID == n.IpfsNode.Identity.Pretty() && contract.DisputeResolution.Payout.BuyerOutput != nil {
		return n.verifyPaymentDestinationIsInWallet(contract.DisputeResolution.Payout.BuyerOutput, wal)
	}
	return nil
}

func (n *OpenBazaarNode) verifyPaymentDestinationIsInWallet(output *pb.DisputeResolution_Payout_Output, wal wallet.Wallet) error {
	addr, err := pb.DisputeResolutionPayoutOutputToAddress(wal, output)
	if err != nil {
		return err
	}

	if !wal.HasKey(addr) {
		return errors.New("moderator dispute resolution payout address is not defined in your wallet to recieve funds")
	}
	return nil
}

func (n *OpenBazaarNode) verifySignatureOnDisputeResolution(contract *pb.RicardianContract) error {

	moderatorID, err := peer.IDB58Decode(contract.BuyerOrder.Payment.Moderator)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	pubkey, err := n.DHT.GetPublicKey(ctx, moderatorID)
	if err != nil {
		log.Errorf("Failed to find public key for %s", moderatorID.Pretty())
		return err
	}
	pubKeyBytes, err := pubkey.Bytes()
	if err != nil {
		return err
	}

	if err := verifyMessageSignature(
		contract.DisputeResolution,
		pubKeyBytes,
		contract.Signatures,
		pb.Signature_DISPUTE_RESOLUTION,
		moderatorID.Pretty(),
	); err != nil {
		switch err.(type) {
		case noSigError:
			return errors.New("contract does not contain a signature for the dispute resolution")
		case invalidSigError:
			return errors.New("guid signature on contact failed to verify")
		case matchKeyError:
			return errors.New("public key in dispute does not match reported ID")
		default:
			return err
		}
	}
	return nil
}

// ReleaseFunds - release funds
func (n *OpenBazaarNode) ReleaseFunds(contract *pb.RicardianContract, records []*wallet.TransactionRecord) error {
	orderID, err := n.CalcOrderID(contract.BuyerOrder)
	if err != nil {
		return err
	}

	// Create inputs
	var inputs []wallet.TransactionInput
	for _, o := range contract.DisputeResolution.Payout.Inputs {
		decodedHash, err := hex.DecodeString(strings.TrimPrefix(o.Hash, "0x"))
		if err != nil {
			return err
		}
		n, ok := new(big.Int).SetString(o.NewValue.Amount, 10)
		if !ok {
			return errors.New("invalid payout input")
		}
		input := wallet.TransactionInput{
			OutpointHash:  decodedHash,
			OutpointIndex: o.Index,
			Value:         *n,
			OrderID:       orderID,
		}
		inputs = append(inputs, input)
	}

	if len(inputs) == 0 {
		return errors.New("transaction has no inputs")
	}
	wal, err := n.Multiwallet.WalletForCurrencyCode(contract.BuyerOrder.Payment.AmountValue.Currency.Code)
	if err != nil {
		return err
	}

	// Create outputs
	var outputs []wallet.TransactionOutput
	if contract.DisputeResolution.Payout.BuyerOutput != nil {
		addr, err := pb.DisputeResolutionPayoutOutputToAddress(wal, contract.DisputeResolution.Payout.BuyerOutput)
		if err != nil {
			return err
		}
		n, ok := new(big.Int).SetString(contract.DisputeResolution.Payout.BuyerOutput.AmountValue.Amount, 10)
		if !ok {
			return errors.New("invalid payout amount")
		}
		output := wallet.TransactionOutput{
			Address: addr,
			Value:   *n,
			OrderID: orderID,
		}
		outputs = append(outputs, output)
	}
	if contract.DisputeResolution.Payout.VendorOutput != nil {
		addr, err := pb.DisputeResolutionPayoutOutputToAddress(wal, contract.DisputeResolution.Payout.VendorOutput)
		if err != nil {
			return err
		}
		n, ok := new(big.Int).SetString(contract.DisputeResolution.Payout.VendorOutput.AmountValue.Amount, 10)
		if !ok {
			return errors.New("invalid payout amount")
		}
		output := wallet.TransactionOutput{
			Address: addr,
			Value:   *n,
			OrderID: orderID,
		}
		outputs = append(outputs, output)
	}
	if contract.DisputeResolution.Payout.ModeratorOutput != nil {
		addr, err := pb.DisputeResolutionPayoutOutputToAddress(wal, contract.DisputeResolution.Payout.ModeratorOutput)
		if err != nil {
			return err
		}
		n, ok := new(big.Int).SetString(contract.DisputeResolution.Payout.ModeratorOutput.AmountValue.Amount, 10)
		if !ok {
			return errors.New("invalid payout amount")
		}
		output := wallet.TransactionOutput{
			Address: addr,
			Value:   *n,
			OrderID: orderID,
		}
		outputs = append(outputs, output)
	}

	// Create signing key
	chaincodeBytes, err := hex.DecodeString(contract.BuyerOrder.Payment.Chaincode)
	if err != nil {
		return err
	}
	mECKey, err := n.MasterPrivateKey.ECPrivKey()
	if err != nil {
		return err
	}
	signingKey, err := wal.ChildKey(mECKey.Serialize(), chaincodeBytes, true)
	if err != nil {
		return err
	}

	// Create signatures
	redeemScriptBytes, err := hex.DecodeString(contract.BuyerOrder.Payment.RedeemScript)
	if err != nil {
		return err
	}
	mySigs, err := wal.CreateMultisigSignature(inputs, outputs, signingKey, redeemScriptBytes, *big.NewInt(0))
	if err != nil {
		return err
	}

	var moderatorSigs []wallet.Signature
	for _, sig := range contract.DisputeResolution.Payout.Sigs {
		s := wallet.Signature{
			Signature:  sig.Signature,
			InputIndex: sig.InputIndex,
		}
		moderatorSigs = append(moderatorSigs, s)
	}

	accept := new(pb.DisputeAcceptance)
	// Create timestamp
	ts, err := ptypes.TimestampProto(time.Now())
	if err != nil {
		return err
	}
	accept.Timestamp = ts
	accept.ClosedBy = n.IpfsNode.Identity.Pretty()
	contract.DisputeAcceptance = accept

	peerID := contract.BuyerOrder.BuyerID.PeerID

	// Update database
	if n.IpfsNode.Identity.Pretty() == contract.BuyerOrder.BuyerID.PeerID {
		err = n.Datastore.Purchases().Put(orderID, *contract, pb.OrderState_DECIDED, true)
		peerID = contract.VendorListings[0].VendorID.PeerID
	} else {
		err = n.Datastore.Sales().Put(orderID, *contract, pb.OrderState_DECIDED, true)
	}
	if err != nil {
		log.Errorf("ReleaseFunds error updating database: %s", err.Error())
	}

	// Build, sign, and broadcast transaction
	txnID, err := wal.Multisign(inputs, outputs, mySigs, moderatorSigs, redeemScriptBytes, *big.NewInt(0), true)
	if err != nil {
		return err
	}

	msg := pb.OrderPaymentTxn{
		Coin:          contract.BuyerOrder.Payment.AmountValue.Currency.Code,
		OrderID:       orderID,
		TransactionID: strings.TrimPrefix(hexutil.Encode(txnID), "0x"),
		WithInput:     true,
	}

	err = n.SendOrderPayment(peerID, &msg)
	if err != nil {
		log.Errorf("error sending order payment: %v", err)
	}

	return nil
}
