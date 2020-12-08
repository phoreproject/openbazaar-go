package repo

import (
	"github.com/golang/protobuf/proto"
	"github.com/phoreproject/pm-go/pb"
	"math/big"
)

// ToV5Dispute scans through the dispute looking for any deprecated fields and
// turns them into their v5 counterpart.
func ToV5Dispute(dispute *pb.Dispute) *pb.Dispute {
	newDispute := proto.Clone(dispute).(*pb.Dispute)

	for i, outpoint := range dispute.Outpoints {
		if outpoint.Value != 0 && outpoint.BigValue == "" {
			newDispute.Outpoints[i].BigValue = big.NewInt(int64(outpoint.Value)).String()
			newDispute.Outpoints[i].Value = 0
		}
	}
	return newDispute
}
