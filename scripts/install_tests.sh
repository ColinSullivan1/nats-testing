#!/bin/sh

# Convenience script should be copied to the test machine
# wget https://raw.githubusercontent.com/ColinSullivan1/nats-testing/master/scripts/install_tests.sh

sudo apt install git
sudo apt install golang
mkdir go 2>/dev/null
export GOPATH=`pwd`/go
go get github.com/nats-io/gnatsd
go get github.com/ColinSullivan1/nats-testing/nats-client-sim
