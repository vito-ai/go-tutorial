package auth

import (
	"errors"
	"net/http"
)

type OptionsTokenRTZR struct {
	ClientId     string
	ClientSecret string
	TokenURL     string
	Client       *http.Client
}

type Option func(*OptionsTokenRTZR) error

func WithClientId(clientId string) Option {
	return func(ot *OptionsTokenRTZR) error {
		ot.ClientId = clientId
		return nil
	}
}

func WithClientSecret(clientSecret string) Option {
	return func(ot *OptionsTokenRTZR) error {
		ot.ClientSecret = clientSecret
		return nil
	}
}

func WithClientToken(token string) Option {
	return func(ot *OptionsTokenRTZR) error {
		ot.TokenURL = token
		return nil
	}
}

func (o *OptionsTokenRTZR) client() *http.Client {
	if o.Client != nil {
		return o.Client
	}
	return http.DefaultClient
}

func (o *OptionsTokenRTZR) token() string {
	if o.TokenURL != "" {
		return o.TokenURL
	}
	return "https://openapi.vito.ai/v1/authenticate"
}

func (o *OptionsTokenRTZR) validate() error {
	if o == nil {
		return errors.New("auth : options must be provided")
	}
	if o.ClientId == "" {
		return errors.New("auth: RTZR_CLIENT_ID must be provided")
	}
	if o.ClientSecret == "" {
		return errors.New("auth: RTZR_CLIENT_SECRET must be provided")
	}
	if o.TokenURL == "" {
		return errors.New("auth: TokenURL must be provided")
	}
	return nil
}
