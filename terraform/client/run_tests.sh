#!/bin/sh

. ./setenv.sh

dur="30s"

latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 256 -tr 1000 -tt $dur -hist lat_256b_1kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 256 -tr 10000 -tt $dur -hist lat_256b_10kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 256 -tr 100000 -tt $dur -hist lat_256b_100kmps

latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 1024 -tr 1000 -tt $dur -hist lat_1k_1kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 1024 -tr 10000 -tt $dur -hist lat_1k_10kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 1024 -tr 100000 -tt $dur -hist lat_1k_100kmps

latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 4096 -tr 1000 -tt $dur -hist lat_4k_1kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 4096 -tr 10000 -tt $dur -hist lat_4k_10kmps
latency-tests -sa nats://servera:4222 -sb nats://serverb:4222 -sz 4096 -tr 100000 -tt $dur -hist lat_4k_100kmps
