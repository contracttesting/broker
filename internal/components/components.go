package components

type Components struct {
	Config     *Config
	HTTPClient *HTTPClient
}

func New() *Components {
	config := NewConfig()
	return &Components{
		HTTPClient: NewHTTPClient(config),
		Config:     config,
	}
}
