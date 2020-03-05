package utils

import (
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/manifest"
)

const (
	IngressClassKey = "kubernetes.io/ingress.class"

	IngressFile               = "ing.yaml"
	ReplicationControllerFile = "rc.yaml"
	ServiceFile               = "svc.yaml"
	SecretFile                = "secret.yaml"
)

// CreateFromPath creates the Ingress and associated service/rc.
// Required: ing.yaml, rc.yaml, svc.yaml must exist in manifestPath
// Optional: secret.yaml, ingAnnotations
// If ingAnnotations is specified it will overwrite any annotations in ing.yaml
// If svcAnnotations is specified it will overwrite any annotations in svc.yaml
func CreateFromPath(c clientset.Interface, manifestPath, ns string, ingAnnotations map[string]string, svcAnnotations map[string]string) error {
	files := []string{
		ReplicationControllerFile,
		ServiceFile,
		SecretFile,
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
			svc.Annotations = svcAnnotations

			_, err = c.CoreV1().Services(ns).Update(&svc)
			if err != nil {
				return err
			}
		}
	}

	if exists := Exists(filepath.Join(manifestPath, SecretFile)); exists {
		content, err := Read(filepath.Join(manifestPath, SecretFile))
		if err != nil {
			return err
		}

		_, err = RunKubectlInput(ns, string(content), "create", "-f", "-", fmt.Sprintf("--namespace=%v", ns))
		if err != nil {
			return err
		}
	}

	ingress, err := manifest.IngressFromManifest(filepath.Join(manifestPath, IngressFile))
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
