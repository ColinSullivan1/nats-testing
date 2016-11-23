# Overview

This is work in progress.

The intention here is to test scalability and monitor NATS and NATS streaming during stress tests.

# Installation

This assumes a clean ubuntu installation.

```
cd $HOME
sudo apt update
sudo apt install git
mkdir -p gopath/src/github.com/ColinSullivan1
git clone https://github.com/ColinSullivan1/nats-testing.git gopath/src/github.com/ColinSullivan1
cd gopath/src/github.com/ColinSullivan1/scripts
sh install_deps.sh
sudo sh setup_ub.sh

# now start monitoring
cd ../monitor/client
sh monitor.sh
```

Log out and back in for your shell settings (namely max fds) to take effect.  On the server you wish to use for monitoring (I use my laptop), install prometheus and grafana.  Alter `server/prometheus.yml` to use the hostname of your test machine.

```
sh start-prom-graf.sh
```

# Running tests

Start servers on the test machine, and from another machine, run the go applications, `nats-client-sim` or `stan-client-sim` with profiles to simulate the load you are testing.  (TODO:  Directions)

# Monitoring
To monitor what is going on a grafana with prometheus and collectd is used.

## Client (Test machine)
 The installation scripts install collectd and a prometheus exporter on the test machine.  

## Server (Monitoring machine)
 For the monitoring machine, you'll have to install prometheus and grafana yourself.

```
http://<monitor host>:3000
```

The first time, you'll want to install and use the dashboard `monitor/server/graphana-dash.json` or create your own from the prometheus `collectd` data.


# Troubleshooting

## Checking collectd exporter on the test machine
```
http://<test host>:9103/metrics
```
Ensure the exporter is reachable and generating data.

## Checking prometheus
```
http://<monitoring host>:9090/targets
```
Ensure prometheus is reachable, and the target collector is `UP`.

# TODO

[ ] Streamline Installation (Too many steps)
[ ] Doc client simulators
