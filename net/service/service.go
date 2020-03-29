package service

import (
	"context"
	"errors"

	inet "gx/ipfs/QmY3ArotKMKaL7YGfbQfyDrib6RVraLqZYWXZvVgZktBxp/go-libp2p-net"
	peer "gx/ipfs/QmYVXrKrKHDC9FobgmcmshCDyWwdrfwfanNQN4oxJ9Fk3h/go-libp2p-peer"
	host "gx/ipfs/QmYrWiWM4qtrnCeT3R14jY3ZZyirDNJgwK57q4qFYePgbd/go-libp2p-host"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	ps "gx/ipfs/QmaCTz9RkrU13bm9kMB54f7atgqM4qkjDZpRwRoJiWXEqs/go-libp2p-peerstore"
	ggio "gx/ipfs/QmddjPSGZb3ieihSseFeCfVRpZzcqczPNsD2DvarSwnjJB/gogo-protobuf/io"

	"io"
	"sync"
	"time"

	ctxio "github.com/jbenet/go-context/io"
	"github.com/op/go-logging"
	"github.com/phoreproject/pm-go/core"
	"github.com/phoreproject/pm-go/ipfs"
	"github.com/phoreproject/pm-go/pb"
	"github.com/phoreproject/pm-go/repo"
)

var log = logging.MustGetLogger("service")

type OpenBazaarService struct {
	host      host.Host
	self      peer.ID
	peerstore ps.Peerstore
	ctx       context.Context
	broadcast chan repo.Notifier
	datastore repo.Datastore
	node      *core.OpenBazaarNode
	sender    map[peer.ID]*messageSender
	senderlk  sync.Mutex
}

func New(node *core.OpenBazaarNode, datastore repo.Datastore) *OpenBazaarService {
	service := &OpenBazaarService{
		host:      node.IpfsNode.PeerHost.(host.Host),
		self:      node.IpfsNode.Identity,
		peerstore: node.IpfsNode.PeerHost.Peerstore(),
		ctx:       node.IpfsNode.Context(),
		broadcast: node.Broadcast,
		datastore: datastore,
		node:      node,
		sender:    make(map[peer.ID]*messageSender),
	}
	node.IpfsNode.PeerHost.SetStreamHandler(protocol.ID(ipfs.IPFSProtocolAppMainnetOne), service.HandleNewStream)
	log.Infof("OpenBazaar service running at %s", ipfs.IPFSProtocolAppMainnetOne)
	return service
}

func (service *OpenBazaarService) WaitForReady() {
	<-service.node.DHT.BootstrapChan
}

func (service *OpenBazaarService) DisconnectFromPeer(p peer.ID) error {
	log.Debugf("Disconnecting from %s", p.Pretty())
	service.senderlk.Lock()
	defer service.senderlk.Unlock()
	ms, ok := service.sender[p]
	if !ok {
		return nil
	}
	if ms != nil && ms.s != nil {
		ms.s.Close()
	}
	delete(service.sender, p)
	return nil
}

func (service *OpenBazaarService) HandleNewStream(s inet.Stream) {
	go service.handleNewMessage(s)
}

func (service *OpenBazaarService) handleNewMessage(s inet.Stream) {
	defer s.Close()
	cr := ctxio.NewReader(service.ctx, s) // ok to use. we defer close stream in this func
	r := ggio.NewDelimitedReader(cr, inet.MessageSizeMax)
	mPeer := s.Conn().RemotePeer()

	// Check if banned
	if service.node.BanManager.IsBanned(mPeer) {
		return
	}

	ms, err := service.messageSenderForPeer(service.ctx, mPeer)
	if err != nil {
		log.Error("Error getting message sender")
		return
	}

	for {
		select {
		// end loop on context close
		case <-service.ctx.Done():
			return
		default:
		}
		// Receive msg
		pmes := new(pb.Message)
		if err := r.ReadMsg(pmes); err != nil {
			s.Reset()
			if err == io.EOF {
				log.Debugf("Disconnected from peer %s", mPeer.Pretty())
			}
			return
		}

		if pmes.IsResponse {
			ms.requestlk.Lock()
			ch, ok := ms.requests[pmes.RequestId]
			if ok {
				// this is a request response
				select {
				case ch <- pmes:
					// message returned to requester
				case <-time.After(time.Second):
					// in case ch is closed on the other end - the lock should prevent this happening
					log.Debug("request id was not removed from map on timeout")
				}
				close(ch)
				delete(ms.requests, pmes.RequestId)
			} else {
				log.Debug("received response message with unknown request id: requesting function may have timed out")
			}
			ms.requestlk.Unlock()
			s.Reset()
			return
		}

		// Get handler for this msg type
		handler := service.HandlerForMsgType(pmes.MessageType)
		if handler == nil {
			s.Reset()
			log.Debug("Got back nil handler from handlerForMsgType")
			return
		}

		// Dispatch handler
		rpmes, err := handler(mPeer, pmes, nil)
		if err != nil {
			log.Debugf("%s handle message error: %s", pmes.MessageType.String(), err)
		}

		// If nil response, return it before serializing
		if rpmes == nil {
			continue
		}

		// give back request id
		rpmes.RequestId = pmes.RequestId
		rpmes.IsResponse = true

		// send out response msg
		if err := ms.SendMessage(service.ctx, rpmes); err != nil {
			s.Reset()
			log.Debugf("send response error: %s", err)
			return
		}
	}
}

func (service *OpenBazaarService) SendRequest(ctx context.Context, p peer.ID, pmes *pb.Message) (*pb.Message, error) {
	log.Debugf("Sending %s request to %s", pmes.MessageType.String(), p.Pretty())
	ms, err := service.messageSenderForPeer(ctx, p)
	if err != nil {
		return nil, err
	}

	rpmes, err := ms.SendRequest(ctx, pmes)
	if err != nil {
		log.Debugf("No response from %s", p.Pretty())
		return nil, err
	}

	if rpmes == nil {
		log.Debugf("No response from %s", p.Pretty())
		return nil, errors.New("no response from peer")
	}

	log.Debugf("Received response from %s", p.Pretty())
	return rpmes, nil
}

func (service *OpenBazaarService) SendMessage(ctx context.Context, p peer.ID, pmes *pb.Message) error {
	if pmes.MessageType != pb.Message_BLOCK {
		log.Debugf("Sending %s message to %s", pmes.MessageType.String(), p.Pretty())
	}
	ms, err := service.messageSenderForPeer(ctx, p)
	if err != nil {
		return err
	}

	if err := ms.SendMessage(ctx, pmes); err != nil {
		return err
	}
	return nil
}
