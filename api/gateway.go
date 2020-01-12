package api

import (
	"net"
	"net/http"

	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/op/go-logging"
	"github.com/phoreproject/openbazaar-go/core"
	"github.com/phoreproject/openbazaar-go/schema"
)

var log = logging.MustGetLogger("api")

// Gateway represents an HTTP API gateway
type Gateway struct {
	listener net.Listener
	handler  http.Handler
	config   schema.APIConfig
	gatewayRunning chan error
}

// NewGateway instantiates a new `Gateway`
func NewGateway(n *core.OpenBazaarNode, authCookie http.Cookie, l net.Listener, config schema.APIConfig, logger logging.Backend, options ...corehttp.ServeOption) (*Gateway, error) {

	log.SetBackend(logging.AddModuleLevel(logger))
	topMux := http.NewServeMux()

	jsonAPI := newJSONAPIHandler(n, authCookie, config)
	wsAPI := newWSAPIHandler(n, authCookie, config)
	n.Broadcast = manageNotifications(n, wsAPI.h.Broadcast)

	topMux.Handle("/ob/", jsonAPI)
	topMux.Handle("/wallet/", jsonAPI)
	topMux.Handle("/manage/", jsonAPI)
	topMux.Handle("/ws", wsAPI)

	var (
		err error
		mux = topMux
	)
	for _, option := range options {
		mux, err = option(n.IpfsNode, l, mux)
		if err != nil {
			return nil, err
		}
	}

	return &Gateway{
		listener: l,
		handler:  topMux,
		config:   config,
	}, nil
}

// Close shutsdown the Gateway listener
func (g *Gateway) Close() error {
	log.Infof("server at %s terminating...", g.listener.Addr())
	return g.listener.Close()
}

// Serve begins listening on the configured address
func (g *Gateway) serve() error {
	var err error
	if g.config.SSL {
		err = http.ListenAndServeTLS(g.listener.Addr().String(), g.config.SSLCert, g.config.SSLKey, g.handler)
	} else {
		err = http.Serve(g.listener, g.handler)
	}
	return err
}

func (g *Gateway) Serve(blocking bool) error {
	if g.gatewayRunning == nil {
		g.gatewayRunning = make(chan error)
		go func() {
			err := g.serve()
			if err != nil {
				g.gatewayRunning <- err
			} else {
				g.gatewayRunning <- nil
			}
		}()
	}

	if blocking {
		return <-g.gatewayRunning
	}

	return nil
}