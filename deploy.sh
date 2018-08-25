#!/bin/bash

if [ ! -z "$TRAVIS_TAG" ] &&
    [ "$TRAVIS_PULL_REQUEST" == "false" ]; then
  echo "This will deploy!"

  ./build.sh

  cd dist

  # Copy libwinpthread-1.dll (user must rename the dll for their system to libwinpthread-1.dll)
  cp ../.travis/win32/libwinpthread-1.dll libwinpthread-1.win32.dll
  cp ../.travis/win64/libwinpthread-1.dll libwinpthread-1.win64.dll

  # Upload to GitHub Release page
  ghr --username phoreproject -t $GITHUB_TOKEN --replace --prerelease --debug $TRAVIS_TAG .
else
  echo "This will not deploy!"
fi
