package conformance

import (
	"fmt"

	"github.com/aledbf/ingress-conformance-bdd/test/utils"
)

func ingressStatusIPOrFQDN() error {
	if state.Ingress == nil {
		return fmt.Errorf("feature without Ingress associated")
	}

	address, err := utils.WaitForIngressAddress(KubeClient, state.Namespace,
		state.Ingress.GetName(), "", utils.WaitForIngressAddressTimeout)
	if err != nil {
		return err
	}

	state.Address = address

	return nil
}

func aNewRandomNamespace() error {
	var err error

	state.Namespace, err = utils.CreateTestNamespace(KubeClient)
	if err != nil {
		return err
	}

	return nil
}

func responseStatusCodeIs(code int) error {
	if state.StatusCode != code {
		return fmt.Errorf("expected status code %v but %v was returned",
			code, state.StatusCode)
	}

	return nil
}
