# Configuration

Environment variables take precedence over the config file

## ENV variables

* FLOATING_IPS_LABEL_SELECTOR
Selector for floating ips in case not all floating ips should be used in the controller. This will be ignored when hcloud_floating_ips are defined.

* HCLOUD_API_TOKEN  
API token for the hetzner cloud access.

* HCLOUD_FLOATING_IP **deprecated**
Floating IP you want to configure. In case of IPv6 can be any of the /64 net. If you want to use multiple IPs use config file or command line parameters. When no floating ips are given, the controller will auto discover them from the hetzner api.

* LEASE_NAME, *default:* "fip"  
Name of the lease created by the lease lock used for leader election

* LEASE_DURATION, *default:* 15
Duration of the lease used by the lease lock. This is the maximum time until a new leader will be elected in case of failure.
More about the leaderelection variables can be found [here](https://godoc.org/k8s.io/client-go/tools/leaderelection).

* LEASE_RENEW_DEADLINE, *default* 10
Duration that the master will retry refreshing its leadership before giving up.
Must be smaller than LEASE_DURATION

* LOG_LEVEL, *default*: Info  
Log level of the controller.

* NODE_ADDRESS_TYPE, *default:* "external"  
Address type of the nodes. This might be set to internal, if your external IPs are  registered as internal IPs on the node objects (e.g. if you have no cloud controller manager). Can be "external" or "internal".

* NODE_LABEL_SELECTOR
Optionally restrict the searched nodes to assign floating ips to by a label selector.
More infos about labels selectors can be found [here](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors)

* NODE_NAME  
Name of the scheduled node. Should be invoked via fieldRef to spec.nodeName

* NAMESPACE  
Namespace the pod is running in. Should be invoked via fieldRef to metadata.namespace

* POD_LABEL_SELECTOR 
Labels selector to find deployment pods with. When this field is empty, the fip-controller will use all labels on its own pod. This is the intended behaviour in most cases.
When the fip-controller deployment has no labels, no pods will be found and the fip-controller will look for ips in nodes instead.

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
  "node_label_selector": "<NODE_LABEL_SELECTOR>",
  "node_name": "<NODE_NAME>",
  "pod_label_selector": "<POD_LABEL_SELECTOR>",
  "pod_name": "<POD_NAME>"
}
```
