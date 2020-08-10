package cmd

import (
	"fmt"
	"os"

	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/phoreproject/multiwallet/util"
	obnet "github.com/phoreproject/pm-go/net"
	"github.com/phoreproject/pm-go/repo"
	"github.com/phoreproject/pm-go/repo/db"
)

type Status struct {
	DataDir string `short:"d" long:"datadir" description:"specify the data directory to be used"`
	Testnet bool   `short:"t" long:"testnet" description:"use the test network"`
}

func (x *Status) Execute(args []string) error {
	// Set repo path
	repoPath, err := repo.GetRepoPath(x.Testnet)
	if err != nil {
		return err
	}
	if x.DataDir != "" {
		repoPath = x.DataDir
	}
	torAvailable := false
	_, err = obnet.GetTorControlPort()
	if err == nil {
		torAvailable = true
	}
	if fsrepo.IsInitialized(repoPath) {
		sqliteDB, err := db.Create(repoPath, "", x.Testnet, util.CoinTypePhore)
		if err != nil {
			os.Exit(1)
		}
		defer sqliteDB.Close()
		if sqliteDB.Config().IsEncrypted() {
			if !torAvailable {
				fmt.Println("Initialized - Encrypted")
				os.Exit(30)
			} else {
				fmt.Println("Initialized - Encrypted")
				fmt.Println("Tor Available")
				os.Exit(31)
			}
		} else {
			_, isEncrypted, err := sqliteDB.Config().GetMnemonic()
			if err != nil {
				os.Exit(1)
			}

			if !torAvailable {
				fmt.Println("Initialized - Not Encrypted")
				if isEncrypted {
					fmt.Println("Seed Words Encrypted")
					os.Exit(25)
				} else {
					os.Exit(20)
				}
			} else {
				fmt.Println("Initialized - Not Encrypted")
				fmt.Println("Tor Available")
				if isEncrypted {
					fmt.Println("Seed Words Encrypted")
					os.Exit(26)
				} else {
					os.Exit(21)
				}
			}
		}
	} else {
		if !torAvailable {
			fmt.Println("Not initialized")
			os.Exit(10)
		} else {
			fmt.Println("Not initialized")
			fmt.Println("Tor Available")
			os.Exit(11)
		}
	}
	return nil
}
