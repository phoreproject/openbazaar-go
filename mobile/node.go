package mobile

import (
	"context"
	"errors"
	"fmt"
	"gx/ipfs/QmRCrPXk2oUwpK1Cj2FXrUotRpddUxz56setkny2gz13Cx/go-libp2p-routing-helpers"
	"gx/ipfs/QmSY3nkMNLzh9GdbFKK5tT7YMfLpf52iUZ8ZRkr29MJaa5/go-libp2p-kad-dht"
	"gx/ipfs/QmSY3nkMNLzh9GdbFKK5tT7YMfLpf52iUZ8ZRkr29MJaa5/go-libp2p-kad-dht/opts"
	ma "gx/ipfs/QmTZBfrPJmjWsCvHEtX5FE6KimVJhsJg5sBbqEFYf4UZtL/go-multiaddr"
	ds "gx/ipfs/QmUadX5EcvrBmxAV9sE7wUWtWSqxns5K84qKJBixmcT1w9/go-datastore"
	"gx/ipfs/QmYVXrKrKHDC9FobgmcmshCDyWwdrfwfanNQN4oxJ9Fk3h/go-libp2p-peer"
	p2phost "gx/ipfs/QmYrWiWM4qtrnCeT3R14jY3ZZyirDNJgwK57q4qFYePgbd/go-libp2p-host"
	"gx/ipfs/QmYxUdYY9S6yg5tSPVin5GFTvtfsLauVcr7reHDD3dM8xf/go-libp2p-routing"
	"gx/ipfs/QmbeHtaBy9nZsW4cHRcvgVY4CnDhXudE2Dr6qDxS7yg9rX/go-libp2p-record"
	"gx/ipfs/Qmc85NSvmSG4Frn9Vb2cBc1rMyULH6D3TNVEfCzSKoUpip/go-multiaddr-net"
	"gx/ipfs/QmddjPSGZb3ieihSseFeCfVRpZzcqczPNsD2DvarSwnjJB/gogo-protobuf/proto"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	ipfsconfig "gx/ipfs/QmUAuYuiafnJRZxDDX7MuruMNsicYNuyub5vUeAcupUBNs/go-ipfs-config"
	ipnspb "gx/ipfs/QmUwMnKKjH3JwGKNVZ3TcP37W93xzqNA4ECFFiMo6sXkkc/go-ipns/pb"

	wi "github.com/OpenBazaar/wallet-interface"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/ipfs/go-ipfs/commands"
	ipfscore "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/corehttp"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/op/go-logging"
	"github.com/phoreproject/multiwallet/util"
	"github.com/phoreproject/openbazaar-go/api"
	"github.com/phoreproject/openbazaar-go/core"
	"github.com/phoreproject/openbazaar-go/ipfs"
	obnet "github.com/phoreproject/openbazaar-go/net"
	rep "github.com/phoreproject/openbazaar-go/net/repointer"
	ret "github.com/phoreproject/openbazaar-go/net/retriever"
	"github.com/phoreproject/openbazaar-go/net/service"
	"github.com/phoreproject/openbazaar-go/repo"
	"github.com/phoreproject/openbazaar-go/repo/db"
	"github.com/phoreproject/openbazaar-go/repo/migrations"
	"github.com/phoreproject/openbazaar-go/schema"
	apiSchema "github.com/phoreproject/openbazaar-go/schema"
	"github.com/phoreproject/openbazaar-go/storage/selfhosted"
	"github.com/phoreproject/openbazaar-go/wallet"
	lis "github.com/phoreproject/openbazaar-go/wallet/listeners"
	"github.com/phoreproject/openbazaar-go/wallet/resync"
	"github.com/tyler-smith/go-bip39"
)

var log = logging.MustGetLogger("mobile")

// Node configuration structure
type Node struct {
	OpenBazaarNode *core.OpenBazaarNode
	config         NodeConfig
	cancel         context.CancelFunc
	ipfsConfig     *ipfscore.BuildCfg
	apiConfig      *apiSchema.APIConfig
}

