
# Terraform to install and run the NATS cluster latency test

blah

# Installation

First install the required tools, 'terraform', and 'mkcert'.

Terraform installation instructions can be found [here](https://www.terraform.io/intro/getting-started/install.html).
mkcert can be found [here](https://github.com/FiloSottile/mkcert).

The first time you are running terraform here, you'll need initialize it 
by running the `terraform init` command in the current directory.

# Packet API Key

Setup a packet account.  Once you have generated an API key from the user profile or
project section of your packet account, you'll need to set the following variable.  

This can be done by creating a 'terraform.tfvars' in this directory with the following 
variable defined:

```text
auth_token = "<your auth token>"
```

# Provisioning the machine instances

## First time Usage

From this directory, to provision the test instances, simply run the command:
`$ terraform apply`

This will do the following:

* Create a `NATS Latency Testing` project in packet
* Add a server machine, `servera.latency.nats.com`
* Add a server machine, `serverb.latency.nats.com`
* Add a client machine, `client.latency.nats.com`
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

## Why the latency client on its own machine

This is the best way to measure end to end latency with respect to timing.  As we
are in the low microsecond range of measurements and measuring tail latency
on higher end machines, we need to very accurately measure time deltas.
This either requires a) sophisticated kernal time syncronization of a machine
provisioned in the cloud, or b) use the same kernel instance to measure time.
In the interest of simplicity and brevity, we chose "b".

# Selecting different machine instances

You can select different machine instances using the `latency_server_type` 
and `latency_client_type` variables. e.g.

latency_server_type = "baremetal_1"
latency_client_type = "baremetal_1"

Descriptions of available machines can be found in [variables.tf]("./variables.tf"),
although you may want to check for updates.  More information on terraform
packet device types can be found [here](https://www.terraform.io/docs/providers/packet/r/device.html).

# Running the tests

Right now, while machine provisioning and server routing are automatic, the tests
are run manually.

## Testing the setup


## Manually running the tests

ssh to the packet client machine



