package factory

import (
	"github.com/phoreproject/pm-go/pb"
	"github.com/phoreproject/pm-go/repo"
	"github.com/golang/protobuf/ptypes/any"
)

func NewMessageWithOrderPayload() repo.Message {
	payload := []byte("test payload")
	return repo.Message{
		Msg: pb.Message{
			MessageType: pb.Message_ORDER,
			Payload:     &any.Any{Value: payload},
		},
	}
}
