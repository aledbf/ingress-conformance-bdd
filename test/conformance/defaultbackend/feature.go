package defaultbackend

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/gherkin"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

var (
	// holds state of the scenarario
	state *tstate.Scenario
)

func aNewRandomNamespace() error {
	var err error

	state.Namespace, err = utils.CreateTestNamespace(utils.KubeClient)
	if err != nil {
		return err
	}

	return nil
}

func theIngressStatusShowsTheIPAddressOrFQDNWhereIsExposed() error {
	if state.Ingress == nil {
		return fmt.Errorf("feature without Ingress associated")
	}

	address, err := utils.WaitForIngressAddress(utils.KubeClient, state.Namespace,
		state.Ingress.GetName(), utils.WaitForIngressAddressTimeout)
	if err != nil {
		return err
	}

	state.Address = address

	return nil
}

func responseStatusCodeIs(code int) error {
	if state.StatusCode != code {
		return fmt.Errorf("expected status code %v but %v was returned",
			code, state.StatusCode)
	}

	return nil
}

func creatingObjectsFromDirectory(path string) error {
	var err error

	state.Ingress, err = utils.CreateFromPath(utils.KubeClient, path, state.Namespace, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func readingIngressFromManifest(file string) error {
	var err error

	state.Ingress, err = utils.IngressFromManifest(file, state.Namespace)
	if err != nil {
		return err
	}

	state.IngressManifest = file

	return nil
}

func creatingIngressFromManifestReturnsAnErrorMessageContaining(arg1 string) error {
	_, err := utils.CreateIngress(utils.KubeClient, state.Ingress)
	if err == nil {
		return fmt.Errorf("expected an error creating an ingress without backend serviceName")
	}

	if strings.Contains(err.Error(), arg1) {
		return nil
	}

	return fmt.Errorf("expected an error containing %v but returned %v", arg1, err.Error())
}

func creatingIngressFromManifest() error {
	_, err := utils.CreateIngress(utils.KubeClient, state.Ingress)
	return err
}

func headerWithValue(header, value string) error {
	state.AddRequestHeader(header, value)
	return nil
}

func sendHTTPRequestWithMethod(arg1 string) error {
	req, err := http.NewRequest(arg1, fmt.Sprintf("http://%v", state.Address), nil)
	if err != nil {
		return err
	}

	err = state.SendRequest(req)
	if err != nil {
		return err
	}

	return nil
}

func sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs(arg1 int, arg2 *gherkin.DataTable) error {
	if len(arg2.Rows) < 1 {
		return fmt.Errorf("expected a table with at least one row")
	}

	for i, row := range arg2.Rows {
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

		if arg1 != state.StatusCode {
			return fmt.Errorf("expected status code %v for path %v and method %v but %v was returned",
				arg1, path, method, state.StatusCode)
		}
	}

	return nil
}

func withPath(arg1 string) error {
	state.RequestPath = arg1

	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^a new random namespace$`, aNewRandomNamespace)
	s.Step(`^reading Ingress from manifest "([^"]*)"$`, readingIngressFromManifest)
	s.Step(`^creating Ingress from manifest returns an error message containing "([^"]*)"$`, creatingIngressFromManifestReturnsAnErrorMessageContaining)
	s.Step(`^creating Ingress from manifest$`, creatingIngressFromManifest)
	s.Step(`^The ingress status shows the IP address or FQDN where is exposed$`, theIngressStatusShowsTheIPAddressOrFQDNWhereIsExposed)
	s.Step(`^Header "([^"]*)" with value "([^"]*)"$`, headerWithValue)
	s.Step(`^Send HTTP request with method "([^"]*)"$`, sendHTTPRequestWithMethod)
	s.Step(`^Response status code is (\d+)$`, responseStatusCodeIs)
	s.Step(`^Send HTTP request with <path> and <method> checking response status code is (\d+):$`, sendHTTPRequestWithPathAndMethodCheckingResponseStatusCodeIs)
	s.Step(`^creating objects from directory "([^"]*)"$`, creatingObjectsFromDirectory)
	s.Step(`^With path "([^"]*)"$`, withPath)

	//// start generated code
	s.BeforeScenario(func(this interface{}) {
		state = tstate.New(nil)
	})

	s.AfterScenario(func(interface{}, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(utils.KubeClient, state.Namespace)
	})
	//// end generated code
}
