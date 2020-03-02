package utils

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	// ensure auth plugins are loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// NamespaceCleanupTimeout failures caused by leaked resources from a previous test run.
const NamespaceCleanupTimeout = 5 * time.Minute

var KubeClient clientset.Interface

// WaitForService waits until the service appears (exist == true), or disappears (exist == false)
func WaitForService(c clientset.Interface, namespace, name string, exist bool, interval, timeout time.Duration) error {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		_, err := c.CoreV1().Services(namespace).Get(name, metav1.GetOptions{})
		switch {
		case err == nil:
			klog.Infof("Service %s in namespace %s found.", name, namespace)
			return exist, nil
		case apierrors.IsNotFound(err):
			klog.Infof("Service %s in namespace %s disappeared.", name, namespace)
			return !exist, nil
		case !IsRetryableAPIError(err):
			klog.Infof("Non-retryable failure while getting service.")
			return false, err
		default:
			klog.Infof("Get service %s in namespace %s failed: %v", name, namespace, err)
			return false, nil
		}
	})
	if err != nil {
		stateMsg := map[bool]string{true: "to appear", false: "to disappear"}
		return fmt.Errorf("error waiting for service %s/%s %s: %v", namespace, name, stateMsg[exist], err)
	}
	return nil
}

//WaitForServiceEndpointsNum waits until the amount of endpoints that implement service to expectNum.
func WaitForServiceEndpointsNum(c clientset.Interface, namespace, serviceName string, expectNum int, interval, timeout time.Duration) error {
	return wait.Poll(interval, timeout, func() (bool, error) {
		klog.Infof("Waiting for amount of service:%s endpoints to be %d", serviceName, expectNum)
		list, err := c.CoreV1().Endpoints(namespace).List(metav1.ListOptions{})
		if err != nil {
			return false, err
		}

		for _, e := range list.Items {
			if e.Name == serviceName && countEndpointsNum(&e) == expectNum {
				return true, nil
			}
		}
		return false, nil
	})
}

func countEndpointsNum(e *corev1.Endpoints) int {
	num := 0
	for _, sub := range e.Subsets {
		num += len(sub.Addresses)
	}
	return num
}

// LoadClientset returns clientset for connecting to kubernetes clusters.
func LoadClientset() (*clientset.Clientset, error) {
	config, err := restclient.InClusterConfig()
	if err != nil {
		// Attempt to use local KUBECONFIG
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		// use the current context in kubeconfig
		var err error
		config, err = kubeconfig.ClientConfig()
		if err != nil {
			return nil, err
		}
	}

	client, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// CreateTestNamespace creates a new namespace using
// ingress-conformance- as prefix.
func CreateTestNamespace(c kubernetes.Interface) (string, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "ingress-conformance-",
		},
	}

	var err error
	ns, err = c.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return "", fmt.Errorf("unable to create namespace: %v", err)
	}

	return ns.Name, nil
}

// DeleteKubeNamespace deletes a namespace and all the objects inside
func DeleteKubeNamespace(c kubernetes.Interface, namespace string) error {
	grace := int64(0)
	pb := metav1.DeletePropagationBackground
	return c.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
		PropagationPolicy:  &pb,
	})
}

func IsRetryableAPIError(err error) bool {
	// These errors may indicate a transient error that we can retry in tests.
	if apierrs.IsInternalError(err) || apierrs.IsTimeout(err) || apierrs.IsServerTimeout(err) ||
		apierrs.IsTooManyRequests(err) || utilnet.IsProbableEOF(err) || utilnet.IsConnectionReset(err) {
		return true
	}
	// If the error sends the Retry-After header, we respect it as an explicit confirmation we should retry.
	if _, shouldRetry := apierrs.SuggestsClientDelay(err); shouldRetry {
		return true
	}
	return false
}
