#!/bin/sh

#
# Convenience script should be copied to the test machine
#

sudo apt update
sudo apt install git
sudo apt install collectd
sudo cp collectd.conf /etc/collectd/collectd.conf
sudo service collectd restart

wget https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz
sudo tar -zxvf go1.7.1.linux-amd64.tar.gz -C /usr/local/

mkdir go 2>/dev/null
export GOPATH=~/go
go get github.com/nats-io/gnatsd
go get github.com/prometheus/collectd_exporter

