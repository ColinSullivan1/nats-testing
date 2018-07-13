# Configure the Packet Provider
provider "packet" {
  auth_token = "${var.auth_token}"
}

# Create a project
resource "packet_project" "latency" {
  name = "NATS Latency Testing"
}

# Create a device for the server and add it to our project
resource "packet_device" "nats_server_a" {
  hostname         = "nats.server.a"
  plan             = "${var.latency_server_type}"
  facility         = "sjc1"
  operating_system = "ubuntu_18_04"
  billing_cycle    = "hourly"
  project_id       = "${packet_project.latency.id}"

  # Copy our install scripts to the machine
  provisioner "file" {
    source      = "bootstrap/"
    destination = "~/"
  }

  provisioner "file" {
    source      = "servera/"
    destination = "~/"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x ~/*.sh",
      "~/install.sh",
    ]
  }
}

resource "packet_device" "nats_server_b" {
  hostname         = "nats.server.b"
  plan             = "${var.latency_server_type}"
  facility         = "sjc1"
  operating_system = "ubuntu_18_04"
  billing_cycle    = "hourly"
  project_id       = "${packet_project.latency.id}"

  # Copy our install script to the machine
  provisioner "file" {
    source      = "bootstrap/"
    destination = "~/"
  }

  provisioner "file" {
    source      = "serverb/"
    destination = "~/"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x ~/*.sh",
      "~/install.sh",
    ]
  }
}

# Create devices for the latency clients and add it to our
# project
resource "packet_device" "nats_client" {
  hostname         = "nats.client"
  plan             = "${var.latency_client_type}"
  facility         = "sjc1"
  operating_system = "ubuntu_18_04"
  billing_cycle    = "hourly"
  project_id       = "${packet_project.latency.id}"

  # Copy our install script to the machine
  provisioner "file" {
    source      = "bootstrap/"
    destination = "~/"
  }

  provisioner "file" {
    source      = "client/"
    destination = "~/"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x ~/*.sh",
      "~/install.sh",
    ]
  }
}

resource "null_resource" "client_update_hosts" {
  # Not perfect, but any changes to any instance of the machines 
  # requires updates
  triggers {
    cluster_instance_ids = "${packet_device.nats_client.id}, ${packet_device.nats_server_a.id}, ${packet_device.nats_server_b.id}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${lookup(packet_device.nats_client.0.network[0], "address")}"
  }

  provisioner "remote-exec" {
    # Add private_ip of each server in the cluster
    inline = [
      "echo '${lookup(packet_device.nats_server_a.0.network[2], "address")} servera nats.server.a' >> /etc/hosts ",
      "echo '${lookup(packet_device.nats_server_b.0.network[2], "address")} serverb nats.server.b' >> /etc/hosts ",
    ]
  }
}

##
## This next set of resources updates the /etc/host files so
## we can have a static nats server config.
##
resource "null_resource" "server_a_update_hosts" {
  # Not perfect, but any changes to any instance of the machines 
  # requires updates
  triggers {
    cluster_instance_ids = "${packet_device.nats_client.id}, ${packet_device.nats_server_a.id}, ${packet_device.nats_server_b.id}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${lookup(packet_device.nats_server_a.0.network[0], "address")}"
  }

  provisioner "remote-exec" {
    # Add private_ip of each server in the cluster
    inline = [
      "echo '${lookup(packet_device.nats_client.0.network[2], "address")} servera nats.server.a' >> /etc/hosts ",
      "echo '${lookup(packet_device.nats_server_b.0.network[2], "address")} serverb nats.server.b' >> /etc/hosts ",
      "~/run_server.sh",
    ]
  }
}

resource "null_resource" "server_b_update_hosts" {
  # Not perfect, but any changes to any instance of the machines 
  # requires updates
  triggers {
    cluster_instance_ids = "${packet_device.nats_client.id}, ${packet_device.nats_server_a.id}, ${packet_device.nats_server_b.id}"
  }

  # Bootstrap script can run on any instance of the cluster
  # So we just choose the first in this case
  connection {
    host = "${lookup(packet_device.nats_server_b.0.network[0], "address")}"
  }

  provisioner "remote-exec" {
    # Add private_ip of each server in the cluster
    inline = [
      "echo '${lookup(packet_device.nats_client.0.network[2], "address")} servera nats.server.a' >> /etc/hosts ",
      "echo '${lookup(packet_device.nats_server_a.0.network[2], "address")} serverb nats.server.b' >> /etc/hosts ",
      "~/run_server.sh",
    ]
  }
}
