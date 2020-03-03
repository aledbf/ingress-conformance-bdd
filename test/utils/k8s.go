package utils

import (
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	v1beta1 "k8s.io/api/networking/v1beta1"
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

const (
	// NamespaceCleanupTimeout failures caused by leaked resources from a previous test run.
	NamespaceCleanupTimeout = 5 * time.Minute
	// WaitForIngressAddressTimeout wait time for valid ingress status
	WaitForIngressAddressTimeout = 5 * time.Minute
	// IngressWaitInterval time to wait between checks for a condition
	IngressWaitInterval = 5 * time.Second
)

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
func WaitForServiceEndpointsNum(c clientset.Interface, namespace, serviceName string,
	expectNum int, interval, timeout time.Duration) error {
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

// CreateIngress creates an Ingress object and retunrs it, throws error if it already exists.
func CreateIngress(c kubernetes.Interface, ingress *v1beta1.Ingress) (*v1beta1.Ingress, error) {
	err := createIngressWithRetries(c, ingress.Namespace, ingress)
	if err != nil {
		return nil, err
	}

	return c.NetworkingV1beta1().Ingresses(ingress.Namespace).Get(ingress.Name, metav1.GetOptions{})
}

func createIngressWithRetries(c kubernetes.Interface, namespace string, obj *v1beta1.Ingress) error {
	if obj == nil {
		return fmt.Errorf("object provided to create is empty")
	}

	createFunc := func() (bool, error) {
		_, err := c.NetworkingV1beta1().Ingresses(namespace).Create(obj)
		if err == nil {
			return true, nil
		}

		if apierrs.IsAlreadyExists(err) {
			return false, err
		}

		if IsRetryableAPIError(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to create object with non-retriable error: %v", err)
	}

	return retryWithExponentialBackOff(createFunc)
}

// WaitForIngressAddress waits for the Ingress to acquire an address.
func WaitForIngressAddress(c clientset.Interface, ns, ingName, class string, timeout time.Duration) (string, error) {
	var address string

	err := wait.PollImmediate(IngressWaitInterval, timeout, func() (bool, error) {
		ipOrNameList, err := getIngressAddress(c, ns, ingName, class)
		if err != nil || len(ipOrNameList) == 0 {
			if IsRetryableAPIError(err) {
				return false, nil
			}

			return false, err
		}

		address = ipOrNameList[0]
		return true, nil
	})

	return address, err
}

// getIngressAddress returns the ips/hostnames associated with the Ingress.
func getIngressAddress(c clientset.Interface, ns, name, class string) ([]string, error) {
	ing, err := c.NetworkingV1beta1().Ingresses(ns).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var addresses []string

	for _, a := range ing.Status.LoadBalancer.Ingress {
		if a.IP != "" {
			addresses = append(addresses, a.IP)
		}

		if a.Hostname != "" {
			addresses = append(addresses, a.Hostname)
		}
	}

	return addresses, nil
}

const (
	// Parameters for retrying with exponential backoff.
	retryBackoffInitialDuration = 100 * time.Millisecond
	retryBackoffFactor          = 3
	retryBackoffJitter          = 0
	retryBackoffSteps           = 6
)

// Utility for retrying the given function with exponential backoff.
func retryWithExponentialBackOff(fn wait.ConditionFunc) error {
	backoff := wait.Backoff{
		Duration: retryBackoffInitialDuration,
		Factor:   retryBackoffFactor,
		Jitter:   retryBackoffJitter,
		Steps:    retryBackoffSteps,
	}

	return wait.ExponentialBackoff(backoff, fn)
}
