package schema

import "errors"

const (
	// SQL Statements
	PragmaUserVersionSQL                    = "pragma user_version = 0;"
	CreateTableConfigSQL                    = "create table config (key text primary key not null, value blob);"
	CreateTableFollowersSQL                 = "create table followers (peerID text primary key not null, proof blob);"
	CreateTableFollowingSQL                 = "create table following (peerID text primary key not null);"
	CreateTableOfflineMessagesSQL           = "create table offlinemessages (url text primary key not null, timestamp integer, message blob);"
	CreateTablePointersSQL                  = "create table pointers (pointerID text primary key not null, key text, address text, cancelID text, purpose integer, timestamp integer);"
	CreateTableKeysSQL                      = "create table keys (scriptAddress text primary key not null, purpose integer, keyIndex integer, used integer, key text, coin text);"
	CreateIndexKeysSQL                      = "create index index_keys on keys (coin);"
	CreateTableUnspentTransactionOutputsSQL = "create table utxos (outpoint text primary key not null, value text, height integer, scriptPubKey text, watchOnly integer, coin text);"
	CreateIndexUnspentTransactionOutputsSQL = "create index index_utxos on utxos (coin);"
	CreateTableSpentTransactionOutputsSQL   = "create table stxos (outpoint text primary key not null, value text, height integer, scriptPubKey text, watchOnly integer, spendHeight integer, spendTxid text, coin text);"
	CreateIndexSpentTransactionOutputsSQL   = "create index index_stxos on stxos (coin);"
	CreateTableTransactionsSQL              = "create table txns (txid text primary key not null, value text, height integer, timestamp integer, watchOnly integer, tx blob, coin text);"
	CreateIndexTransactionsSQL              = "create index index_txns on txns (coin);"
	CreateTableTransactionMetadataSQL       = "create table txmetadata (txid text primary key not null, address text, memo text, orderID text, thumbnail text, canBumpFee integer);"
	CreateTableInventorySQL                 = "create table inventory (invID text primary key not null, slug text, variantIndex integer, count integer);"
	CreateIndexInventorySQL                 = "create index index_inventory on inventory (slug);"
	CreateTablePurchasesSQL                 = "create table purchases (orderID text primary key not null, contract blob, state integer, read integer, timestamp integer, total integer, thumbnail text, vendorID text, vendorHandle text, title text, shippingName text, shippingAddress text, paymentAddr text, funded integer, transactions blob, lastDisputeTimeoutNotifiedAt integer not null default 0, lastDisputeExpiryNotifiedAt integer not null default 0, disputedAt integer not null default 0, coinType not null default '', paymentCoin not null default '');"
	CreateIndexPurchasesSQL                 = "create index index_purchases on purchases (paymentAddr, timestamp);"
	CreateTableSalesSQL                     = "create table sales (orderID text primary key not null, contract blob, state integer, read integer, timestamp integer, total integer, thumbnail text, buyerID text, buyerHandle text, title text, shippingName text, shippingAddress text, paymentAddr text, funded integer, transactions blob, lastDisputeTimeoutNotifiedAt integer not null default 0, coinType not null default '', paymentCoin not null default '');"
	CreateIndexSalesSQL                     = "create index index_sales on sales (paymentAddr, timestamp);"
	CreatedTableWatchedScriptsSQL           = "create table watchedscripts (scriptPubKey text primary key not null, coin text);"
	CreateIndexWatchedScriptsSQL            = "create index index_watchscripts on watchedscripts (coin);"
	CreateTableDisputedCasesSQL             = "create table cases (caseID text primary key not null, buyerContract blob, vendorContract blob, buyerValidationErrors blob, vendorValidationErrors blob, buyerPayoutAddress text, vendorPayoutAddress text, buyerOutpoints blob, vendorOutpoints blob, state integer, read integer, timestamp integer, buyerOpened integer, claim text, disputeResolution blob, lastDisputeExpiryNotifiedAt integer not null default 0, coinType not null default '', paymentCoin not null default '');"
	CreateIndexDisputedCasesSQL             = "create index index_cases on cases (timestamp);"
	CreateTableChatSQL                      = "create table chat (messageID text primary key not null, peerID text, subject text, message text, read integer, timestamp integer, outgoing integer);"
	CreateIndexChatSQL                      = "create index index_chat on chat (peerID, subject, read, timestamp);"
	CreateTableNotificationsSQL             = "create table notifications (notifID text primary key not null, serializedNotification blob, type text, timestamp integer, read integer);"
	CreateIndexNotificationsSQL             = "create index index_notifications on notifications (read, type, timestamp);"
	CreateTableCouponsSQL                   = "create table coupons (slug text, code text, hash text);"
	CreateIndexCouponsSQL                   = "create index index_coupons on coupons (slug);"
	CreateTableModeratedStoresSQL           = "create table moderatedstores (peerID text primary key not null);"
	CreateMessagesSQL                       = "create table messages (messageID text primary key not null, orderID text, message_type integer, message blob, peerID text, url text, acknowledged bool, tries integer, created_at integer, updated_at integer, err string, received_at integer, pubkey blob);"
	CreateIndexMessagesSQLMessageID         = "create index index_messages_messageID on messages (messageID);"
	CreateIndexMessagesSQLOrderIDMType      = "create index index_messages_orderIDmType on messages (orderID, message_type);"
	CreateIndexMessagesSQLPeerIDMType       = "create index index_messages_peerIDmType on messages (peerID, message_type);"
	// End SQL Statements

	// Configuration defaults
	EthereumRegistryAddressMainnet = "0x5c69ccf91eab4ef80d9929b3c1b4d5bc03eb0981"
	EthereumRegistryAddressRinkeby = "0x5cEF053c7b383f430FC4F4e1ea2F7D31d8e2D16C"
	EthereumRegistryAddressRopsten = "0x403d907982474cdd51687b09a8968346159378f3"

	DataPushNodeOne   = "QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh"
	DataPushNodeTwo   = "Qma2LRYB4xLaoxsMCL2kb93WKCW4EotUMhgvQUSqE6tCka"
	DataPushNodeThree = "QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf"
	DataPushNodeFour  = "QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv"

	BootstrapNodeDefaultOne               = "/ip4/54.227.172.110/tcp/5001/ipfs/QmWbi8z4uPkEdrWHtgxCkQGE5vxJnrStXAeEQnupmQnKRh"
	BootstrapNodeDefaultTwo               = "/ip4/144.202.25.235/tcp/5001/ipfs/Qma2LRYB4xLaoxsMCL2kb93WKCW4EotUMhgvQUSqE6tCka"
	BootstrapNodeDefaultThree             = "/ip4/54.175.193.226/tcp/5001/ipfs/QmZLs6zVpVtkoR8oYyAbCxujvC6weU5CgUPTYx8zKMAtTf"
	BootstrapNodeDefaultFour              = "/ip4/34.239.133.237/tcp/5001/ipfs/QmNSnS2K3TkSQjxJhaRBSZxotUQp1yxLss4zKDVbhRc9nv"
	BootstrapNodeDefault_LeMarcheSerpette = "/ip4/159.203.115.78/tcp/5001/ipfs/QmPJuP4Myo8pGL1k56b85Q4rpaoSnmn5L3wLjYHTzbBrk1"
	BootstrapNodeDefault_BrixtonVillage   = "/ip4/104.131.19.44/tcp/5001/ipfs/QmRvbZttqh6CPFiMKWa1jPfRR9JxagYRv4wsvMAG4ADUTj"

	IPFSCachingRouterDefaultURI = "https://routing.api.phore.io"

	// Testnet defaults
	DataPushNodeTestnetOne  = "QmYJ42wLrkZE119DoDEd4TASuWFTEdwY9A9MpYS2okvfvJ"
	BootstrapNodeTestnetOne = "/ip4/3.89.75.185/tcp/5001/ipfs/QmYJ42wLrkZE119DoDEd4TASuWFTEdwY9A9MpYS2okvfvJ"
	// End Configuration defaults
)

