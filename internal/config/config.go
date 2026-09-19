// Package config loads and validates the BFF configuration.
//
// Profiles are declared in YAML; their API keys are never written there.
// Each profile names an environment variable, and loading fails closed when
// that variable is unset or empty. Keys are stored as Secret so they cannot
// be printed by accident.
package config

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LookupEnv abstracts os.LookupEnv so tests can inject an environment.
type LookupEnv func(key string) (string, bool)

// Config is the fully resolved configuration.
type Config struct {
	Server   Server   `yaml:"server"`
	Auth     Auth     `yaml:"auth"`
	Cache    Cache    `yaml:"cache"`
	Activity Activity `yaml:"activity"`
	Upstream Upstream `yaml:"upstream"`
}

// Server holds HTTP listener settings.
type Server struct {
	Addr            string        `yaml:"addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

// Auth holds browser-facing authentication settings.
type Auth struct {
	KeyEnv     string        `yaml:"key_env"`
	Key        Secret        `yaml:"-"`
	SessionTTL time.Duration `yaml:"session_ttl"`
}

// Cache holds the upstream response cache settings.
type Cache struct {
	TTL time.Duration `yaml:"ttl"`
}

// Activity holds the session poller settings.
type Activity struct {
	PollInterval          time.Duration `yaml:"poll_interval"`
	Window                time.Duration `yaml:"window"`
	MaxSessionsPerProfile int           `yaml:"max_sessions_per_profile"`
	IdleStopAfter         time.Duration `yaml:"idle_stop_after"`
}

// Upstream holds the Hermes client settings and the profile list.
type Upstream struct {
	Timeout  time.Duration `yaml:"timeout"`
	Profiles []Profile     `yaml:"profiles"`
}

// Profile is one Hermes profile reachable at its own base URL with its own key.
type Profile struct {
	Name    string `yaml:"name"`
	BaseURL string `yaml:"base_url"`
	KeyEnv  string `yaml:"key_env"`
	Key     Secret `yaml:"-"`
}

// MissingEnvError reports a referenced environment variable that is unset or
// empty. Profile is empty when the variable belongs to the auth section.
type MissingEnvError struct {
	Var     string
	Profile string
}

func (e *MissingEnvError) Error() string {
	if e.Profile == "" {
		return fmt.Sprintf("config: environment variable %s is required but empty", e.Var)
	}
	return fmt.Sprintf("config: environment variable %s (profile %q) is required but empty", e.Var, e.Profile)
}

// Load reads the YAML file at path and resolves secrets from the process
// environment.
func Load(path string) (*Config, error) {
	f, err := os.Open(path) //nolint:gosec // path comes from the operator's flag, not user input
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	defer f.Close()
	return Parse(f, os.LookupEnv)
}

// Parse decodes YAML from r, applies defaults, validates, and resolves every
// referenced environment variable through env.
func Parse(r io.Reader, env LookupEnv) (*Config, error) {
	cfg := defaults()
	dec := yaml.NewDecoder(r)
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("config: decode: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := cfg.resolveSecrets(env); err != nil {
		return nil, err
	}
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Server:   Server{Addr: ":8080", ReadTimeout: 10 * time.Second, ShutdownTimeout: 10 * time.Second},
		Auth:     Auth{SessionTTL: 24 * time.Hour},
		Cache:    Cache{TTL: 3 * time.Second},
		Activity: Activity{PollInterval: 5 * time.Second, Window: 10 * time.Minute, MaxSessionsPerProfile: 10, IdleStopAfter: 30 * time.Second},
		Upstream: Upstream{Timeout: 2 * time.Second},
	}
}

func (c *Config) validate() error {
	if c.Auth.KeyEnv == "" {
		return errors.New("config: auth.key_env is required")
	}
	if len(c.Upstream.Profiles) == 0 {
		return errors.New("config: upstream.profiles must list at least one profile")
	}
	seen := make(map[string]struct{}, len(c.Upstream.Profiles))
	for i := range c.Upstream.Profiles {
		p := &c.Upstream.Profiles[i]
		switch {
		case p.Name == "":
			return fmt.Errorf("config: profile #%d has no name", i)
		case strings.ContainsAny(p.Name, "/ \t"):
			return fmt.Errorf("config: profile %q: name must not contain slashes or whitespace", p.Name)
		case p.KeyEnv == "":
			return fmt.Errorf("config: profile %q: key_env is required", p.Name)
		case p.BaseURL == "":
			return fmt.Errorf("config: profile %q: base_url is required", p.Name)
		}
		u, err := url.Parse(p.BaseURL)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return fmt.Errorf("config: profile %q: base_url must be an absolute http(s) URL", p.Name)
		}
		p.BaseURL = strings.TrimRight(p.BaseURL, "/")
		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf("config: profile %q is declared twice", p.Name)
		}
		seen[p.Name] = struct{}{}
	}
	return nil
}

func (c *Config) resolveSecrets(env LookupEnv) error {
	v, ok := env(c.Auth.KeyEnv)
	if !ok || v == "" {
		return &MissingEnvError{Var: c.Auth.KeyEnv}
	}
	c.Auth.Key = Secret(v)
	for i := range c.Upstream.Profiles {
		p := &c.Upstream.Profiles[i]
		v, ok := env(p.KeyEnv)
		if !ok || v == "" {
			return &MissingEnvError{Var: p.KeyEnv, Profile: p.Name}
		}
		p.Key = Secret(v)
	}
	return nil
}
