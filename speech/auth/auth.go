package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vito-ai/auth/credentials"
)

// TokenProvider Interface
type TokenProvider interface {
	Token(context.Context) (*ReturnZeroToken, error)
	IsValid() bool
}

// Default Token Provider for RTZR
type tokenProviderRTZR struct {
	token  *ReturnZeroToken
	opts   OptionsTokenRTZR
	Client *http.Client
}

func NewRTZRTokenProvider(opts ...Option) (TokenProvider, error) {
	var tp tokenProviderRTZR

	creds := credentials.GetDefaultClientCreds()
	tp.opts.ClientId = creds.ClientId
	tp.opts.ClientSecret = creds.ClientSecret

	for _, opt := range opts {
		err := opt(&tp.opts)
		if err != nil {
			return nil, err
		}
	}

	tp.opts.TokenURL = tp.opts.token()
	if err := tp.opts.validate(); err != nil {
		return nil, err
	}

	tp.Client = tp.opts.client()
	return &tp, nil
}

func (tp *tokenProviderRTZR) IsValid() bool {
	return tp.token.IsValid()
}

func (tp *tokenProviderRTZR) Token(ctx context.Context) (*ReturnZeroToken, error) {
	if tp.token.IsValid() {
		return tp.token, nil
	}

	formData := url.Values{}
	formData.Set("client_id", tp.opts.ClientId)
	formData.Set("client_secret", tp.opts.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tp.opts.TokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tp.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error when making authentication request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error response from authentication server: %s", resp.Status)
	}

	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	result := &ReturnZeroToken{}
	if err = json.Unmarshal(respByte, result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response body: %w", err)
	}

	if result.AccessToken == "" {
		return nil, fmt.Errorf("unmarshalled response is missing access_token")
	}
	if result.ExpireAt <= time.Now().Unix() {
		return nil, fmt.Errorf("unmarshalled response has invalid expire_at timestamp")
	}

	tp.token = result
	return result, nil
}
