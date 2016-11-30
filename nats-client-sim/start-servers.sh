#!/bin/bash

count=10
url=localhost

while getopts c:a:h: option
do
        case "${option}"
        in
                c) count=${OPTARG};;
                a) url=${OPTARG};;
        esac
done

echo "Starting $count servers listening on $url."

i=0
cd server
while [  $i -lt $count ]; do
    port=$(( 4200 + $i ))
    cmd="$GOPATH/bin/gnatsd -config gnatsd.conf -a $url -p $port -l s$i.log"
    echo "$cmd"
    $cmd &
    let i=i+1 
done

cd ..


