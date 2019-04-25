# WIP
This project is currently work in progress. It might not work and/or not be documented. Please use with care and feel free to open bug tickets and pull requests. :)

# hcloud-fip-controller
[![Go Report Card](https://goreportcard.com/badge/github.com/cbeneke/hcloud-fip-controller)](https://goreportcard.com/report/github.com/cbeneke/hcloud-fip-controller)
[![GitHub license](https://img.shields.io/github/license/cbeneke/hcloud-fip-controller.svg)](https://github.com/cbeneke/hcloud-fip-controller/blob/master/LICENSE)

This is a small controller, meant to be running on a kubernetes cluster on hetzner cloud virtual machines.

The running pod will check the hetzner cloud API every 30 seconds to check if the configured IP Address (or subnet in case of IPv6) is assigned to the server, the running pod is scheduled to. If not it will update the assignment.
You need to make sure, to have the IP Address(es) configured on **every** node for this failover to correctly work, as the controller will not take care of the network configuration of the node.

# Configuration

## ENV variables

* NODE_NAME (required): The name of the scheduled node. Should be invoked via
  fieldRef to spec.nodeName
* HETZNER_API_TOKEN: The API token for the hetzner cloud. This must be set
  either via ENV or config file

## config.json fields

* hetznerApiToken: The API token for the hetzner cloud. This must be set either
  via ENV or config file
* floatingIPAddress: The floating IP address. If IPv6 it may be any IP address
  located in the /64 net
* nodeAddressType: Which node address type to check. This defaults to
  "external" but might be set to "internal", if the external Cluster IP is
  stored in the NodeInternalIP field (e.g. if cluster is not set up with hetzner
  cloud controller).

# Deploy to kubernetes

You can install the deployment in a kubernetes cluster with the following
commands

```
$ kubectl create namespace fip-controller
$ kubectl apply -f deploy/rbac.yaml
$ kubectl applf -f deploy/deployment.yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: fip-controller-config
  namespace: fip-controller
data:
  config: |
    {
      "floatingIPAddress": "<floating_ip_address>"
    }
---
apiVersion: v1
kind: Secret
metadata:
  name: hcloud
  namespace: fip-controller
stringData:
  token: <hetzner_api_token>
```
