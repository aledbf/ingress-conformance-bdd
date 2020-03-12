package conformance

import (
	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"k8s.io/client-go/kubernetes"
)

// KubeClient Kubernetes API client
var KubeClient *kubernetes.Clientset

var state *tstate.Scenario
