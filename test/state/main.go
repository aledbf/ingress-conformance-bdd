package state

import (
	"io/ioutil"
	"net/http"

	v1beta1 "k8s.io/api/networking/v1beta1"
)

type Feature struct {
	client *http.Client

	ResponseBody    []byte
	ResponseHeaders http.Header

	StatusCode int

	Ingress *v1beta1.Ingress
	Address string
}

func New(client *http.Client) *Feature {
	if client == nil {
		client = &http.Client{}
	}

	return &Feature{
		client: client,
	}
}

func (f *Feature) SendRequest(req *http.Request) error {
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
