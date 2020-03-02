package state

import (
	"net/http"

	v1beta1 "k8s.io/api/networking/v1beta1"
)

type HTTP struct {
	Client *http.Client

	URL         string
	RequestBody interface{}

	Response *http.Response
}

type Feature struct {
	HTTP *HTTP

	Namespace string

	objectReference *v1beta1.Ingress
	address         string
}

func New() *Feature {
	return &Feature{
		HTTP: &HTTP{
			Client: &http.Client{},
		},
	}
}

func (f *Feature) SetIngress(obj *v1beta1.Ingress) {
	f.objectReference = obj
}

func (f *Feature) GetIngress() *v1beta1.Ingress {
	return f.objectReference
}

func (f *Feature) SetStatusAddress(address string) {
	f.address = address
}

func (f *Feature) GetStatusAddress() string {
	return f.address
}
