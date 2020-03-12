package conformance

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

type defaultbackend struct{}

const (
	minimumRowCount = 1
)

func (f *defaultbackend) readingIngressManifest(file string) error {
	var err error

	state.Ingress, err = utils.IngressFromManifest(file, state.Namespace)
	if err != nil {
		return err
	}

	state.IngressManifest = file

	return nil
}

func (f *defaultbackend) creatingIngressFromManifest() error {
	_, err := utils.CreateIngress(KubeClient, state.Ingress)
	return err
}

func (f *defaultbackend) newIngressFromManifestWithError(expected string) error {
	_, err := utils.CreateIngress(KubeClient, state.Ingress)
	if err == nil {
		return fmt.Errorf("expected an error creating an ingress without backend serviceName")
	}

	if strings.Contains(err.Error(), expected) {
		return nil
	}

	return fmt.Errorf("expected an error containing %v but returned %v", expected, err.Error())
}

func (f *defaultbackend) headerWithValue(header, value string) error {
	state.AddRequestHeader(header, value)
	return nil
}

func (f *defaultbackend) sendHTTPRequestWithMethod(method string) error {
	req, err := http.NewRequest(method, fmt.Sprintf("http://%v", state.Address), nil)
	if err != nil {
		return err
	}

	err = state.SendRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func (f *defaultbackend) headerIs(header, value string) error {
	lheader := strings.ToLower(header)
	rvalue, ok := state.ResponseHeaders[lheader]
	if !ok {
		return fmt.Errorf("expected response containing header %v", lheader)
	}

	if len(rvalue) > 1 {
		return fmt.Errorf("header %v contains more than one value", lheader)
	}

	if value != rvalue[0] {
		return fmt.Errorf("unexpected value for header %v (expected %v but %v was returned)", header, value, rvalue)
	}

	return nil
}

func (f *defaultbackend) requestsWithPathAndMethod(statusCode int, testTable *gherkin.DataTable) error {
	if len(testTable.Rows) < minimumRowCount {
		return fmt.Errorf("expected a table with at least one row")
	}

	for i, row := range testTable.Rows {
		if i == 0 {
			continue
		}

		path := row.Cells[0].Value
		method := row.Cells[1].Value

		req, err := http.NewRequest(method,
			fmt.Sprintf("http://%v%v", state.Address, state.RequestPath), nil)
		if err != nil {
			return err
		}

		err = state.SendRequest(req)
		if err != nil {
			return err
		}

		if statusCode != state.StatusCode {
			return fmt.Errorf("expected status code %v for path %v and method %v but %v was returned",
				statusCode, path, method, state.StatusCode)
		}
	}

	return nil
}

func (f *defaultbackend) withPath(path string) error {
	state.RequestPath = path

	return nil
}

// DefaultBackendContext adds steps to setup and verify tests
func DefaultBackendContext(s *godog.Suite) {
	f := &defaultbackend{}

	s.Step(`^a new random namespace$`, aNewRandomNamespace)
	s.Step(`^reading Ingress from manifest "([^"]*)"$`, f.readingIngressManifest)
	s.Step(`^creating Ingress from manifest returns an error message containing "([^"]*)"$`,
		f.newIngressFromManifestWithError)
	s.Step(`^creating Ingress from manifest$`, f.creatingIngressFromManifest)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`,
		ingressStatusIPOrFQDN)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, f.headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, f.sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, responseStatusCodeIs)
	s.Step(`^Header "([^"]*)" is "([^"]*)"$`, f.headerIs)
	s.Step(`^Send HTTP request with <path> and <method> checking response status code is (\d+):$`,
		f.requestsWithPathAndMethod)
	s.Step(`^With path "([^"]*)"$`, f.withPath)

	s.BeforeScenario(func(this interface{}) {
		state = tstate.New(nil)
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(KubeClient, state.Namespace)
	})
}
