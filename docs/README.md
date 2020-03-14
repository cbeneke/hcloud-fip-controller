# hcloud-fip-controller
hcloud-fip-controller is a small controller, to handle floating IP management in a kubernetes cluster on hetzner cloud virtual machines.

The running pods will check the hetzner cloud API every 30 seconds to check if the configured IP Addresses (or subnets in case of IPv6) are assigned to the server, the leader is scheduled to. If not it will update the assignments.
You need to make sure, to have the IP Addresses configured on **every** node for this failover to correctly work, as the controller will not take care of the network configuration of the nodes.

# Table of Contents
* [Configuration](configuration.md)
* [Deploy to Kubernetes](deploy.md)
* [Running multiple controller](multiple_controller.md)
