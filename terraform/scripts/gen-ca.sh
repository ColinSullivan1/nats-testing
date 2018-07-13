#!/bin/sh

# Generate self signed root CA cert
openssl req -nodes -x509 -newkey rsa:2048 -keyout ca-key.pem -out ca.pem -subj "/C=US/ST=CO/L=Denver/O=Synadia/OU=NATS/CN=*/emailAddress=info@nats.io"

# Clean up
rm *.srl *.csr *.crt 2>/dev/null
