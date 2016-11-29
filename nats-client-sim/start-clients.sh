#!/bin/bash


count=10
config=config.json
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
    cmd="./nats-client-sim -url \"nats://$url:400$i\" -config $config"
    echo "$cmd"
    $cmd &
    let i=i+1 
done

