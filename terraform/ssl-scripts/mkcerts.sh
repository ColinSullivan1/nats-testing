#!/bin/sh

# Install using instructions here:  https://github.com/FiloSottile/mkcert

nodename="unset"
mkdir ca_cert
export CAROOT=`pwd`/ca_cert
mkcert -install

function makeMachineCert() {
  echo making certs for "$nodename"
  mkdir -p $nodename 
  cd $nodename
  mkcert $nodename
  ls
  echo mv $nodename*-key.pem ../../$nodename/$nodename-key.pem
  mv $nodename*-key.pem ../../$nodename/$nodename-key.pem
  echo mv $nodename*.pem ../../$nodename/$nodename.pem
  mv $nodename*.pem ../../$nodename/$nodename.pem
  cp -f ../ca_cert/rootCA.pem ../../$nodename/ca.pem
  cd ..
}

nodename="client"
makeMachineCert

nodename="servera"
makeMachineCert

nodename="serverb"
makeMachineCert
