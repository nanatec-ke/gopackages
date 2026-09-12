package definite

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Environment selects the default API host.
type Environment string

const (
	// UAT is the User Acceptance Testing environment — the one the public docs cover.
	UAT Environment = "uat"
	// Production is the live environment.
	Production Environment = "production"
)

// Credentials are the HTTP Basic credentials issued by Definite Assurance.
// They are supplied by your Definite administrator, not the public docs.
type Credentials struct {
	Username string `json:"username"` // Basic-auth username
	Password string `json:"password"` // Basic-auth password
}

// CacheStrategy selects where cached reference data (products, makes & models) lives.
type CacheStrategy string

const (
	// CacheMemory keeps entries in-process. This is the default.
	CacheMemory CacheStrategy = "MEMORY"
	// CacheFile persists entries to a JSON file so they survive restarts.
	CacheFile CacheStrategy = "FILE"
	// CacheNone disables caching — every reference-data call hits the API.
	CacheNone CacheStrategy = "NONE"
)

// Config contains everything needed to create a Client.
type Config struct {
	// Credentials authenticate every request. Required.
	Credentials Credentials

	// Environment selects the default base URL. Defaults to UAT.
	Environment Environment

	// BaseURL overrides Environment when set. Trailing slashes are trimmed.
	BaseURL string

	// APIUser is sent as APIUser on payment initiation when the request omits it.
	// Defaults to Credentials.Username.
	APIUser string

	// Timeout bounds each request, applied through the request context, so it
	// holds even with a custom HTTPClient. Defaults to 30 seconds.
	Timeout time.Duration

	// Debug logs requests and responses — including bodies — to stdout.
	// Leave it off in production.
	Debug bool

	// CacheStrategy selects the reference-data cache backend. Defaults to CacheMemory.
	CacheStrategy CacheStrategy

	// CacheTTL is how long reference data stays cached. Defaults to one hour.
	CacheTTL time.Duration

	// CacheDir is the root directory for CacheFile. Defaults to os.TempDir().
	CacheDir string

	// Cache replaces the built-in backend — for example with a shared Redis
	// implementation so every replica shares one copy. It overrides CacheStrategy.
	Cache Cache

	// Headers are merged into every request. They can never override Authorization.
	Headers map[string]string

	// HTTPClient replaces the default *http.Client, e.g. to add tracing or a proxy.
	// Timeout still applies to every request.
	HTTPClient *http.Client
}

// Validate checks the configuration and applies defaults. NewClient calls it for you.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Credentials.Username) == "" || c.Credentials.Password == "" {
		return fmt.Errorf("basic auth credentials (username and password) are required")
	}
	switch c.Environment {
	case "":
		c.Environment = UAT
	case UAT, Production:
	default:
		return fmt.Errorf("invalid Environment %q, must be %q or %q", c.Environment, UAT, Production)
	}
	switch c.CacheStrategy {
	case "":
		c.CacheStrategy = CacheMemory
	case CacheMemory, CacheFile, CacheNone:
	default:
		return fmt.Errorf("invalid CacheStrategy %q, must be %q, %q or %q", c.CacheStrategy, CacheMemory, CacheFile, CacheNone)
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}
	if c.CacheTTL < 0 {
		return fmt.Errorf("CacheTTL must not be negative")
	}
	if c.CacheTTL == 0 {
		c.CacheTTL = DefaultCacheTTL
	}
	return nil
}

// GetBaseURL returns the resolved base URL: BaseURL when set, otherwise the
// host for Environment. Trailing slashes are always trimmed.
func (c *Config) GetBaseURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/")
	}
	if c.Environment == Production {
		return BaseURLProduction
	}
	return BaseURLUAT
}
