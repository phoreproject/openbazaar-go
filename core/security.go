package core



// Lock / Unlock wallet request
type ManageWalletRequest struct {
	WalletPassword string `json:"password"`
	UnlockTimestamp int   `json:"unlockTimestamp"`
}