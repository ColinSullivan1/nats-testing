
# Terraform to install and run the NATS cluster latency test

# Installation

First install the required tools, 'terraform', and 'mkcert'.

Terraform installation instructions can be found [here](https://www.terraform.io/intro/getting-started/install.html).
mkcert can be found [here](https://github.com/FiloSottile/mkcert).

The first time you are running terraform here, you'll need initialize it
by running the `terraform init` command in the relevant provider directory.

## First time Usage

From each provider director, to provision the test instances, simply run the command:
`$ terraform apply`

This will do the following:

* Create a `NATS Latency Testing` project in packet
* Add a server machine, `servera`
* Add a server machine, `serverb`
* Add a client machine, `client`
* Install scripts, SSL certificates, and relevant configuration files
* Route the servers to each other
* Launch the NATS servers
* Print relevant information.

This sets up a latency test that can be envisioned as a triangle.  

```text
Server A - - - - - Server B
    \                /
     \              /
      \            /
       \          /
        \        /
    Latency Test (client)
```

## Why the latency client on its own instance

This is the best way to measure end to end latency with respect to timing.  As we
are in the low microsecond range of measurements and measuring tail latency
on higher end machines, we need to very accurately measure time deltas.
This either requires a) sophisticated kernal time syncronization of a machine
provisioned in the cloud, or b) use the same kernel instance to measure time.
In the interest of simplicity and brevity, we chose "b".

## Manually running the tests

To manually run tests, ssh to the client machine, and run the `run_tests.sh` script.





