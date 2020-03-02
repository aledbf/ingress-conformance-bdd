package defaultbackend

import (
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type feature struct {
	state  *utils.State
	client *clientset.Clientset

	namespace string

	ingressIPOrFQDN string
}

func (f *feature) aNewRandomNamespace() error {
	ns, err := utils.CreateTestNamespace(f.client)
	if err != nil {
		return err
	}

	f.namespace = ns
	return nil
}

func (f *feature) ingressStatusIPOrFQDN() error {
	f.ingressIPOrFQDN = "1.1.1.1"
	return nil
}

func (f *feature) anIngressIsCreatedWithHostAndNoBackend(arg1 string) error {
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

func (f *feature) anIngressIsCreatedWithFoobarHostWithInvalidBackend() error {
	return nil
}

func (f *feature) anIngressIsCreatedWithHostWithInvalidBackend(arg1 string) error {
	return godog.ErrPending
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
	s.Step(`^an Ingress is created with host "([^"]*)" and no backend$`, f.anIngressIsCreatedWithHostAndNoBackend)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`, f.ingressStatusIPOrFQDN)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, f.headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, f.sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, f.responseStatusCodeIs)
	s.Step(`^Header "([^"]*)" is "([^"]*)"$`, f.headerIs)
	s.Step(`^an Ingress is created with foo\.bar host with invalid backend$`, f.anIngressIsCreatedWithFoobarHostWithInvalidBackend)
	s.Step(`^an Ingress is created with host "([^"]*)" with invalid backend$`, f.anIngressIsCreatedWithHostWithInvalidBackend)
	s.Step(`^Send HTTP request with <path> and <method> checking response status code is (\d+):$`, f.sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs)
	s.Step(`^With path "([^"]*)"$`, f.withPath)

	s.BeforeScenario(func(this interface{}) {
		switch this.(type) {
		case *gherkin.Scenario:
			f.state = utils.NewState()
		}
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		utils.DeleteKubeNamespace(c, f.namespace)
		f.namespace = ""
	})
}
