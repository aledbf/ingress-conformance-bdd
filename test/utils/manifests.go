package utils

import (
	"fmt"
	"path/filepath"

	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// CreateFromPath creates the Ingress and associated service/rc.
// Required: ing.yaml, rc.yaml, svc.yaml must exist in manifestPath
// Optional: secret.yaml, ingAnnotations
// If ingAnnotations is specified it will overwrite any annotations in ing.yaml
// If svcAnnotations is specified it will overwrite any annotations in svc.yaml
func CreateFromPath(c clientset.Interface, manifestPath, ns string,
	ingAnnotations map[string]string, svcAnnotations map[string]string) error {
	files := []string{
		replicationControllerFile,
		serviceFile,
		secretFile,
	}

	for _, file := range files {
		content, err := Read(filepath.Join(manifestPath, file))
		if err != nil {
			return err
		}

		_, err = RunKubectlInput(ns, string(content), "create", "-f", "-", fmt.Sprintf("--namespace=%v", ns))
		if err != nil {
			return err
		}
	}

	if len(svcAnnotations) > 0 {
		svcList, err := c.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil {
			return err
		}

		for _, svc := range svcList.Items {
			s := &svc
			s.Annotations = svcAnnotations

			_, err = c.CoreV1().Services(ns).Update(s)
			if err != nil {
				return err
			}
		}
	}

	if exists := Exists(filepath.Join(manifestPath, secretFile)); exists {
		content, err := Read(filepath.Join(manifestPath, secretFile))
		if err != nil {
			return err
		}

		_, err = RunKubectlInput(ns, string(content), "create", "-f", "-", fmt.Sprintf("--namespace=%v", ns))
		if err != nil {
			return err
		}
	}

	ingress, err := IngressFromManifest(filepath.Join(manifestPath, ingressFile))
	if err != nil {
		return err
	}

	ingress.Namespace = ns
	ingress.Annotations = map[string]string{
		IngressClassKey: "",
	}

	for k, v := range ingAnnotations {
		ingress.Annotations[k] = v
	}

	_, err = CreateIngress(c, ingress)
	if err != nil {
		return err
	}

	return nil
}

// IngressFromManifest reads a .json/yaml file and returns the ingress in it.
func IngressFromManifest(fileName string) (*networkingv1beta1.Ingress, error) {
	var ing networkingv1beta1.Ingress
	data, err := Read(fileName)
	if err != nil {
		return nil, err
	}

	json, err := utilyaml.ToJSON(data)
	if err != nil {
		return nil, err
	}
	if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), json, &ing); err != nil {
		return nil, err
	}
	return &ing, nil
}
