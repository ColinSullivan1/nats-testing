#!/bin/bash

count=10
config=configs/pub25_sub25_2x2.json
url=localhost

while getopts c:u:g: option
do
        case "${option}"
        in
                c) count=${OPTARG};;
                u) url=${OPTARG};;
                g) config=${OPTARG};;
        esac
done

echo "Starting $count client simulator(s) to $url with config $config."

i=0
while [  $i -lt $count ]; do
    port=$(( 4200 + $i ))
    cmd="./nats-client-sim -url nats://$url:$port -config $config -report 5"
    echo "$cmd"
    $cmd >cli.$i.log 2>&1 &
    let i=i+1 
done

