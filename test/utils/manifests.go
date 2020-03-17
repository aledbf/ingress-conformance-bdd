package utils

import (
	"fmt"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	// IngressClassKey indicates the class of an Ingress to be used
	// when determining which controller should implement the Ingress
	IngressClassKey = "kubernetes.io/ingress.class"

	ingressFile               = "ing.yaml"
	replicationControllerFile = "rc.yaml"
	serviceFile               = "svc.yaml"
	secretFile                = "secret.yaml"
)

var (
	// IngressClassValue sets the value of the class of Ingresses
	IngressClassValue = ""
)

// CreateFromPath creates the Ingress and associated service/rc.
// Required: ing.yaml, rc.yaml, svc.yaml must exist in manifestPath
// Optional: secret.yaml, ingAnnotations, svcAnnotations
// If ingAnnotations is specified it will overwrite any annotations in ing.yaml
// If svcAnnotations is specified it will overwrite any annotations in svc.yaml
func CreateFromPath(c clientset.Interface,
	manifestPath, ns string,
	ingAnnotations map[string]string,
	svcAnnotations map[string]string) (*networkingv1beta1.Ingress, error) {

	rc := new(corev1.ReplicationController)
	err := createFromFile(filepath.Join(manifestPath, replicationControllerFile), ns, rc)
	if err != nil {
		return nil, err
	}

	_, err = c.CoreV1().ReplicationControllers(ns).Create(rc)
	if err != nil {
		return nil, err
	}

	svc := new(corev1.Service)
	err = createFromFile(filepath.Join(manifestPath, serviceFile), ns, svc)
	if err != nil {
		return nil, err
	}

	if len(svcAnnotations) > 0 {
		svc.Annotations = svcAnnotations
	}

	_, err = c.CoreV1().Services(ns).Create(svc)
	if err != nil {
		return nil, err
	}

	err = WaitForServiceEndpointsNum(c, ns, svc.Name, 1, 2*time.Second, 5*time.Minute)
	if err != nil {
		return nil, err
	}

	secretPath := filepath.Join(manifestPath, secretFile)
	if Exists(secretPath) {
		secret := new(corev1.Secret)
		err = createFromFile(filepath.Join(manifestPath, secretFile), ns, secret)
		if err != nil {
			return nil, err
		}

		_, err = c.CoreV1().Secrets(ns).Create(secret)
		if err != nil {
			return nil, err
		}
	}

	ing := new(networkingv1beta1.Ingress)
	err = createFromFile(filepath.Join(manifestPath, ingressFile), ns, ing)
	if err != nil {
		return nil, err
	}

	if ing.Annotations == nil {
		ing.Annotations = map[string]string{}
	}

	if len(ingAnnotations) > 0 {
		ing.Annotations = ingAnnotations
	}

	if IngressClassValue != "" {
		ing.Annotations[IngressClassKey] = IngressClassValue
	}

	ing, err = c.NetworkingV1beta1().Ingresses(ns).Create(ing)
	if err != nil {
		return nil, err
	}

	return ing, nil
}

func createFromFile(path, ns string, obj runtime.Object) error {
	file, err := filesource.GetAbsPath(path)
	if err != nil {
		return err
	}

	if exists := Exists(file); !exists {
		return fmt.Errorf("file %v does not exists", file)
	}

	data, err := Read(file)
	if err != nil {
		return err
	}

	json, err := utilyaml.ToJSON(data)
	if err != nil {
		return err
	}

	if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), json, obj); err != nil {
		return err
	}

	return nil
}

// IngressFromManifest reads a .json/yaml file and returns the ingress in it.
func IngressFromManifest(file, namespace string) (*networkingv1beta1.Ingress, error) {
	ing := new(networkingv1beta1.Ingress)

	err := createFromFile(file, namespace, ing)
	if err != nil {
		return nil, err
	}

	ing.SetNamespace(namespace)

	if ing.Annotations == nil {
		ing.Annotations = map[string]string{}
	}

	if IngressClassValue != "" {
		ing.Annotations[IngressClassKey] = IngressClassValue
	}

	return ing, nil
}
