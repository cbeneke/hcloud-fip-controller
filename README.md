# WIP
This project is currently work in progress. It might not work and/or not be documented. Please use with care and feel free to open bug tickets and pull requests. :)

# hcloud-fip-controller
[![Go Report Card](https://goreportcard.com/badge/github.com/cbeneke/hcloud-fip-controller)](https://goreportcard.com/report/github.com/cbeneke/hcloud-fip-controller)
[![GitHub license](https://img.shields.io/github/license/cbeneke/hcloud-fip-controller.svg)](https://github.com/cbeneke/hcloud-fip-controller/blob/master/LICENSE)

This is a small controller, meant to be running on a kubernetes cluster on hetzner cloud virtual machines.

The running pod will check the hetzner cloud API every 30 seconds to check if the configured IP Address (or subnet in case of IPv6) is assigned to the server, the running pod is scheduled to. If not it will update the assignment.
You need to make sure, to have the IP Address(es) configured on **every** node for this failover to correctly work, as the controller will not take care of the network configuration of the node.

# Configuration

Environment variables take precedence over the config file

## ENV variables

* HCLOUD_API_TOKEN  
API token for the hetzner cloud access.

* HCLOUD_FLOATING_IP  
Floating IP you want to configure. In case of IPv6 can be any of the /64 net. If you want to use multiple IPs use config file or command line parameters

* NODE_ADDRESS_TYPE, *default:* "external"  
Address type of the nodes. This might be set to internal, if your external IPs are  registered as internal IPs on the node objects (e.g. if you have no cloud controller manager). Can be "external" or "internal".

* NODE_NAME  
name of the scheduled node. Should be invoked via fieldRef to spec.nodeName

## config.json fields

Valid fields in the config.json file and their respective ENV variable are

```json
{
  "hcloudFloatingIPs": [
    "<HCLOUD_FLOATING_IP>"
  ],
  "hcloudApiToken": "<HCLOUD_API_TOKEN>",
  "nodeAddressType": "<NODE_ADDRESS_TYPE>",
  "nodeName": "<NODE_NAME>"
}
```

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
  config.json: |
    {
      "hcloudFloatingIPs": [
        "<hcloud-floaiting-ip>",
        "<hcloud-floaiting-ip>",
        ...
      ]
    }
---
apiVersion: v1
kind: Secret
metadata:
  name: fip-controller-secrets
  namespace: fip-controller
stringData:
  HCLOUD_API_TOKEN: <hcloud_api_token>
```
