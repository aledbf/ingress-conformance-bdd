package state

import (
	"io/ioutil"
	"net/http"

	v1beta1 "k8s.io/api/networking/v1beta1"
)

// Scenario holds state for a test scenario
type Scenario struct {
	client *http.Client

	ResponseBody    []byte
	ResponseHeaders http.Header

	StatusCode int

	Namespace string

	Ingress *v1beta1.Ingress
	Address string
}

// New creates a new state to use in a test Scenario
func New(client *http.Client) *Scenario {
	if client == nil {
		client = &http.Client{}
	}

	return &Scenario{
		client: client,
	}
}

// SendRequest sends an HTTP request and updates the
// state. In case of an error, the HTTP state is
// removed and returns an error.
func (f *Scenario) SendRequest(req *http.Request) error {
	resp, err := f.client.Do(req)
	if err != nil {
		f.ResponseBody = nil
		f.StatusCode = 0
		f.ResponseHeaders = nil

		return err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	f.ResponseBody = bodyBytes
	f.ResponseHeaders = resp.Header.Clone()
	f.StatusCode = resp.StatusCode

	defer resp.Body.Close()

	return nil
}
