# ingress-conformance
Conformance test suite for Kubernetes Ingress (POC)

### Requirements (open to changes)

- Existing, running, Kubernetes cluster.
- An ingress controller is installed and running.
- e2e tests use the ingress status field to determine the FQDN/IP address to be used in the base URL.
- Is not relevant if the cluster is running in a cloud provider (or not).
- Only ports 80 and 443 are used.
- Tests requiring a TLS connection generate self-signed certificates.

### Usage:

```
  make <target>
  help             Display this help
  test             Run conformance tests using 'go test' (local development)
  build-image      Build image to run conformance test suite
  run-conformance  Run conformance tests using a pod
  build-report     Run tests and generate HTML report in directory
  show-report      Starts NGINX locally to access reports using http://localhost
  local-cluster    Create local cluster using kind
  codegen          Generate or update missing Go code defined in feature files
  verify-codegen   Verifies if generated Go code is in sync with feature files
```

### Run tests

```
make local-cluster (optional)
make test
```

### Run tests and prepare reports

```
make local-cluster (optional)
make show-report
```

### Test a different ingress controller

1. Fork the repository
2. Configure project Secret `INGRESS_CONTROLLER`:
  - Containing a link to an installation script, like https://gist.github.com/aledbf/7e67bcb338fa6a1696eb5b101597224e (Raw link)

The secret is used in the Github Action [Workflow](https://github.com/aledbf/ingress-conformance-bdd/blob/master/.github/workflows/main.yaml#L34)

That is it. The Github workflow is ready to run using a different ingress controller or configuration.


#### TODO:

- Use an existing Kubernetes cluster (using `KUBECONFIG`)
- Test a Cloud Ingress controller, like [ingress-gce](https://github.com/kubernetes/ingress-gce)
- Run the test suite agains multiple Kubernetes versions
