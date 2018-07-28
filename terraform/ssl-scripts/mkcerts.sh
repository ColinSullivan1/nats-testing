#!/bin/sh

# Install using instructions here:  https://github.com/FiloSottile/mkcert

nodename="unset"

function makeMachineCert() {
  echo making certs for "$nodename"
  mkdir -p $nodename 
  cd $nodename
  export CAROOT=`pwd`
  mkcert $nodename.latency.nats.com latency.nats.com '*.latency.nats.com' $nodename 127.0.0.1 localhost
  ls
  echo mv $nodename.latency.nats.*-key.pem ../../$nodename/$nodename-key.pem
  mv $nodename.latency.nats.*-key.pem ../../$nodename/$nodename-key.pem
  echo mv $nodename.latency.nats.*.pem ../../$nodename/$nodename.pem
  mv $nodename.latency.nats.*.pem ../../$nodename/$nodename.pem
  mv rootCA.pem ../../$nodename/ca.pem
  mv rootCA-key.pem rootCA-key.pem.bak
  cd ..
}

nodename="client"
makeMachineCert

nodename="servera"
makeMachineCert

nodename="serverb"
makeMachineCert
