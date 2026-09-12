package definite

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
)

// Client is the Definite Assurance API client. Create one with NewClient and
// reuse it — it is safe for concurrent use.
type Client struct {
	cfg        *Config
	http       *http.Client
	baseURL    string
	authHeader string
	cache      Cache
	refLocks   sync.Map // cache key -> *sync.Mutex; fills a cold cache once, not once per caller
}

// NewClient validates cfg, applies defaults, and returns a ready-to-use Client.
// cfg is copied, so later changes to it have no effect on the client.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, &ClientError{Type: InternalError, Code: ErrInvalidConfig, Operation: "NewClient", Message: "config is required"}
	}
	c := *cfg
	if cfg.Headers != nil {
		c.Headers = make(map[string]string, len(cfg.Headers))
		for k, v := range cfg.Headers {
			c.Headers[k] = v
		}
	}
	if err := c.Validate(); err != nil {
		return nil, newInternalError("NewClient", ErrInvalidConfig, err)
	}

	cache := c.Cache
	if cache == nil {
		switch c.CacheStrategy {
		case CacheNone:
			cache = noopCache{}
		case CacheFile:
			fc, err := NewFileCache(c.CacheDir, c.GetBaseURL(), c.Credentials)
			if err != nil {
				return nil, newInternalError("NewClient", ErrInvalidConfig, fmt.Errorf("file cache: %w", err))
			}
			cache = fc
		default:
			cache = NewMemoryCache()
		}
	}

	hc := c.HTTPClient
	if hc == nil {
		hc = &http.Client{}
	}

	return &Client{
		cfg:        &c,
		http:       hc,
		baseURL:    c.GetBaseURL(),
		authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte(c.Credentials.Username+":"+c.Credentials.Password)),
		cache:      cache,
	}, nil
}

// BaseURL returns the resolved base URL requests are sent to.
func (c *Client) BaseURL() string { return c.baseURL }

// ClearCache drops cached reference data so the next call re-fetches it.
func (c *Client) ClearCache() { c.cache.Clear() }

// request describes one API call.
type request struct {
	op     string // Client method name, used in errors
	method string
	path   string
	query  url.Values
	body   any // JSON-encoded when non-nil
}

// do sends r and returns the body of a 2xx response. Transport failures,
// timeouts and non-2xx statuses come back as *ClientError. Failures the API
// reports inside a 2xx body are left to the decode helpers.
func (c *Client) do(ctx context.Context, r request) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	// A bytes.Reader body gives the request an explicit Content-Length. Without
	// one it would be sent chunked, which the Definite gateway rejects.
	var body io.Reader
	var payload []byte
	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return nil, newInternalError(r.op, ErrMarshalRequest, err)
		}
		payload = b
		body = bytes.NewReader(b)
	}

	u := c.baseURL + r.path
	if len(r.query) > 0 {
		u += "?" + r.query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, r.method, u, body)
	if err != nil {
		return nil, newInternalError(r.op, ErrCreateRequest, err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", c.authHeader) // last, so configured headers can never override it

	c.debugf("%s %s", r.method, u)
	if payload != nil {
		c.debugf("request body: %s", payload)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, transportError(r.op, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, transportError(r.op, ctx.Err())
		}
		return nil, newInternalError(r.op, ErrReadResponse, err)
	}
	c.debugf("response %d: %s", resp.StatusCode, respBody)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newHTTPError(r.op, resp.StatusCode, respBody)
	}
	return respBody, nil
}

// transportError classifies a failure to complete the HTTP exchange. The cause
// stays wrapped, so errors.Is(err, context.Canceled) still works for callers.
func transportError(op string, err error) *ClientError {
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return &ClientError{Type: InternalError, Code: ErrTimeout, Operation: op, Message: "request timed out: " + err.Error(), Err: err}
	}
	return &ClientError{Type: InternalError, Code: ErrNetwork, Operation: op, Message: "request failed: " + err.Error(), Err: err}
}

