#!/bin/bash

cd $(dirname $0)/..

githash=`git log --pretty=format:%h,%ad --name-only --date=short . | head -n 1`
ver=`cat ver`

echo "-X main.version=$ver -X main.githash=$githash"
