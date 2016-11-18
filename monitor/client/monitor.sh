#!/bin/sh

kill `ps -a | grep collectd_export | cut -c 1-6`
nohup go/bin/collectd_exporter -collectd.listen-address=":25826" &
