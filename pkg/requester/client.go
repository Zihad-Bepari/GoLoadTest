package requester

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/goloadtest/goloadtest/pkg/types"
)

// ResponseRecord holds the outcome of a single request execution.
type ResponseRecord struct {
	EndpointName string
	URL          string
	Method       string
	StatusCode   int
	Duration     time.Duration
	Timestamp    time.Time
	Success      bool
	ServerDown   bool // true on 5xx server crash or network drop/refusal
	ErrorMessage string
	BytesRead    int64
}

// Client wraps an optimized http.Client for high-concurrency benchmarking.
type Client struct {
	httpClient *http.Client
}

// NewClient creates an optimized HTTP client tuned for high-throughput load generation.
//
// Security: allowInsecureTLS should only be true for internal/self-signed servers in
// controlled test environments. It MUST remain false for any production HTTPS endpoints.
func NewClient(timeout time.Duration, maxIdleConns int, allowInsecureTLS bool) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxIdleConns <= 0 {
		maxIdleConns = 1000
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          maxIdleConns,
		MaxIdleConnsPerHost:   maxIdleConns,
		MaxConnsPerHost:       0, // unlimited
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			// #nosec G402 - InsecureSkipVerify is opt-in via allowInsecureTLS config flag.
			// It is false by default and should only be enabled for internal test environments.
			InsecureSkipVerify: allowInsecureTLS, //nolint:gosec
		},
	}

	return &Client{
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// Execute sends an HTTP request according to the endpoint specification.
func (c *Client) Execute(ctx context.Context, spec types.EndpointSpec) ResponseRecord {
	startTime := time.Now()
	record := ResponseRecord{
		EndpointName: spec.Name,
		URL:          spec.URL,
		Method:       spec.Method,
		Timestamp:    startTime,
	}

	reqTimeout := spec.Timeout
	if reqTimeout <= 0 {
		reqTimeout = 10 * time.Second
	}

	reqCtx, cancel := context.WithTimeout(ctx, reqTimeout)
	defer cancel()

	var bodyReader io.Reader
	if spec.Body != "" {
		bodyReader = strings.NewReader(spec.Body)
	}

	reqMethod := strings.ToUpper(strings.TrimSpace(spec.Method))
	if reqMethod == "" {
		reqMethod = "GET"
	}

	req, err := http.NewRequestWithContext(reqCtx, reqMethod, spec.URL, bodyReader)
	if err != nil {
		record.Duration = time.Since(startTime)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("invalid request: %v", err)
		return record
	}

	// Apply Headers
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "GoLoadTest/1.0 (Cross-Language Load Tester)")
	}
	if spec.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Apply Authentication
	if spec.Auth != nil {
		c.applyAuth(req, spec.Auth)
	}

	resp, err := c.httpClient.Do(req)
	record.Duration = time.Since(startTime)

	if err != nil {
		record.Success = false
		record.ServerDown = true
		record.ErrorMessage = categorizeError(err)
		return record
	}
	defer resp.Body.Close()

	record.StatusCode = resp.StatusCode

	// Discard body while counting bytes read to reuse connection
	n, _ := io.Copy(io.Discard, resp.Body)
	record.BytesRead = n

	// Validate expected status code
	if len(spec.ExpectedStatus) > 0 {
		matched := false
		for _, expected := range spec.ExpectedStatus {
			if resp.StatusCode == expected {
				matched = true
				break
			}
		}
		record.Success = matched
		if resp.StatusCode >= 500 {
			record.ServerDown = true
		}
		if !matched {
			record.ErrorMessage = fmt.Sprintf("unexpected status: %d", resp.StatusCode)
		}
	} else {
		// Default rule:
		// 2xx & 3xx: Success
		// 4xx (400, 401, 403, 404): Handled by server (e.g. protected endpoints). Server is healthy!
		// 5xx (500, 502, 503, 504): Server crash / breakdown
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			record.Success = true
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			record.Success = true
			record.ServerDown = false
			record.ErrorMessage = fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		} else {
			record.Success = false
			record.ServerDown = true
			record.ErrorMessage = fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
		}
	}

	return record
}

func (c *Client) applyAuth(req *http.Request, auth *types.AuthConfig) {
	switch strings.ToLower(auth.Type) {
	case "bearer":
		token := auth.Token
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		req.Header.Set("Authorization", token)
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "apikey":
		if strings.ToLower(auth.In) == "query" {
			q := req.URL.Query()
			q.Set(auth.Key, auth.Value)
			req.URL.RawQuery = q.Encode()
		} else {
			req.Header.Set(auth.Key, auth.Value)
		}
	case "custom":
		for k, v := range auth.Headers {
			req.Header.Set(k, v)
		}
	}
}

// categorizeError groups network and system errors into actionable clean messages.
func categorizeError(err error) string {
	if err == nil {
		return ""
	}

	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "context canceled") {
		return "context_canceled"
	}
	if strings.Contains(errStr, "connection refused") {
		return "connection_refused"
	}
	if strings.Contains(errStr, "deadline exceeded") {
		return "request_timeout"
	}

	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return "request_timeout"
	}

	if urlErr, ok := err.(*url.Error); ok {
		if urlErr.Timeout() {
			return "request_timeout"
		}
		if opErr, ok := urlErr.Err.(*net.OpError); ok {
			return fmt.Sprintf("net_error: %v", opErr.Err)
		}
		return fmt.Sprintf("url_error: %v", urlErr.Err)
	}

	return err.Error()
}