// NewNode create the configuration file for a new node
func NewNode(repoPath string, authenticationToken string, testnet bool, userAgent string, walletTrustedPeer string, password string, mnemonic string) *Node {
	// Node config
	nodeconfig := &NodeConfig{
		RepoPath:            repoPath,
		AuthenticationToken: "",
		Testnet:             testnet,
		UserAgent:           userAgent,
		WalletTrustedPeer:   walletTrustedPeer,
	}

	// Use Mobile struct to carry config data
	node, err := NewNodeWithConfig(nodeconfig, password, mnemonic)
	if err != nil {
		fmt.Println(err)
	}
	return node
}

// NewNodeWithConfig create a new node using the configuration file from NewNode()
func NewNodeWithConfig(config *NodeConfig, password string, mnemonic string) (*Node, error) {
	// Lockfile
	repoLockFile := filepath.Join(config.RepoPath, fsrepo.LockFile)
	os.Remove(repoLockFile)

	// Logging
	backendStdout := logging.NewLogBackend(os.Stdout, "", 0)
	logger = logging.NewBackendFormatter(backendStdout, stdoutLogFormat)
	logging.SetBackend(logger)

	migrations.WalletCoinType = util.ExtendCoinType(config.CoinType)

	sqliteDB, err := initializeRepo(config.RepoPath, "", "", true, time.Now(), config.CoinType)
	if err != nil && err != repo.ErrRepoExists {
		return nil, err
	}

	// Get creation date. Ignore the error and use a default timestamp.
	creationDate, _ := sqliteDB.Config().GetCreationDate()

	// Load configs
	configFile, err := ioutil.ReadFile(path.Join(config.RepoPath, "config"))
	if err != nil {
		return nil, err
	}

	apiConfig, err := apiSchema.GetAPIConfig(configFile)
	if err != nil {
		return nil, err
	}

	dataSharing, err := apiSchema.GetDataSharing(configFile)
	if err != nil {
		return nil, err
	}

	walletsConfig, err := apiSchema.GetWalletsConfig(configFile)
	if err != nil {
		return nil, err
	}

	ipnsExtraConfig, err := apiSchema.GetIPNSExtraConfig(configFile)
	if err != nil {
		return nil, err
	}

	// Create user-agent file
	userAgentBytes := []byte(core.USERAGENT + config.UserAgent)
	ioutil.WriteFile(path.Join(config.RepoPath, "root", "user_agent"), userAgentBytes, os.ModePerm)

	// IPFS node setup
	r, err := fsrepo.Open(config.RepoPath)
	if err != nil {
		return nil, err
	}

	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}

	identityKey, err := sqliteDB.Config().GetIdentityKey()
	if err != nil {
		return nil, err
	}
	identity, err := ipfs.IdentityFromKey(identityKey)
	if err != nil {
		return nil, err
	}
	cfg.Identity = identity
	cfg.Swarm.DisableNatPortMap = true

	// Setup testnet
	if config.Testnet {
		// set testnet bootstrap addrs
		testnetBootstrapAddrs, err := apiSchema.GetTestnetBootstrapAddrs(configFile)
		if err != nil {
			log.Error(err)
			return nil, err
		}
		cfg.Bootstrap = testnetBootstrapAddrs

		// don't use pushnodes on testnet
		dataSharing.PushTo = []string{}
	}

	// Mnemonic
	mn, _, err := sqliteDB.Config().GetMnemonic()
	if err != nil {
		return nil, err
	}
	params := chaincfg.MainNetParams

	// Master key setup
	seed := bip39.NewSeed(mn, "")
	mPrivKey, err := hdkeychain.NewMaster(seed, &params)
	if err != nil {
		return nil, err
	}

	// Multiwallet setup
	multiwalletConfig := &wallet.WalletConfig{
		ConfigFile:           walletsConfig,
		DB:                   sqliteDB.DB(),
		Params:               &params,
		RepoPath:             config.RepoPath,
		Logger:               logger,
		WalletCreationDate:   creationDate,
		Mnemonic:             mn,
		DisableExchangeRates: config.DisableExchangerates,
	}
	mw, err := wallet.NewMultiWallet(multiwalletConfig)
	if err != nil {
		return nil, err
	}

	core.PublishLock.Lock()

	// Set up the ban manager
	settings, err := sqliteDB.Settings().Get()
	if err != nil && err != db.SettingsNotSetError {
		return nil, err
	}
	var blockedNodes []peer.ID
	if settings.BlockedNodes != nil {
		for _, pid := range *settings.BlockedNodes {
			id, err := peer.IDB58Decode(pid)
			if err != nil {
				continue
			}
			blockedNodes = append(blockedNodes, id)
		}
	}
	bm := obnet.NewBanManager(blockedNodes)

	// Push nodes
	var pushNodes []peer.ID
	for _, pnd := range dataSharing.PushTo {
		p, err := peer.IDB58Decode(pnd)
		if err != nil {
			return nil, err
		}
		pushNodes = append(pushNodes, p)
	}

	// OpenBazaar node setup
	core.Node = &core.OpenBazaarNode{
		BanManager:                    bm,
		Datastore:                     sqliteDB,
		MasterPrivateKey:              mPrivKey,
		Multiwallet:                   mw,
		OfflineMessageFailoverTimeout: 5 * time.Second,
		PushNodes:                     pushNodes,
		RepoPath:                      config.RepoPath,
		UserAgent:                     core.USERAGENT,
		IPNSQuorumSize:                uint(ipnsExtraConfig.DHTQuorumSize),
	}

	if len(cfg.Addresses.Gateway) <= 0 {
		return nil, errors.New("no gateway addresses configured")
	}

	// override with mobile routing config
	ignoredURI := ""
	ncfg := ipfs.PrepareIPFSConfig(r, ignoredURI, config.Testnet, config.Testnet)
	ncfg.Routing = constructMobileRouting

	return &Node{OpenBazaarNode: core.Node, config: *config, ipfsConfig: ncfg, apiConfig: apiConfig}, nil
}

func constructMobileRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching, validator record.Validator) (routing.IpfsRouting, error) {
	dhtRouting, err := dht.New(
		ctx, host,
		dhtopts.Client(true),
		dhtopts.Datastore(dstore),
		dhtopts.Validator(validator),
	)
	if err != nil {
		return nil, err
	}
	apiRouter := ipfs.NewAPIRouter(schema.IPFSCachingRouterDefaultURI, dhtRouting.Validator)
	apiRouter.Start(nil)
	cachingRouter := ipfs.NewCachingRouter(dhtRouting, &apiRouter)
	return cachingRouter, nil
}

// startIPFSNode start the node
func (n *Node) startIPFSNode(repoPath string, config *ipfscore.BuildCfg) (*ipfscore.IpfsNode, commands.Context, error) {
	cctx, cancel := context.WithCancel(context.Background())
	n.cancel = cancel

	ctx := commands.Context{}
	nd, err := ipfscore.NewNode(cctx, config)
	if err != nil {
		return nil, ctx, err
	}

	ctx.Online = true
	ctx.ConfigRoot = repoPath
	ctx.LoadConfig = func(_ string) (*ipfsconfig.Config, error) {
		return fsrepo.ConfigAt(repoPath)
	}
	ctx.ConstructNode = func() (*ipfscore.IpfsNode, error) {
		return nd, nil
	}
	return nd, ctx, nil
}

