package withouthost

//// start generated code
import (
	"fmt"
	"net/http"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"

	tstate "github.com/aledbf/ingress-conformance-bdd/test/state"
	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

var (
	// holds state of the scenarario
	state *tstate.Scenario
)

//// end generated code

func aNewRandomNamespace() error {
	var err error

	state.Namespace, err = utils.CreateTestNamespace(utils.KubeClient)
	if err != nil {
		return err
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

func sendGETHTTPRequest() error {
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

func theHTTPResponseCodeIs(arg1 int) error {
	if state.StatusCode != arg1 {
		return fmt.Errorf("expected status code %v but %v was returned",
			arg1, state.StatusCode)
	}

	return nil
}

func headerIsNotPresent(arg1 string) error {
	if value, ok := state.ResponseHeaders[arg1]; ok {
		return fmt.Errorf("expected no header with name %v but exists (value %v)", arg1, value)
	}

	return nil
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^a new random namespace$`, aNewRandomNamespace)
	s.Step(`^creating objects from directory "([^"]*)"$`, creatingObjectsFromDirectory)
	s.Step(`^the ingress status shows the IP address or FQDN where is exposed$`, theIngressStatusShowsTheIPAddressOrFQDNWhereIsExposed)
	s.Step(`^send GET HTTP request$`, sendGETHTTPRequest)
	s.Step(`^the HTTP response code is (\d+)$`, theHTTPResponseCodeIs)
	s.Step(`^Header "([^"]*)" is not present$`, headerIsNotPresent)

	//// start generated code
	s.BeforeScenario(func(this *messages.Pickle) {
		state = tstate.New(nil)
	})

	s.AfterScenario(func(*messages.Pickle, error) {
		// delete namespace an all the content
		_ = utils.DeleteKubeNamespace(utils.KubeClient, state.Namespace)
	})
	//// end generated code
}
