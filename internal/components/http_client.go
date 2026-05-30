package components

import (
	"context"
	"net/http"

	"resty.dev/v3"
)

type HTTPClient struct {
	restClient *resty.Client
}

func NewHTTPClient(config *Config) *HTTPClient {
	brokerURL := config.BrokerURL

	if brokerURL == "" {
		brokerURL = "http://localhost:8080"
	}

	httpClient := &HTTPClient{
		restClient: resty.New().SetBaseURL(brokerURL),
	}

	httpClient.
		restClient.
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	return httpClient
}

// StdClient returns resty's underlying *http.Client so tests can attach
// httpmock.ActivateNonDefault to intercept requests (resty does not use
// http.DefaultTransport).
func (c *HTTPClient) StdClient() *http.Client {
	return c.restClient.Client()
}

// SetBaseURL overrides the broker base URL after construction, so the
// --broker-url flag (parsed after the client is built) can take effect.
func (c *HTTPClient) SetBaseURL(url string) {
	c.restClient.SetBaseURL(url)
}

func (c *HTTPClient) Get(ctx context.Context, url string) (*resty.Response, error) {
	return c.restClient.R().Get(url)
}

func (c *HTTPClient) Post(ctx context.Context, url string, body any) (*resty.Response, error) {
	return c.restClient.R().SetBody(body).Post(url)
}

func (c *HTTPClient) Put(ctx context.Context, url string, body any) (*resty.Response, error) {
	return c.restClient.R().SetBody(body).Put(url)
}

func (c *HTTPClient) Delete(ctx context.Context, url string) (*resty.Response, error) {
	return c.restClient.R().Delete(url)
}
