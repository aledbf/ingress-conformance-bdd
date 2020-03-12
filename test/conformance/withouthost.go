package conformance

import (
	"fmt"
	"net/http"

	"github.com/cucumber/godog"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type withoutHost struct{}

func creatingObjectsFromDirectory(path string) error {
	ing, err := utils.CreateFromPath(KubeClient, path, state.Namespace, nil, nil)
	if err != nil {
		return err
	}

	state.Ingress = ing
	return nil
}

func (f *withoutHost) sendGETHTTPRequest() error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%v", state.Address), nil)
	if err != nil {
		return err
	}

	err = state.SendRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (f *withoutHost) headerIsNotPresent(header string) error {
	if value, ok := state.ResponseHeaders[header]; ok {
		return fmt.Errorf("expected no header with name %v but exists (value %v)", header, value)
	}

	return nil
}

// WithoutHostContext adds steps to setup and verify tests
func WithoutHostContext(s *godog.Suite) {
	f := &withoutHost{}

	s.Step(`^a new random namespace$`, aNewRandomNamespace)
	s.Step(`^the ingress status shows the IP address or FQDN where is exposed$`,
		ingressStatusIPOrFQDN)
	s.Step(`^send GET HTTP request$`, f.sendGETHTTPRequest)
	s.Step(`^Header "([^"]*)" is not present$`, f.headerIsNotPresent)
	s.Step(`^creating objects from directory "([^"]*)"$`, creatingObjectsFromDirectory)
	s.Step(`^the HTTP response code is (\d+)$`, responseStatusCodeIs)

	s.BeforeScenario(func(this interface{}) {
		state = tstate.New(nil)
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(KubeClient, state.Namespace)
	})
}
