package utils

import (
	"net/http"
)

type State struct {
	Client *http.Client

	URL         string
	RequestBody interface{}

	Response *http.Response
}

func NewState() *State {
	return &State{
		Client: &http.Client{},
	}
}
