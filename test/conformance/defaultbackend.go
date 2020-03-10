package conformance

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"
	"k8s.io/client-go/kubernetes"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type defaultbackend struct {
	kubeClient *kubernetes.Clientset

	state *tstate.Scenario
}

const (
	minimumRowCount = 1
)

func (f *defaultbackend) aNewRandomNamespace() error {
	var err error

	f.state.Namespace, err = utils.CreateTestNamespace(f.kubeClient)
	if err != nil {
		return err
	}

	return nil
}

func (f *defaultbackend) readingIngressManifest(file string) error {
	f.state.IngressManifest = file

	ing, err := utils.IngressFromManifest(file, f.state.Namespace)
	if err != nil {
		return err
	}

	f.state.Ingress = ing

	return nil
}

func (f *defaultbackend) creatingIngressFromManifest() error {
	_, err := utils.CreateIngress(f.kubeClient, f.state.Ingress)
	return err
}

func (f *defaultbackend) newIngressFromManifestWithError(expected string) error {
	_, err := utils.CreateIngress(f.kubeClient, f.state.Ingress)
	if err == nil {
		return fmt.Errorf("expected an error creating an ingress without backend serviceName")
	}

	if strings.Contains(err.Error(), expected) {
		return nil
	}

	return fmt.Errorf("expected an error containing %v but returned %v", expected, err.Error())
}

func (f *defaultbackend) ingressStatusIPOrFQDN() error {
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

func (f *defaultbackend) headerWithValue(arg1, arg2 string) error {
	return nil
}

func (f *defaultbackend) sendHTTPRequestWithMethod(arg1 string) error {
	return nil
}

func (f *defaultbackend) responseStatusCodeIs(arg1 int) error {
	return nil
}

func (f *defaultbackend) headerIs(arg1, arg2 string) error {
	return nil
}

func (f *defaultbackend) sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs(statusCode int,
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

func (f *defaultbackend) withPath(arg1 string) error {
	return godog.ErrPending
}

// DefaultBackendContext adds steps to setup and verify tests
func DefaultBackendContext(s *godog.Suite, c *kubernetes.Clientset) {
	f := &defaultbackend{
		kubeClient: c,
	}

	s.Step(`^a new random namespace$`, f.aNewRandomNamespace)
	s.Step(`^reading Ingress from manifest "([^"]*)"$`, f.readingIngressManifest)
	s.Step(`^creating Ingress from manifest returns an erro message containing "([^"]*)"$`, f.newIngressFromManifestWithError)
	s.Step(`^creating Ingress from manifest$`, f.creatingIngressFromManifest)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`,
		f.ingressStatusIPOrFQDN)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, f.headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, f.sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, f.responseStatusCodeIs)
	s.Step(`^Header "([^"]*)" is "([^"]*)"$`, f.headerIs)
	//s.Step(`^an Ingress is created with host "([^"]*)" with an invalid backend$`,
	//	f.anIngressIsCreatedWithFoobarHostWithInvalidBackend)
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
