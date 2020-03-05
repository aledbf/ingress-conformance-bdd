package defaultbackend

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"
	v1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type feature struct {
	kubeClient *kubernetes.Clientset

	state *tstate.Scenario
}

const (
	minimumRowCount = 1
	httpPort        = 80
)

func (f *feature) aNewRandomNamespace() error {
	var err error

	f.state.Namespace, err = utils.CreateTestNamespace(f.kubeClient)
	if err != nil {
		return err
	}

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

	f.state.Ingress = ingSpec

	return nil
}

func (f *feature) ingressCreationrrorMessageContains(expected string) error {
	_, err := utils.CreateIngress(f.kubeClient, f.state.Ingress)
	if err == nil {
		return fmt.Errorf("expected an error creating an ingress without backend serviceName")
	}

	if strings.Contains(err.Error(), expected) {
		return nil
	}

	return fmt.Errorf("expected an error containing %v but returned %v", expected, err.Error())
}

func (f *feature) ingressStatusIPOrFQDN() error {
	if f.state.Ingress == nil {
		return fmt.Errorf("feature without Ingress associated")
	}

	address, err := utils.WaitForIngressAddress(f.kubeClient, f.state.Namespace,
		f.state.Ingress.GetName(), "", utils.WaitForIngressAddressTimeout)
	if err != nil {
		return err
	}

	f.state.Address = address

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
										ServicePort: intstr.FromInt(httpPort),
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

	ingSpec, err = utils.CreateIngress(f.kubeClient, ingSpec)
	if err != nil {
		return err
	}

	f.state.Ingress = ingSpec

	return nil
}

func (f *feature) sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs(statusCode int,
	testTable *gherkin.DataTable) error {
	if len(testTable.Rows) < minimumRowCount {
		return fmt.Errorf("expected a table with at least one row")
	}

	for i, row := range testTable.Rows {
		if i == 0 {
			continue
		}

		path := row.Cells[0].Value
		method := row.Cells[1].Value

		req, err := http.NewRequest(method, fmt.Sprintf("http://%v%v", f.state.Address, path), nil)
		if err != nil {
			return err
		}

		err = f.state.SendRequest(req)
		if err != nil {
			return err
		}

		if statusCode != f.state.StatusCode {
			return fmt.Errorf("expected status code %v for path %v and method %v but %v was returned",
				statusCode, path, method, f.state.StatusCode)
		}
	}

	return nil
}

func (f *feature) withPath(arg1 string) error {
	return godog.ErrPending
}

// FeatureContext adds steps to setup and verify tests
func FeatureContext(s *godog.Suite, c *kubernetes.Clientset) {
	f := &feature{
		kubeClient: c,
	}

	s.Step(`^a new random namespace$`, f.aNewRandomNamespace)
	s.Step(`^creating an Ingress with host "([^"]*)" without backend serviceName$`,
		f.anIngressIsCreatedWithHostAndNoBackend)
	s.Step(`^The error message contains "([^"]*)"$`, f.ingressCreationrrorMessageContains)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`,
		f.ingressStatusIPOrFQDN)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, f.headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, f.sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, f.responseStatusCodeIs)
	s.Step(`^Header "([^"]*)" is "([^"]*)"$`, f.headerIs)
	s.Step(`^an Ingress is created with host "([^"]*)" with an invalid backend$`,
		f.anIngressIsCreatedWithFoobarHostWithInvalidBackend)
	s.Step(`^Send HTTP request with <path> and <method> checking response status code is (\d+):$`,
		f.sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs)
	s.Step(`^With path "([^"]*)"$`, f.withPath)

	s.BeforeScenario(func(this interface{}) {
		f.state = tstate.New(nil)
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(c, f.state.Namespace)
	})
}
