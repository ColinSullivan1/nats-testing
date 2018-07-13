#!/bin/sh

if [ "$#" -ne 2 ]; then
    echo "Usage: $0 <tag> <hostname>"
    exit
fi

subj="/C=US/ST=CO/L=Denver/O=Synadia/OU=NATS/CN=$2/emailAddress=info@nats.io"
echo $subj

# Generate self signed root CA cert
openssl req -nodes -x509 -newkey rsa:2048 -keyout ca-key.pem -out ca.pem -subj $subj

# Generate $1-server cert to be signed
openssl req -nodes -newkey rsa:2048 -keyout $1-key.pem -out $1-server.csr -subj $subj

# Sign the $1-server cert
openssl x509 -req -in $1-server.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out $1-server.crt

# Create $1-server PEM file
cat $1-key.pem $1-server.crt > $1-cert.pem

# Clean up
rm *.srl *.csr *.crt 2>/dev/null
