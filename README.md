[![Go Report Card](https://goreportcard.com/badge/github.com/cbeneke/hcloud-fip-controller)](https://goreportcard.com/report/github.com/cbeneke/hcloud-fip-controller)

# WIP
This project is currently work in progress. It might not work and/or not be documented. Please use with care and feel free to open bug tickets and pull requests. :)

# hcloud-fip-controller

This is a small controller, meant to be running on a kubernetes cluster on hetzner cloud virtual machines. The pod must have a NODE_NAME environment variable declared:

```
apiVersion: v1
kind: Pod
metadata: {...}
spec:
  containers:
  - env:
    - name: NODE_NAME
      valueFrom:
        fieldRef:
          apiVersion: v1
          fieldPath: spec.nodeName
     [...]
```

The running pod will check the hetzner cloud API every 30 seconds to check if the configured IP Address (or subnet in case of IPv6) is assigned to the server, the running pod is scheduled to. If not it will update the assignment.
You need to make sure, to have the IP Address(es) configured on **every** node for this failover to correctly work, as the controller will not take care of the network configuration of the node.

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
