WINDOWS INSTALL NOTES
====================

### Install dependencies

You need to have Go, Git, and GCC installed to compile and run the OpenBazaar-Go daemon.

- **Go v1.11**
    + https://golang.org/dl
    + Note: OpenBazaar has not been tested on v1.12 and may cause problems
- **Git**
    + https://git-scm.com/download/win
- **TDM-GCC/MinGW-w64**
    + http://tdm-gcc.tdragon.net/ 
    + A standard installation should automatically add the correct `PATH`, but if it doesn't:
        * Open the command prompt and type: `setx PATH "%PATH;C:\TDM-GCC-64\bin"`

### Setup Go

Create a directory to store all your Go projects (e.g. `C:\goprojects`):

- Set your GOPATH
    + Open the command prompt and type: `setx GOPATH "C:\goprojects"`

### Install openbazaar-go

- Install `openbazaar-go`:
    + Open the command prompt and run: `go get github.com/phoreproejct/openbazaar-go`. This will use git to checkout the source code into `%GOPATH%\src\github.com\phoreproject\openbazaar-go`.
- Checkout an Phore Marketplace release:
    + Run `git checkout v0.13.5` to checkout a release version.
    + Note: `go get` leaves the repo pointing at `master` which is a branch used for active development. Running Phore Marketplace from `master` is NOT recommended. Check the [release versions](https://github.com/phoreproject/openbazaar-go/releases) for the available versions that you use in checkout.
- To compile and run `openbazaar-go`:
    + Open the command prompt and navigate the source directory: `cd %GOPATH%\src\github.com\phoreproject\openbazaar-go` 
    + Type: `go run openbazaard.go start`
