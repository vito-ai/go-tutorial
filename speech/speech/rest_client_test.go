package speech

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock RoundTripper for HTTP client to mock requests and responses
type MockRoundTripper struct {
	roundTripFunc func(req *http.Request) *http.Response
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req), nil
}

// Test case when NewRestClient fails due to missing environment variables
func TestNewRestClientEnvError(t *testing.T) {
	// Clear environment variables for the test
	os.Unsetenv("RTZR_CLIENT_ID")
	os.Unsetenv("RTZR_CLIENT_SECRET")

	_, err := NewRestClient()
	if err == nil {
		t.Fatal("Expected error due to missing environment variables, but got nil")
	}
}

// Test NewRestClient success case
func TestNewRestClientSuccess(t *testing.T) {
	// Set required environment variables
	os.Setenv("RTZR_CLIENT_ID", "test-client-id")
	os.Setenv("RTZR_CLIENT_SECRET", "test-client-secret")

	// Create the restClient
	client, err := NewRestClient()

	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.httpClient)

	// Clean up the environment variables
	os.Unsetenv("RTZR_CLIENT_ID")
	os.Unsetenv("RTZR_CLIENT_SECRET")
}

// Test Recognize function with a completed response
func TestRecognizeSuccess(t *testing.T) {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{
			roundTripFunc: func(req *http.Request) *http.Response {
				// 첫 번째 요청: 파일 업로드 (async)
				if strings.Contains(req.URL.String(), "/transcribe") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"id": "12345"}`)),
					}
				}
				// 두 번째 요청: 결과 가져오기 (completed)
				if strings.Contains(req.URL.String(), "12345") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "completed"}`)),
					}
				}
				return nil
			},
		},
	}

	// Mock된 클라이언트 생성
	c := &restClient{
		httpClient: mockClient,
		endpoint:   "https://mock-endpoint/transcribe",
	}

	// RecognizeRequest에 대한 모킹된 데이터 설정
	request := &RecognizeRequest{
		Config: RecognitionConfig{
			ModelName: "sommers",
		},
		AudioSource: RecognitionAudio{
			FilePath: "../go_rtzr_api/output.wav",
		},
	}

	// Recognize 호출
	response, err := c.Recognize(context.Background(), request)

	// 결과 및 에러 체크
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "completed", response.Status)
}

// Test RecognizeAsync error case when file upload fails
func TestRecognizeAsyncFileError(t *testing.T) {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{
			roundTripFunc: func(req *http.Request) *http.Response {
				// Return a failed response
				return &http.Response{
					StatusCode: 500,
					Body:       io.NopCloser(strings.NewReader("internal server error")),
				}
			},
		},
	}

	c := &restClient{
		httpClient: mockClient,
		endpoint:   "https://mock-endpoint",
	}

	// Mock recognize request
	request := &RecognizeRequest{
		AudioSource: RecognitionAudio{
			FilePath: "invalid_path.wav",
		},
		Config: RecognitionConfig{},
	}

	_, err := c.RecognizeAsync(context.Background(), request)
	assert.Error(t, err)
}

// Test Recognize polling with NotFinishErr error
func TestReceiveResultWithPollingNotFinished(t *testing.T) {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{
			roundTripFunc: func(req *http.Request) *http.Response {
				if strings.Contains(req.URL.String(), "/transcribe") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"id": "12345"}`)),
					}
				} else if strings.Contains(req.URL.String(), "12345") {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "not_completed"}`)),
					}
				}
				return nil
			},
		},
	}

	c := &restClient{
		httpClient: mockClient,
		endpoint:   "https://mock-endpoint",
	}

	// Test polling
	_, err := c.receiveResultWithPolling(context.Background(), "12345", 1*time.Second)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFinish, err)
}

// Test receiveResult when result is completed
func TestReceiveResultSuccess(t *testing.T) {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{
			roundTripFunc: func(req *http.Request) *http.Response {
				// Mock successful result retrieval
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"status": "completed"}`)),
				}
			},
		},
	}

	c := &restClient{
		httpClient: mockClient,
		endpoint:   "https://mock-endpoint",
	}

	response, err := c.ReceiveResult(context.Background(), "12345")
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "completed", response.Status)
}

// Test receiveResult when result is not completed (not finished)
func TestReceiveResultNotFinished(t *testing.T) {
	mockClient := &http.Client{
		Transport: &MockRoundTripper{
			roundTripFunc: func(req *http.Request) *http.Response {
				// Mock not finished response
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader(`{"status": "not_completed"}`)),
				}
			},
		},
	}

	c := &restClient{
		httpClient: mockClient,
		endpoint:   "https://mock-endpoint",
	}

	_, err := c.ReceiveResult(context.Background(), "12345")
	assert.Error(t, err)
	assert.Equal(t, ErrNotFinish, err)
}
func TestRecognizeAsync_MissingConfigField(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "missing config field"}`))
	}))
	defer mockServer.Close()

	client, err := NewRestClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.endpoint = mockServer.URL

	req := &RecognizeRequest{
		// Missing or incomplete config fields
		AudioSource: RecognitionAudio{
			FilePath: "test.wav",
		},
	}

	_, err = client.RecognizeAsync(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected an error due to missing config field, got nil")
	}
}

func TestRecognizeAsync_InvalidConfigField(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid config field"}`))
	}))
	defer mockServer.Close()

	client, err := NewRestClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.endpoint = mockServer.URL

	req := &RecognizeRequest{
		Config: RecognitionConfig{
			ModelName: "invalid-model", // assuming this is invalid
		},
		AudioSource: RecognitionAudio{
			FilePath: "test.wav",
		},
	}

	_, err = client.RecognizeAsync(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected an error due to invalid config field, got nil")
	}
}

func TestCreateFileField_InvalidFilePath(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	client, err := NewRestClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.endpoint = mockServer.URL

	req := &RecognizeRequest{
		Config: RecognitionConfig{
			ModelName: "sommers",
		},
		AudioSource: RecognitionAudio{
			FilePath: "invalid_path.wav", // Invalid file path
		},
	}

	_, err = client.RecognizeAsync(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected an error due to invalid file path, got nil")
	}
}

func TestCreateFileField_FileOpenError(t *testing.T) {
	// oldOpen := os.Open
	// defer func() { os.Open = oldOpen }()
	// os.Open = func(name string) (*os.File, error) {
	// 	return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
	// }

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer mockServer.Close()

	client, err := NewRestClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.endpoint = mockServer.URL

	req := &RecognizeRequest{
		Config: RecognitionConfig{
			ModelName: "sommers",
		},
		AudioSource: RecognitionAudio{
			FilePath: "test.wav",
		},
	}

	_, err = client.RecognizeAsync(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected an error due to file open failure, got nil")
	}
}

func TestRecognizeAsync_InvalidServerResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"unexpected_field": "value"}`)) // Invalid response structure
	}))
	defer mockServer.Close()

	client, err := NewRestClient()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	client.endpoint = mockServer.URL

	req := &RecognizeRequest{
		Config: RecognitionConfig{
			ModelName: "sommers",
		},
		AudioSource: RecognitionAudio{
			FilePath: "test.wav",
		},
	}

	_, err = client.RecognizeAsync(context.Background(), req)
	if err == nil {
		t.Fatalf("Expected an error due to invalid server response, got nil")
	}
}
