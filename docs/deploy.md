# Deploy to kubernetes
## Using helm

Install helm in the latest version and add the cbeneke/helm-charts repo

```
$ helm repo add cbeneke https://cbeneke.github.com/helm-charts
$ helm repo update
```

Then configure and install the chart

```
# When using helm v3:
$ helm install hcloud-fip-controller -f values.yaml cbeneke/hcloud-fip-controller

# OR helm v2:
$ helm install --name hcloud-fip-controller -f values.yaml cbeneke/hcloud-fip-controller
```

Please refer to the [helm-charts repo](https://github.com/cbeneke/helm-charts/tree/master/charts/hcloud-fip-controller#configuration)
which config options are available.

## Manual installation
You can install the deployment in a kubernetes cluster with the
following commands

```
$ kubectl create namespace fip-controller
$ kubectl apply -f deploy/rbac.yaml
$ kubectl apply -f deploy/deployment.yaml
$ cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: fip-controller-config
  namespace: fip-controller
data:
  config.json: |
    {
      "hcloud_floating_ips": [
        "<hcloud-floating-ip>",
        "<hcloud-floating-ip>",
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

### Using a DaemonSet

Alternatively, you can run the controller as a DaemonSet. This ensures
one controller will run on each worker node. Instead of using the
`deployment.yaml` file apply the `daemonset.yaml` file from the deploy
folder.
