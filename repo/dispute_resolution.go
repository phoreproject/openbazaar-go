package repo

import (
	"math/big"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/pm-go/pb"
)

// ToV5DisputeResolution scans through the dispute resolution looking for any deprecated fields and
// turns them into their v5 counterpart.
func ToV5DisputeResolution(disputeResolution *pb.DisputeResolution) *pb.DisputeResolution {
	newDisputeResolution := proto.Clone(disputeResolution).(*pb.DisputeResolution)
	if disputeResolution.Payout == nil {
		return newDisputeResolution
	}

	for i, input := range disputeResolution.Payout.Inputs {
		if input.Value != 0 && input.BigValue == "" {
			input.BigValue = strconv.FormatUint(input.Value, 10)
			newDisputeResolution.Payout.Inputs[i] = input
		}
	}

	if disputeResolution.Payout.BuyerOutput != nil &&
		disputeResolution.Payout.BuyerOutput.Amount != 0 &&
		disputeResolution.Payout.BuyerOutput.BigAmount == "" {
		newDisputeResolution.Payout.BuyerOutput.BigAmount = big.NewInt(int64(disputeResolution.Payout.BuyerOutput.Amount)).String()
		newDisputeResolution.Payout.BuyerOutput.Amount = 0
	}
	if disputeResolution.Payout.VendorOutput != nil &&
		disputeResolution.Payout.VendorOutput.Amount != 0 &&
		disputeResolution.Payout.VendorOutput.BigAmount == "" {
		newDisputeResolution.Payout.VendorOutput.BigAmount = big.NewInt(int64(disputeResolution.Payout.VendorOutput.Amount)).String()
		newDisputeResolution.Payout.VendorOutput.Amount = 0
	}
	if disputeResolution.Payout.ModeratorOutput != nil &&
		disputeResolution.Payout.ModeratorOutput.Amount != 0 &&
		disputeResolution.Payout.ModeratorOutput.BigAmount == "" {
		newDisputeResolution.Payout.ModeratorOutput.BigAmount = big.NewInt(int64(disputeResolution.Payout.ModeratorOutput.Amount)).String()
		newDisputeResolution.Payout.ModeratorOutput.Amount = 0
	}
	return newDisputeResolution
}
