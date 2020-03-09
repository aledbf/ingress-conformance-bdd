FROM k8s.gcr.io/debian-base-amd64:v2.0.0

RUN clean-install \
    bash \
    ca-certificates \
    curl

ARG KUBE_VERSION

RUN curl -sSL https://storage.googleapis.com/kubernetes-release/release/${KUBE_VERSION}/bin/linux/amd64/kubectl -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl

COPY run_e2e.sh /
COPY features /features
COPY ingress-conformance-bdd.test /

ENV RESULTS_DIR="/tmp/results"
ENV KUBECONFIG=""

CMD [ "/run_e2e.sh" ]