// Start start openbazaard (OpenBazaar daemon)
func (n *Node) Start() error {
	nd, ctx, err := n.startIPFSNode(n.config.RepoPath, n.ipfsConfig)
	if err != nil {
		return err
	}

	// Extract the DHT from the tiered routing so it will be more accessible later
	tiered, ok := nd.Routing.(routinghelpers.Tiered)
	if !ok {
		return errors.New("IPFS routing is not a type routinghelpers.Tiered")
	}
	var dhtRouting *dht.IpfsDHT
	for _, router := range tiered.Routers {
		if r, ok := router.(*ipfs.CachingRouter); ok {
			dhtRouting, err = r.DHT()
			if err != nil {
				return err
			}
		}
	}
	if dhtRouting == nil {
		return errors.New("IPFS DHT routing is not configured")
	}

	n.OpenBazaarNode.IpfsNode = nd
	n.OpenBazaarNode.DHT = dhtRouting

	// Get current directory root hash
	ipnskey := namesys.IpnsDsKey(nd.Identity)
	ival, hasherr := nd.Repo.Datastore().Get(ipnskey)
	if hasherr != nil {
		log.Error("get ipns key:", hasherr)
	}
	ourIpnsRecord := new(ipnspb.IpnsEntry)
	err = proto.Unmarshal(ival, ourIpnsRecord)
	if err != nil {
		log.Error("unmarshal record value", err)
	}
	n.OpenBazaarNode.RootHash = string(ourIpnsRecord.Value)

	configFile, err := ioutil.ReadFile(path.Join(n.OpenBazaarNode.RepoPath, "config"))
	if err != nil {
		return err
	}
	republishInterval, err := apiSchema.GetRepublishInterval(configFile)
	if err != nil {
		return err
	}

	// Offline messaging storage
	n.OpenBazaarNode.MessageStorage = selfhosted.NewSelfHostedStorage(n.OpenBazaarNode.RepoPath, n.OpenBazaarNode.IpfsNode, n.OpenBazaarNode.PushNodes, n.OpenBazaarNode.SendStore)

	// Build pubsub
	publisher := ipfs.NewPubsubPublisher(context.Background(), nd.PeerHost, nd.Routing, nd.Repo.Datastore(), nd.PubSub)
	subscriber := ipfs.NewPubsubSubscriber(context.Background(), nd.PeerHost, nd.Routing, nd.Repo.Datastore(), nd.PubSub)
	ps := ipfs.Pubsub{Publisher: publisher, Subscriber: subscriber}
	n.OpenBazaarNode.Pubsub = ps

	// Start gateway
	// Create authentication cookie
	var authCookie http.Cookie
	authCookie.Name = "OpenBazaar_Auth_Cookie"

	if n.config.AuthenticationToken != "" {
		authCookie.Value = n.config.AuthenticationToken
		n.apiConfig.Authenticated = true
	}
	gateway, err := newHTTPGateway(core.Node, ctx, authCookie, *n.apiConfig)
	if err != nil {
		return err
	}
	go gateway.Serve()

	go func() {
		resyncManager := resync.NewResyncManager(n.OpenBazaarNode.Datastore.Sales(), n.OpenBazaarNode.Multiwallet)
		if !n.config.DisableWallet {
			if resyncManager == nil {
				core.Node.WaitForMessageRetrieverCompletion()
			}
			TL := lis.NewTransactionListener(n.OpenBazaarNode.Multiwallet, core.Node.Datastore, core.Node.Broadcast)
			for ct, wal := range n.OpenBazaarNode.Multiwallet {
				WL := lis.NewWalletListener(core.Node.Datastore, core.Node.Broadcast, ct)
				wal.AddTransactionListener(WL.OnTransactionReceived)
				wal.AddTransactionListener(TL.OnTransactionReceived)
			}
			su := wallet.NewStatusUpdater(n.OpenBazaarNode.Multiwallet, n.OpenBazaarNode.Broadcast, n.OpenBazaarNode.IpfsNode.Context())
			go su.Start()
			go n.OpenBazaarNode.Multiwallet.Start()
			if resyncManager != nil {
				go resyncManager.Start()
				go func() {
					core.Node.WaitForMessageRetrieverCompletion()
					resyncManager.CheckUnfunded()
				}()
			}
		}
		n.OpenBazaarNode.Service = service.New(n.OpenBazaarNode, n.OpenBazaarNode.Datastore)
		n.OpenBazaarNode.Service.WaitForReady()
		MR := ret.NewMessageRetriever(ret.MRConfig{
			Db:        n.OpenBazaarNode.Datastore,
			IPFSNode:  n.OpenBazaarNode.IpfsNode,
			DHT:       n.OpenBazaarNode.DHT,
			BanManger: n.OpenBazaarNode.BanManager,
			Service:   core.Node.Service,
			PrefixLen: 14,
			PushNodes: core.Node.PushNodes,
			Dialer:    nil,
			SendAck:   n.OpenBazaarNode.SendOfflineAck,
			SendError: n.OpenBazaarNode.SendError,
		})
		go MR.Run()
		n.OpenBazaarNode.MessageRetriever = MR
		PR := rep.NewPointerRepublisher(n.OpenBazaarNode.DHT, n.OpenBazaarNode.Datastore, n.OpenBazaarNode.PushNodes, n.OpenBazaarNode.IsModerator)
		go PR.Run()
		n.OpenBazaarNode.PointerRepublisher = PR
		MR.Wait()

		core.PublishLock.Unlock()
		core.Node.UpdateFollow()
		if !core.InitalPublishComplete {
			core.Node.SeedNode()
		}
		core.Node.SetUpRepublisher(republishInterval)
	}()

	return nil
}

