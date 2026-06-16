# Deploy to kubernetes

The controller ships as a Helm chart in the [`deploy/`](../deploy) folder of
this repository. The chart version and `appVersion` are kept in lockstep and
bumped automatically on every release.

## Using helm

Every release publishes the chart as an OCI artifact to Docker Hub. Install a
released version directly:

```
$ helm install hcloud-fip-controller \
    oci://registry-1.docker.io/cbeneke/hcloud-fip-controller \
    --namespace fip-controller \
    --create-namespace \
    --set hcloudApiToken=<hcloud_api_token>
```

Use `--version <x.y.z>` to pin a specific chart version (the chart version
matches the application version).

Alternatively, install the chart directly from a checkout of this repository.
Provide your Hetzner Cloud API token and let the chart create the namespace:

```
$ helm install hcloud-fip-controller ./deploy \
    --namespace fip-controller \
    --create-namespace \
    --set hcloudApiToken=<hcloud_api_token>
```

Alternatively, supply your own values file:

```
$ helm install hcloud-fip-controller ./deploy \
    --namespace fip-controller \
    --create-namespace \
    -f values.yaml
```

### Using an existing secret

If you manage the API token yourself, point the chart at an existing secret
that contains an `HCLOUD_API_TOKEN` key instead of letting the chart create one:

```
$ helm install hcloud-fip-controller ./deploy \
    --namespace fip-controller \
    --create-namespace \
    --set existingSecretName=my-hcloud-secret
```

### Using a DaemonSet

By default the controller runs as a `Deployment` with 3 replicas. To run one
controller per node, deploy it as a `DaemonSet` instead:

```
$ helm install hcloud-fip-controller ./deploy \
    --namespace fip-controller \
    --create-namespace \
    --set kind=DaemonSet \
    --set hcloudApiToken=<hcloud_api_token>
```

### Configuration

All controller options can be passed as environment variables through the
`config` map in the values. See the [configuration options](./configuration.md)
for the full list. Example `values.yaml`:

```yaml
hcloudApiToken: <hcloud_api_token>

config:
  NODE_ADDRESS_TYPE: external
  LOG_LEVEL: Info
```

The most relevant chart values are:

| Value               | Default                        | Description                                       |
|---------------------|--------------------------------|---------------------------------------------------|
| `kind`              | `Deployment`                   | Workload kind (`Deployment` or `DaemonSet`)       |
| `replicaCount`      | `3`                            | Replicas (Deployment only)                        |
| `image.repository`  | `cbeneke/hcloud-fip-controller`| Container image repository                        |
| `image.tag`         | `""`                           | Image tag, defaults to `v<appVersion>`            |
| `hcloudApiToken`    | `""`                           | Hetzner Cloud API token (creates a Secret)        |
| `existingSecretName`| `""`                           | Use an existing secret with `HCLOUD_API_TOKEN`    |
| `config`            | `{}`                           | Extra controller options as environment variables |
| `healthCheck.port`  | `8080`                         | Port for the liveness/readiness/metrics endpoints |
| `monitoring.otelEndpoint` | `""`                     | OTLP endpoint for traces (traces emitted only when set) |
| `monitoring.serviceMonitor.enabled` | `false`       | Create a Service + Prometheus Operator ServiceMonitor |

See [monitoring](./monitoring.md) for the exported metrics and tracing details.

## Manual installation

If you prefer not to use Tiller/Helm releases, render the chart and apply the
manifests directly:

```
$ kubectl create namespace fip-controller
$ helm template hcloud-fip-controller ./deploy \
    --namespace fip-controller \
    --set hcloudApiToken=<hcloud_api_token> | kubectl apply -f -
```
