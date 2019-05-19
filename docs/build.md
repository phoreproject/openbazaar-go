### Build on your local platform
You can find step by step tutorial here:

For Linux: [here](https://github.com/phoreproject/openbazaar-go/blob/master/docs/install-linux.md)

For MacOS: [here](https://github.com/phoreproject/openbazaar-go/blob/master/docs/install-osx.md)

For Windows: [here](https://github.com/phoreproject/openbazaar-go/blob/master/docs/install-windows.md)

For Raspberry: [here](https://github.com/phoreproject/openbazaar-go/blob/master/docs/install-pi3.md)


### Multiplatform build
Multiplatform build use docker and go. 
+ To install go use tutorial from links above.
+ Easy way to install docker is presented on official docker documentation site [here](https://docs.docker.com/install/)

#### Next steps
1. Start docker daemon
2. go to openbazaar-go directory `cd $GOPATH/src/github.com/phoreproject/openbazaar-go`
3. start 'build.sh' script `./build.sh`

Compilation can take long time.
Script use [xgo](https://github.com/karalabe/xgo) to build executables. 