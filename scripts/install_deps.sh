#!/bin/sh

#
# Convenience script should be copied to the test machine
#
sudo apt update
sudo apt install collectd
sudo cp collectd.conf /etc/collectd/collectd.conf
sudo service collectd restart

wget https://storage.googleapis.com/golang/go1.7.1.linux-amd64.tar.gz
tar -zxvf go1.7.1.linux-amd64.tar.gz -C $HOME

export PATH=$HOME/go/bin:$PATH
export GOPATH=$HOME/gopath

echo "Getting gnatsd"
go get github.com/nats-io/gnatsd

echo "Getting promethous exporter"
go get github.com/prometheus/collectd_exporter

