# Test Results

This test shows a trend and approximates where higher bcrypt costs (# of rounds) have an impact on performance with large numbers of connections being authenticated. Results are from a strained macbook pro. 

* `cost_0` is no hashing, straight password compare.
* `cost_16` is using a bcrypt hash with 16 rounds.

The NATS server password utility default is 11 at the time of this test.

Command used to generate password:
` go run mkpasswd.go -c <count> -p`, password is `password`

**Total Connect time** is the total amount of time taken for all connections to connect somewhat simultaneously to the server.  There is a random wait between 0 and 2 seconds to prevent I/O errors due to resource contraints, so an optimal result should be around 2 seconds when resources are available.

**Total Reconnect time** is the amount of time it takes for all connections to reconnect to the server.


Be sure system resources are available.  e.g. `ulimit -n 1024`

## 50 Connections

```
Test cost_0, 50 connections.
Total Connect time:   1.928884994s
Total Reconnect time: 2.000224079s

Test cost_4, 50 connections.
Total Connect time:   1.914441538s
Total Reconnect time: 1.995116695s

Test cost_8, 50 connections.
Total Connect time:   1.937140415s
Total Reconnect time: 2.00428709s

Test cost_11, 50 connections.
Total Connect time:   2.169499743s
Total Reconnect time: 1.999557917s

Test cost_16, 50 connections.
Total Connect time:   40.945506102s
Total Reconnect time: 41.842097712s
```
## 100 Connections
```
Test cost_0, 100 connections.
Total Connect time:   1.992474181s
Total Reconnect time: 1.999985661s

Test cost_4, 100 connections.
Total Connect time:   1.964213315s
Total Reconnect time: 1.997066938s

Test cost_8, 100 connections.
Total Connect time:   1.993879046s
Total Reconnect time: 1.997553861s

Test cost_11, 100 connections.
Total Connect time:   2.479678586s
Total Reconnect time: 2.732054677s

Test cost_16, 100 connections.
Total Connect time:   1m20.672663307s
Total Reconnect time: 1m20.056788706s
```
## 150 Connections
```
Test cost_0, 150 connections.
Total Connect time:   2.002085473s
Total Reconnect time: 2.013116551s

Test cost_4, 150 connections.
Total Connect time:   1.990636397s
Total Reconnect time: 1.992973878s

Test cost_8, 150 connections.
Total Connect time:   2.017742115s
Total Reconnect time: 1.997850965s

Test cost_11, 150 connections.
Total Connect time:   3.671307203s
Total Reconnect time: 3.777471022s

Test cost_16, 150 connections.
Total Connect time:   1m55.908984636s
Total Reconnect time: 1m59.137358796s
```
