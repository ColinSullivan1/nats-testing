# Test Results

This test shows a trend and approximates where higher bcrypt costs (# of rounds) have an impact on performance with large numbers of connections being authenticated. Results are from a strained macbook pro.

* `cost_0` is no hashing, straight password compare.
* `cost_16` is using a bcrypt hash with 16 rounds.

The NATS server password utility default is 11 at the time of this test.

Command used to generate password:
` go run mkpasswd.go -c <cost> -p`, password is `password`

**Time (seconds) ** is the total amount of time taken for all connections to connect somewhat simultaneously to the server.

Be sure system resources are available.  e.g. `ulimit -n 1024`

Not all tests were able to complete using a bcrypt cost of 20 due to connection errors.

##System Specs
```
Architecture:          x86_64
CPU op-mode(s):        32-bit, 64-bit
Byte Order:            Little Endian
CPU(s):                48
On-line CPU(s) list:   0-47
Thread(s) per core:    2
Core(s) per socket:    6
Socket(s):             4
NUMA node(s):          4
Vendor ID:             GenuineIntel
CPU family:            6
Model:                 63
Stepping:              2
CPU MHz:               2352.171
BogoMIPS:              5801.93
Virtualization:        VT-x
L1d cache:             32K
L1i cache:             32K
L2 cache:              256K
L3 cache:              30720K
NUMA node0 CPU(s):     0-5,24-29
NUMA node1 CPU(s):     6-11,30-35
NUMA node2 CPU(s):     12-17,36-41
NUMA node3 CPU(s):     18-23,42-47

total       used       free     shared    buffers     cached
Mem:     131924976    3679216  128245760       2104     364004    1176060
-/+ buffers/cache:    2139152  129785824
Swap:    134107132          0  134107132

```

## Raw Results
View results in [PFD](./ConnBcryptCostResults.pdf) or [Numbers](./ConnBcryptCostResults.numbers)

## Graphs
### 50 Connections
<img src="imgs/50Conns.png" alt="Connections Image" style="width: 600px;"/>
### 100 Connections
<img src="imgs/100Conns.png" alt="Connections Image" style="width: 600px;"/>
### 200 Connections
<img src="imgs/200Conns.png" alt="Connections Image" style="width: 600px;"/>
### 500 Connections
<img src="imgs/500Conns.png" alt="Connections Image" style="width: 600px;"/>
### 1000 Connections
<img src="imgs/1000Conns.png" alt="Connections Image" style="width: 600px;"/>
