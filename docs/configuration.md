# Configuration

Environment variables take precedence over the config file

## ENV variables

* HCLOUD_API_TOKEN  
API token for the hetzner cloud access.

* HCLOUD_FLOATING_IP  
Floating IP you want to configure. In case of IPv6 can be any of the /64 net. If you want to use multiple IPs use config file or command line parameters

* LEASE_NAME, *default:* "fip"  
Name of the lease created by the lease lock used for leader election

* LEASE_DURATION, *default:* 30  
Duration of the lease used by the lease lock. This is the maximum time until a new leader will be elected in case of failure.

* LOG_LEVEL, *default*: Info  
Log level of the controller.

* NODE_ADDRESS_TYPE, *default:* "external"  
Address type of the nodes. This might be set to internal, if your external IPs are  registered as internal IPs on the node objects (e.g. if you have no cloud controller manager). Can be "external" or "internal".

* NODE_NAME  
Name of the scheduled node. Should be invoked via fieldRef to spec.nodeName

* NAMESPACE  
Namespace the pod is running in. Should be invoked via fieldRef to metadata.namespace

* POD_NAME  
Name of the pod. Should be invoked via fieldRef to metadata.name

## config.json fields

Valid fields in the config.json file and their respective ENV variables are

```json
{
  "hcloud_floating_ips": [
    "<HCLOUD_FLOATING_IP>"
  ],
  "hcloud_api_token": "<HCLOUD_API_TOKEN>",
  "lease_duration": "<LEASE_DURATION>",
  "lease_name": "<LEASE_NAME>",
  "log_level": "<LOG_LEVEL>",
  "namespace": "<NAMESPACE>",
  "node_address_type": "<NODE_ADDRESS_TYPE>",
  "node_name": "<NODE_NAME>",
  "pod_name": "<POD_NAME>"
}
```
