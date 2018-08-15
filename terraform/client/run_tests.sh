#!/bin/sh

. ./setenv.sh

dur="5s"

#
# Run each test three times.  The best of three will be reported.
#
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 1000 -tt $dur -hist lat_256b_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 1000 -tt $dur -hist lat_256b_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 1000 -tt $dur -hist lat_256b_1kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 10000 -tt $dur -hist lat_256b_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 10000 -tt $dur -hist lat_256b_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 10000 -tt $dur -hist lat_256b_10kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 100000 -tt $dur -hist lat_256b_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 100000 -tt $dur -hist lat_256b_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 256 -tr 100000 -tt $dur -hist lat_256b_100kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 1000 -tt $dur -hist lat_1k_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 1000 -tt $dur -hist lat_1k_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 1000 -tt $dur -hist lat_1k_1kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 10000 -tt $dur -hist lat_1k_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 10000 -tt $dur -hist lat_1k_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 10000 -tt $dur -hist lat_1k_10kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 100000 -tt $dur -hist lat_1k_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 100000 -tt $dur -hist lat_1k_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 1024 -tr 100000 -tt $dur -hist lat_1k_100kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 1000 -tt $dur -hist lat_4k_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 1000 -tt $dur -hist lat_4k_1kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 1000 -tt $dur -hist lat_4k_1kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 10000 -tt $dur -hist lat_4k_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 10000 -tt $dur -hist lat_4k_10kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 10000 -tt $dur -hist lat_4k_10kmps

latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 100000 -tt $dur -hist lat_4k_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 100000 -tt $dur -hist lat_4k_100kmps
latency-tests -sa nats://luser:top_secret@servera:4222 -sb nats://luser:top_secret@serverb:4222 -sz 4096 -tr 100000 -tt $dur -hist lat_4k_100kmps
