package llm

type GigachatDriver struct {
	token string
}

func NewGigachatDriver(token string) *GigachatDriver {
	return &GigachatDriver{token: token}
}

func (d *GigachatDriver) SendRequest(prompt string) (string, error) {
	return "fallback answer", nil
}
