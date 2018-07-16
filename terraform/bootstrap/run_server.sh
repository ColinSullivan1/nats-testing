#!/bin/sh

. ./setenv.sh

echo "Starting NATS server process."
gnatsd -version
gnatsd -config gnatsd.conf 2>server.err 1>server.out &
echo "Started NATS server."

# This sleep is a hack to allow terraform to start the NATS server
# in a background process.
sleep 2 
