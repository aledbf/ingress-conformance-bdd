package utils

import (
	"fmt"
	"path/filepath"
	"strings"

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
// Optional: secret.yaml, ingAnnotations, svcAnnotations
// If ingAnnotations is specified it will overwrite any annotations in ing.yaml
// If svcAnnotations is specified it will overwrite any annotations in svc.yaml
func CreateFromPath(c clientset.Interface,
	manifestPath, ns string,
	ingAnnotations map[string]string,
	svcAnnotations map[string]string) (*networkingv1beta1.Ingress, error) {
	files := []string{
		replicationControllerFile,
		serviceFile,
		ingressFile,
	}

	for _, file := range files {
		f, err := filesource.GetAbsPath(filepath.Join(manifestPath, file))
		if err != nil {
			return nil, err
		}

		err = createFromFile(f, ns)
		if err != nil {
			return nil, err
		}
	}

	if len(svcAnnotations) > 0 {
		svcList, err := c.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, svc := range svcList.Items {
			s := &svc
			s.Annotations = svcAnnotations

			_, err = c.CoreV1().Services(ns).Update(s)
			if err != nil {
				return nil, err
			}
		}
	}

	if sf, err := filesource.GetAbsPath(filepath.Join(manifestPath, secretFile)); err == nil {
		if exists := Exists(sf); !exists {
			return nil, fmt.Errorf("file %v does not exists", sf)
		}

		_, err = RunKubectl(ns, "create", "-f", sf)
		if err != nil {
			return nil, err
		}
	}

	ingList, err := c.NetworkingV1beta1().Ingresses(ns).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(ingList.Items) == 0 {
		return nil, fmt.Errorf("there are no Ingress objects present in namespace %v", ns)
	}

	if len(ingAnnotations) > 0 {
		for _, ing := range ingList.Items {
			i := &ing
			i.Annotations = ingAnnotations

			_, err := c.NetworkingV1beta1().Ingresses(ns).Update(i)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ingList.Items[0], nil
}

func createFromFile(file, ns string) error {
	if exists := Exists(file); !exists {
		return fmt.Errorf("file %v does not exists", file)
	}

	out, err := RunKubectl(ns, "apply", "-f", file)
	if err != nil {
		return err
	}

	if strings.Contains(out, "service/") {
		// parse service name
		// wait until endpoints are available
	}

	return nil
}

// IngressFromManifest reads a .json/yaml file and returns the ingress in it.
func IngressFromManifest(file, namespace string) (*networkingv1beta1.Ingress, error) {
	var ing networkingv1beta1.Ingress

	data, err := Read(file)
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

	ing.SetNamespace(namespace)

	return &ing, nil
}
