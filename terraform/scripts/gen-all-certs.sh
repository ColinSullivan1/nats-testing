#!/bin/sh

mkdir tmp 2>/dev/null
cd tmp
../gen-ca.sh

# generate server a
../gen-certs.sh srva nats.server.a

# generate server b
../gen-certs.sh srvb nats.server.b

# generate client certs
../gen-certs.sh lattest nats.client

# copy the certs
cp srva-cert.pem srva-key.pem ca.pem ../../servera
cp srvb-cert.pem srvb-key.pem ca.pem ../../serverb

cp lattest-cert.pem lattest-key.pem ../../client

cd ..