// Stop stop openbazaard
func (n *Node) Stop() error {
	core.OfflineMessageWaitGroup.Wait()
	core.Node.Datastore.Close()
	repoLockFile := filepath.Join(core.Node.RepoPath, fsrepo.LockFile)
	os.Remove(repoLockFile)
	core.Node.Multiwallet.Close()
	core.Node.IpfsNode.Close()
	return nil
}

// initializeRepo create the database
func initializeRepo(dataDir, password, mnemonic string, testnet bool, creationDate time.Time, coinType wi.CoinType) (*db.SQLiteDatastore, error) {
	// Database
	sqliteDB, err := db.Create(dataDir, password, testnet, util.ExtendCoinType(coinType))
	if err != nil {
		return sqliteDB, err
	}

	// Initialize the IPFS repo if it does not already exist
	err = repo.DoInit(dataDir, 4096, testnet, password, mnemonic, creationDate, sqliteDB.Config().Init)
	if err != nil {
		return sqliteDB, err
	}
	return sqliteDB, nil
}

// Collects options, creates listener, prints status message and starts serving requests
func newHTTPGateway(node *core.OpenBazaarNode, ctx commands.Context, authCookie http.Cookie, config apiSchema.APIConfig) (*api.Gateway, error) {
	// Get API configuration
	cfg, err := ctx.GetConfig()
	if err != nil {
		return nil, err
	}

	// Create a network listener
	gatewayMaddr, err := ma.NewMultiaddr(cfg.Addresses.Gateway[0])
	if err != nil {
		return nil, fmt.Errorf("newHTTPGateway: invalid gateway address: %q (err: %s)", cfg.Addresses.Gateway, err)
	}
	var gwLis manet.Listener
	gwLis, err = manet.Listen(gatewayMaddr)
	if err != nil {
		return nil, fmt.Errorf("newHTTPGateway: manet.Listen(%s) failed: %s", gatewayMaddr, err)
	}

	// Setup an options slice
	var opts = []corehttp.ServeOption{
		corehttp.MetricsCollectionOption("gateway"),
		corehttp.CommandsROOption(ctx),
		corehttp.VersionOption(),
		corehttp.IPNSHostnameOption(),
		corehttp.GatewayOption(cfg.Gateway.Writable, "/ipfs", "/ipns"),
	}

	if len(cfg.Gateway.RootRedirect) > 0 {
		opts = append(opts, corehttp.RedirectOption("", cfg.Gateway.RootRedirect))
	}

	// Create and return an API gateway
	return api.NewGateway(node, authCookie, manet.NetListener(gwLis), config, logger, opts...)
}
