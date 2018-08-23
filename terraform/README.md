
# Using Terraform to install and run the NATS cluster latency test

# Installation

First install the required tools, 'terraform'.

Terraform installation instructions can be found [here](https://www.terraform.io/intro/getting-started/install.html).

The first time you are running terraform here, you'll need initialize it
by running the `terraform init` command in the relevant provider directory.

## TLS

The provided SSL Certificates should work, but if you find a need to update them
you'll want to install [CFSSL](https://github.com/cloudflare/cfssl) and modify the
relevant [configuration files]("./ssl-scripts/config").  The certificates can be
generated to replace existing certificates through using the `ssl-scripts/gencerts.sh`.

## First time Usage

From each provider director, to provision the test instances, simply run the command:
`$ terraform apply`

## Test Setup

After setting up authorization and environment for the cloud you'd like to test with.  
See the README.MD files in the appropriate cloud vendor directory.

Supported Cloud Vendors:

* [GCP]("./gcp")
* [Packet]("./packet")

When launced, terraform will do the following:

* Create a server machine, `servera`
* Create a server machine, `serverb`
* Create a client machine, `client`
* Install scripts, SSL certificates, and relevant configuration files
* Route NATS servers to each other
* Launch the NATS servers
* Print relevant information about server locations, monitoring ect.

This will setup up a latency test that can be envisioned as a triangle.  

```text
Server A - - - - - Server B
    \                /
     \              /
      \            /
       \          /
        \        /
    Latency Test (client)
```

WARNING:  Unlike most perforamnce testing by default this testing is aimed to
simulate a secure production environment.  To that end, it includes:

 * NATS server clustering
 * End to End bidirectional TLS (client and routes connections)
 * Manual cipher suite overrides
 * Filtering of subjects between clustered servers
 * Authorization of clients and route connections
 * Publish and subscribe authorization for clients

Adjust configuration files as necessary to mirror your environment.

## Why the latency client on its own instance

Have the latency client on it's own instance is the best way to measure
end to end latency with respect to timing.  As we are in the low microsecond
range of measurements and measuring tail latency on higher end machines,
we need to very accurately measure time deltas.  This either requires:

a) sophisticated kernal time syncronization of a machine provisioned in the cloud

or

b) use the same kernel instance to measure time

In the interest of simplicity and brevity, we chose "b".

## Manually running the tests

To manually run tests, ssh to the client machine, and run the `run_tests.sh` script.






