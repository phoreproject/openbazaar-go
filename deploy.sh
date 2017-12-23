#!/bin/bash

if [ ! -z "$TRAVIS_TAG" ] &&
    [ "$TRAVIS_PULL_REQUEST" == "false" ]; then
  echo "This will deploy!"

  # Cross-compile for all platforms
  export CGO_ENABLED=1
  docker pull karalabe/xgo-latest
  go get github.com/karalabe/xgo
  mkdir dist/ && cd dist/
  xgo --targets=windows/386,windows/amd64,darwin/amd64,linux/386,linux/amd64,linux/arm ../
  chmod +x *

  # Copy libwinpthread-1.dll (user must rename the dll for their system to libwinpthread-1.dll)
  cp ../.travis/win32/libwinpthread-1.dll libwinpthread-1.win32.dll
  cp ../.travis/win64/libwinpthread-1.dll libwinpthread-1.win64.dll

  # Upload to GitHub Release page
  ghr --username meyer9 -t $GITHUB_TOKEN --replace --prerelease --debug $TRAVIS_TAG .
else
  echo "This will not deploy!"
fi