var (
	// Errors
	ErrorEmptyMnemonic = errors.New("mnemonic string must not be empty")
	// End Errors
)

var (
	DataPushNodes = []string{DataPushNodeOne, DataPushNodeTwo, DataPushNodeThree, DataPushNodeFour}

	BootstrapAddressesDefault = []string{
		BootstrapNodeDefaultOne,
		BootstrapNodeDefaultTwo,
		BootstrapNodeDefaultThree,
		BootstrapNodeDefaultFour,
		BootstrapNodeDefault_LeMarcheSerpette,
		BootstrapNodeDefault_BrixtonVillage,
	}
	BootstrapAddressesTestnet = []string{
		BootstrapNodeTestnetOne,
	}
)

func EthereumDefaultOptions() map[string]interface{} {
	return map[string]interface{}{
		"RegistryAddress":        EthereumRegistryAddressMainnet,
		"RinkebyRegistryAddress": EthereumRegistryAddressRinkeby,
		"RopstenRegistryAddress": EthereumRegistryAddressRopsten,
	}
}

const (
	WalletTypeAPI = "API"
	WalletTypeSPV = "SPV"
)

const (
	CoinAPIOpenBazaarPHR = "https://phr.blockbook.api.phore.io/api"
	CoinAPIOpenBazaarBTC = "https://btc.blockbook.api.phore.io/api"
	CoinAPIOpenBazaarETH = "https://mainnet.infura.io"

	CoinAPIOpenBazaarTPHR = "https://tphr.blockbook.api.phore.io/api"
	CoinAPIOpenBazaarTBTC = "https://tbtc.blockbook.api.phore.io/api"
	CoinAPIOpenBazaarTETH = "https://rinkeby.infura.io"
)

var (
	CoinPoolPHR = []string{CoinAPIOpenBazaarPHR}
	CoinPoolBTC = []string{CoinAPIOpenBazaarBTC}
	CoinPoolETH = []string{CoinAPIOpenBazaarETH}

	CoinPoolTPHR = []string{CoinAPIOpenBazaarTPHR}
	CoinPoolTBTC = []string{CoinAPIOpenBazaarTBTC}
	CoinPoolTETH = []string{CoinAPIOpenBazaarTETH}
)
