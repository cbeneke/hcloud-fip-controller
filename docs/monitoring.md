# Monitoring

The controller exposes Prometheus metrics and can emit OpenTelemetry traces.

## Metrics

Metrics are served in Prometheus format at `/metrics` on the health server
(`HEALTH_CHECK_ADDRESS`, default `:8080`). In addition to the standard Go
runtime and process collectors, the following controller metrics are exported:

| Metric                                         | Type      | Description                                             |
|------------------------------------------------|-----------|--------------------------------------------------------|
| `fip_controller_reconciliations_total`         | counter   | Reconciliation runs, labelled by `result` (success/error) |
| `fip_controller_reconcile_duration_seconds`    | histogram | Duration of reconciliation runs                        |
| `fip_controller_floating_ip_reassignments_total` | counter | Floating IP (re)assignments performed                  |
| `fip_controller_managed_floating_ips`          | gauge     | Number of floating IPs currently managed               |
| `fip_controller_leader`                        | gauge     | `1` if this instance is the leader, otherwise `0`      |

### Scraping with the Prometheus Operator

The Helm chart can create a `Service` and a `ServiceMonitor` for scraping:

```yaml
monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    labels:
      release: kube-prometheus-stack
```

This requires the Prometheus Operator CRDs to be installed in the cluster.

## Tracing (OpenTelemetry)

Traces are only emitted when an OTLP endpoint is configured via
`OTEL_EXPORTER_OTLP_ENDPOINT` (see [configuration](./configuration.md)). When
unset, tracing is fully disabled with no runtime overhead.

Each reconciliation run produces a span (`UpdateFloatingIPs`) with attributes
for the number of managed floating IPs and running servers, and an event per
floating IP reassignment. Traces are exported over OTLP/gRPC.

Configure the endpoint through the Helm chart:

```yaml
monitoring:
  otelEndpoint: otel-collector.observability:4317
```
