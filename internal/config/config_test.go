package config

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

const validYAML = `
server:
  addr: ":9090"
auth:
  key_env: BFF_API_KEY
cache:
  ttl: 5s
upstream:
  timeout: 1500ms
  profiles:
    - name: default
      base_url: http://hermes:8642
      key_env: HERMES_KEY_DEFAULT
    - name: coder-agent
      base_url: http://hermes:8642/p/coder-agent/
      key_env: HERMES_KEY_CODER
`

func envFrom(m map[string]string) LookupEnv {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestParse_ResolvesKeysFromEnvAndAppliesDefaults(t *testing.T) {
	env := envFrom(map[string]string{
		"BFF_API_KEY":        "bff-secret",
		"HERMES_KEY_DEFAULT": "k-default",
		"HERMES_KEY_CODER":   "k-coder",
	})

	cfg, err := Parse(strings.NewReader(validYAML), env)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := &Config{
		Server: Server{Addr: ":9090", ReadTimeout: 10 * time.Second, ShutdownTimeout: 10 * time.Second},
		Auth:   Auth{KeyEnv: "BFF_API_KEY", Key: "bff-secret", SessionTTL: 24 * time.Hour},
		Cache:  Cache{TTL: 5 * time.Second},
		Activity: Activity{
			PollInterval:          5 * time.Second,
			Window:                10 * time.Minute,
			MaxSessionsPerProfile: 10,
			IdleStopAfter:         30 * time.Second,
		},
		Upstream: Upstream{
			Timeout: 1500 * time.Millisecond,
			Profiles: []Profile{
				{Name: "default", BaseURL: "http://hermes:8642", KeyEnv: "HERMES_KEY_DEFAULT", Key: "k-default"},
				// Trailing slash is normalised away so path joining is uniform.
				{Name: "coder-agent", BaseURL: "http://hermes:8642/p/coder-agent", KeyEnv: "HERMES_KEY_CODER", Key: "k-coder"},
			},
		},
	}
	if diff := cmp.Diff(want, cfg); diff != "" {
		t.Errorf("config mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_FailsWhenReferencedEnvIsEmpty(t *testing.T) {
	env := envFrom(map[string]string{
		"BFF_API_KEY":        "bff-secret",
		"HERMES_KEY_DEFAULT": "k-default",
		"HERMES_KEY_CODER":   "", // set but empty must be treated as missing
	})

	_, err := Parse(strings.NewReader(validYAML), env)
	if err == nil {
		t.Fatal("expected error for empty HERMES_KEY_CODER, got nil")
	}
	var me *MissingEnvError
	if !errors.As(err, &me) {
		t.Fatalf("expected *MissingEnvError, got %T: %v", err, err)
	}
	if me.Var != "HERMES_KEY_CODER" || me.Profile != "coder-agent" {
		t.Errorf("MissingEnvError = %+v, want Var=HERMES_KEY_CODER Profile=coder-agent", *me)
	}
	if strings.Contains(err.Error(), "k-default") {
		t.Errorf("error message leaks another key value: %q", err.Error())
	}
}

func TestParse_RejectsInvalidProfiles(t *testing.T) {
	env := envFrom(map[string]string{"BFF_API_KEY": "x", "K": "y"})
	cases := map[string]string{
		"no profiles": `
auth: {key_env: BFF_API_KEY}
upstream: {profiles: []}`,
		"duplicate name": `
auth: {key_env: BFF_API_KEY}
upstream:
  profiles:
    - {name: a, base_url: http://h, key_env: K}
    - {name: a, base_url: http://h/p/a, key_env: K}`,
		"missing base_url": `
auth: {key_env: BFF_API_KEY}
upstream:
  profiles:
    - {name: a, key_env: K}`,
		"base_url without scheme": `
auth: {key_env: BFF_API_KEY}
upstream:
  profiles:
    - {name: a, base_url: hermes:8642, key_env: K}`,
		"name with slash": `
auth: {key_env: BFF_API_KEY}
upstream:
  profiles:
    - {name: a/b, base_url: http://h, key_env: K}`,
		"missing auth key_env": `
upstream:
  profiles:
    - {name: a, base_url: http://h, key_env: K}`,
	}
	for name, yml := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(strings.NewReader(yml), env); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestSecret_NeverPrintsItsValue(t *testing.T) {
	s := Secret("hunter2")

	for name, got := range map[string]string{
		"String":  s.String(),
		"Sprint":  fmt.Sprint(s),
		"Sprintf": fmt.Sprintf("%v %s %q %+v", s, s, s, s),
	} {
		if strings.Contains(got, "hunter2") {
			t.Errorf("%s leaks the secret: %q", name, got)
		}
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Info("cfg", "key", s, "nested", struct{ K Secret }{s})
	if strings.Contains(buf.String(), "hunter2") {
		t.Errorf("slog output leaks the secret: %s", buf.String())
	}
	if !strings.Contains(buf.String(), Redacted) {
		t.Errorf("slog output should show the %q placeholder: %s", Redacted, buf.String())
	}

	b, err := s.MarshalJSON()
	if err != nil || strings.Contains(string(b), "hunter2") {
		t.Errorf("MarshalJSON = %s, %v; must not contain the value", b, err)
	}
	if s.Reveal() != "hunter2" {
		t.Errorf("Reveal() = %q, want the raw value", s.Reveal())
	}
}
