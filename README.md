# ingress-conformance
Conformance test suite for Kubernetes Ingress (POC)

### Assumptions (open to change)

- Tests will use an existing, running, Kubernetes cluster.
- An ingress controller is installed and running.
- e2e tests use the ingress status field to determine the FQDN/IP address to be used in the base URL.
- Timeouts are configurable. The default is five minutes per test.
- Is not relevant if the cluster is running in a cloud provider (or not).
- Only ports 80 and 443 are used.
- HTTPS tests should generate self signed certificates using the FQDN/IP address of the ingress to test.