// decodeEnvelope decodes a { status, message, data, errorCode } response and
// turns an error status into a *ClientError.
//
// The envelope is read before data, so an error response is reported as the
// API's own error even when its data does not match T. An empty body is a
// successful, empty response.
func decodeEnvelope[T any](op string, code int, fallback string, body []byte) (*Response[T], error) {
	out := &Response[T]{Raw: json.RawMessage(body)}
	if len(bytes.TrimSpace(body)) == 0 {
		return out, nil
	}
	var env struct {
		Status    FlexString      `json:"status"`
		Message   FlexString      `json:"message"`
		ErrorCode FlexString      `json:"errorCode"`
		Data      json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, decodeError(op, body, err)
	}
	out.Status, out.Message, out.ErrorCode = env.Status, env.Message, env.ErrorCode
	if !out.Succeeded() {
		return nil, newAPIError(op, code, messageOr(env.Message, fallback), string(env.ErrorCode), body)
	}
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &out.Data); err != nil {
			return nil, decodeError(op, body, err)
		}
	}
	return out, nil
}

// decodeBoolEnvelope decodes a response that reports its outcome in a boolean
// "success" field — newProposal and generateCertificate — into out.
//
// Only an explicit false is a failure; an absent field counts as success. As a
// safeguard the standard envelope is honoured too: a {"status": "error"} body
// is a failure even though these endpoints are not documented to send one.
func decodeBoolEnvelope(op string, code int, fallback string, body []byte, out any) error {
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var env struct {
		Success   *FlexBool  `json:"success"`
		Status    FlexString `json:"status"`
		Message   FlexString `json:"message"`
		ErrorCode FlexString `json:"errorCode"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return decodeError(op, body, err)
	}
	statusFailed := !(&Response[struct{}]{Status: env.Status}).Succeeded()
	if (env.Success != nil && !bool(*env.Success)) || statusFailed {
		return newAPIError(op, code, messageOr(env.Message, fallback), string(env.ErrorCode), body)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return decodeError(op, body, err)
	}
	return nil
}

// decodeError distinguishes a body that is not JSON at all from JSON whose
// structure does not match the expected type. Both carry the raw body.
func decodeError(op string, body []byte, err error) *ClientError {
	if !json.Valid(body) {
		return &ClientError{Type: InternalError, Code: ErrNonJSONResponse, Operation: op,
			Message: "failed to parse response: " + err.Error(), Response: body, Err: err}
	}
	return &ClientError{Type: InternalError, Code: ErrUnmarshalResponse, Operation: op,
		Message: "response did not match the expected shape: " + err.Error(), Response: body, Err: err}
}

func messageOr(m FlexString, fallback string) string {
	if m == "" {
		return fallback
	}
	return string(m)
}

// referenceData serves a reference-data endpoint from cache when a live entry
// exists; otherwise it fetches, and caches only a successful, non-empty body.
func referenceData[T any](ctx context.Context, c *Client, op, key, path string, code int, fallback string, forceRefresh bool) (*Response[T], error) {
	if !forceRefresh {
		if resp, ok := cachedEnvelope[T](c, op, key, code, fallback); ok {
			return resp, nil
		}
	}

	mu, _ := c.refLocks.LoadOrStore(key, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()

	// Another caller may have filled the cache while this one waited.
	if !forceRefresh {
		if resp, ok := cachedEnvelope[T](c, op, key, code, fallback); ok {
			return resp, nil
		}
	}

	body, err := c.do(ctx, request{op: op, method: http.MethodGet, path: path})
	if err != nil {
		return nil, err
	}
	resp, err := decodeEnvelope[T](op, code, fallback, body)
	if err != nil {
		return nil, err // error responses are never cached
	}
	if len(bytes.TrimSpace(body)) > 0 {
		c.cache.Set(key, body, c.cfg.CacheTTL)
	}
	return resp, nil
}

// cachedEnvelope decodes a cached body. An unreadable entry is dropped so it is re-fetched.
func cachedEnvelope[T any](c *Client, op, key string, code int, fallback string) (*Response[T], bool) {
	body, ok := c.cache.Get(key)
	if !ok {
		return nil, false
	}
	resp, err := decodeEnvelope[T](op, code, fallback, body)
	if err != nil {
		c.debugf("discarding unreadable cache entry %q", key)
		c.cache.Remove(key)
		return nil, false
	}
	c.debugf("serving %q from cache", key)
	return resp, true
}

func (c *Client) debugf(format string, args ...any) {
	if c.cfg.Debug {
		fmt.Printf("[DEFINITE DEBUG] "+format+"\n", args...)
	}
}
