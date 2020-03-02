package defaultbackend

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"
	v1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type feature struct {
	state *state.Feature

	client *clientset.Clientset
}

func (f *feature) aNewRandomNamespace() error {
	ns, err := utils.CreateTestNamespace(f.client)
	if err != nil {
		return err
	}

	f.state.Namespace = ns
	return nil
}

func (f *feature) anIngressIsCreatedWithHostAndNoBackend(host string) error {
	ingSpec := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "defaultbackend",
			Namespace: f.state.Namespace,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
								},
							},
						},
					},
				},
			},
		},
	}

	f.state.SetIngress(ingSpec)
	return nil
}

func (f *feature) ingressCreationrrorMessageContains(expected string) error {
	_, err := utils.CreateIngress(f.client, f.state.GetIngress())
	if err == nil {
		return fmt.Errorf("Expected an error creating an ingress without backend serviceName")
	}

	if strings.Contains(err.Error(), expected) {
		return nil
	}

	return fmt.Errorf("Expected an error containing %v but returned %v", expected, err.Error())
}

func (f *feature) ingressStatusIPOrFQDN() error {
	if f.state.GetIngress() == nil {
		return fmt.Errorf("Feature without Ingress associated")
	}

	address, err := utils.WaitForIngressAddress(f.client, f.state.Namespace, f.state.GetIngress().GetName(), "", utils.WaitForIngressAddressTimeout)
	if err != nil {
		return err
	}

	f.state.SetStatusAddress(address)
	return nil
}

func (f *feature) headerWithValue(arg1, arg2 string) error {
	return nil
}

func (f *feature) sendHTTPRequestWithMethod(arg1 string) error {
	return nil
}

func (f *feature) responseStatusCodeIs(arg1 int) error {
	return nil
}

func (f *feature) headerIs(arg1, arg2 string) error {
	return nil
}

func (f *feature) anIngressIsCreatedWithFoobarHostWithInvalidBackend(host string) error {
	ingSpec := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "defaultbackend",
			Namespace: f.state.Namespace,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: "/",
									Backend: v1beta1.IngressBackend{
										ServiceName: "non-existing",
										ServicePort: intstr.FromInt(80),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	var err error
	ingSpec, err = utils.CreateIngress(f.client, ingSpec)
	if err != nil {
		return err
	}

	f.state.SetIngress(ingSpec)
	return nil
}

func (f *feature) sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs(arg1 int, arg2 *gherkin.DataTable) error {
	return godog.ErrPending
}

func (f *feature) withPath(arg1 string) error {
	return godog.ErrPending
}

func FeatureContext(s *godog.Suite, c *clientset.Clientset) {
	f := &feature{
		client: c,
	}

	s.Step(`^a new random namespace$`, f.aNewRandomNamespace)
	s.Step(`^creating an Ingress with host "([^"]*)" without backend serviceName$`, f.anIngressIsCreatedWithHostAndNoBackend)
	s.Step(`^The error message contains "([^"]*)"$`, f.ingressCreationrrorMessageContains)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`, f.ingressStatusIPOrFQDN)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, f.headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, f.sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, f.responseStatusCodeIs)
	s.Step(`^Header "([^"]*)" is "([^"]*)"$`, f.headerIs)
	s.Step(`^an Ingress is created with host "([^"]*)" with an invalid backend$`, f.anIngressIsCreatedWithFoobarHostWithInvalidBackend)
	s.Step(`^Send HTTP request with <path> and <method> checking response status code is (\d+):$`, f.sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs)
	s.Step(`^With path "([^"]*)"$`, f.withPath)

	s.BeforeScenario(func(this interface{}) {
		f.state = state.New()
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		utils.DeleteKubeNamespace(c, f.state.Namespace)
	})
}
