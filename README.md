# loki-proxy

loki-proxy is a reverse proxy server to aggreate logs by tenant header ("X-Scope-OrgID") to global tenant and store orignal tenant as label

## Installing the Chart

This Helm chart repository enables you to install a loki-proxy Helm chart directly from it into your Kubernetes cluster.

```shell
# Add groundcover Helm repoistory (once)
helm repo add groundcover https://helm.groundcover.com

# Update groundcover Helm repository to fetch latest charts
helm repo update groundcover

# Deploy loki-proxy release
helm install loki-proxy groundcover/loki-proxy
```

## Configuration

The following table lists the configurable parameters of the template Helm chart and their default values.

| Parameter           | Description                                                | Default                             |
| ------------------- | ---------------------------------------------------------- | ----------------------------------- |
| `target.tenant_id`  | Global tenant ID to push all incoming logs                 | `customers`                         |
| `target.label_name` | Label name of orignal tenant header value on global tenant | `customer`                          |
| `taraget.url`       | Loki push api url                                          | `http://loki:3100/loki/api/v1/push` |
