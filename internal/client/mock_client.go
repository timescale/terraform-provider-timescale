package client

import (
	_ "embed"
	"net/http"
	"net/http/httptest"
)

type ClientMock struct {
}

func (c *ClientMock) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{}, nil
}

func NewMockClient(httpMock httptest.Server) *MockClient {
	return &MockClient{
		httpClient: httpMock.Client(),
	}
}

type MockClient = Client
