#!/bin/sh

#
# Ubuntu configuration file for a server dedicated
# to a NATS server process (gnatsd or nats-streaming-server)
#
# An experimental WIP
#
sysctl -w fs.file-max=13189848

# Not all systems have netfilter installed.
#sudo sysctl -w net.netfilter.nf_conntrack_tcp_timeout_time_wait=5
#sudo sysctl -w net.netfilter.nf_conntrack_max=196608

# we need all the ports we can get.
sysctl -w net.ipv4.ip_local_port_range="1025 65535"

# Keep the default sizes lower to scale # of connections.
# For throughput on a smaller # of connections, increase it
sysctl -w net.ipv4.tcp_rmem="4096 4096 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 4096 16777216"
sysctl -w net.ipv4.tcp_max_syn_backlog=32768

# Increase backlogs for processing.  Usually gnatsd
# won't fall behind (unless swapping out), but be
# safe.
sysctl -w net.core.somaxconn=32768
sysctl -w net.core.netdev_max_backlog=32678
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216

# VM is bad for us, reduce swapping (TODO: Go down to 0?)
sysctl -w vm.swappiness=1
sysctl -w vm.dirty_background_ratio=10
sysctl -w vm.dirty_ratio=15

# increase max fds, more important for streaming, but just be greedy.
# if we had other processes here, be a bit more conservative.
u=`id -u -n`
echo "$u hard nofile 1024000" > max_fds.conf
echo "$u soft nofile 1024000" >> max_fds.conf
mv max_fds.conf /etc/security/limits.d

# echo $1	monitorhost  >> /etc/hosts

