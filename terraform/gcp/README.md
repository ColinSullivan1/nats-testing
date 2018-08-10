# GCP latency testing with terraform

## Getting Started

A prerequisite to running this test on GCP is to setup a GCP account that can run  

1) Create a GCP account and project.
2) Generate an account.json file, as described here:  https://medium.com/@josephbleroy/using-terraform-with-google-cloud-platform-part-1-n-6b5e4074c059
3) Enable the compute engine API and ensure there are proper permissions.

## Running the test

Once all of the google resources are setup, you can run the test.  The test will do the following for you automatically:

1) Setup three GCE instances ("servera", "serverb", and "client")
2) Install and start clustered NATS servers.
3) Run a series of latency tests

### Variables

You can enter the variables via command line, manually when you run the test, or setup a `terraform.tfvars` file with your variables required to run.

Here's an example:

```text
project="nats-latency-testing"
account_json_file="terraform-account.json"
username="colinsullivan"
server_type = "f1-micro"
client_type = "f1-micro"

# You can override other variables found in variables.tf
# e.g.
# private_key_file = "gce_private_key.pem"
# zone = "asia-south1"
# region = "asia-south1-c"

```