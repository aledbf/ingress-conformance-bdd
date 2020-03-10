package conformance

import (
	"github.com/cucumber/godog"
	"k8s.io/client-go/kubernetes"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type withoutHost struct {
	kubeClient *kubernetes.Clientset

	state *tstate.Scenario
}

func (f *withoutHost) aNewRandomNamespace() error {
	var err error

	f.state.Namespace, err = utils.CreateTestNamespace(f.kubeClient)
	if err != nil {
		return err
	}

	return nil
}

func (f *withoutHost) anEchoDeploymentExists() error {
	return godog.ErrPending
}

func (f *withoutHost) anIngressIsCreatedWithoutHostUsingEchoServiceAsBackend() error {
	return godog.ErrPending
}

func (f *withoutHost) theIngressStatusShowsTheIPAddressOrFQDNWhereIsExposed() error {
	return godog.ErrPending
}

func (f *withoutHost) sendGETHTTPRequest() error {
	return godog.ErrPending
}

func (f *withoutHost) iReceiveValidHTPPResponseCode(arg1 int) error {
	return godog.ErrPending
}

func (f *withoutHost) headerIsNotPresent(arg1 string) error {
	return godog.ErrPending
}

// WithoutHostContext adds steps to setup and verify tests
func WithoutHostContext(s *godog.Suite, c *kubernetes.Clientset) {
	f := &withoutHost{
		kubeClient: c,
	}

	s.Step(`^a new random namespace$`, f.aNewRandomNamespace)
	s.Step(`^an echo deployment exists$`, f.anEchoDeploymentExists)
	s.Step(`^an Ingress is created without host using echo service as backend$`,
		f.anIngressIsCreatedWithoutHostUsingEchoServiceAsBackend)
	s.Step(`^the ingress status shows the IP address or FQDN where is exposed$`,
		f.theIngressStatusShowsTheIPAddressOrFQDNWhereIsExposed)
	s.Step(`^send GET HTTP request$`, f.sendGETHTTPRequest)
	s.Step(`^I receive valid HTPP response code (\d+)$`, f.iReceiveValidHTPPResponseCode)
	s.Step(`^Header "([^"]*)" is not present$`, f.headerIsNotPresent)

	s.BeforeScenario(func(this interface{}) {
		f.state = tstate.New(nil)
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(c, f.state.Namespace)
	})
}
