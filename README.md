# ingress-conformance
Conformance test suite for Kubernetes Ingress (POC)

### Requirements (open to changes)

- Existing, running, Kubernetes cluster.
- An ingress controller is installed and running.
- e2e tests use the ingress status field to determine the FQDN/IP address to be used in the base URL.
- Is not relevant if the cluster is running in a cloud provider (or not).
- Only ports 80 and 443 are used.
- Tests requiring a TLS connection generate self-signed certificates.
