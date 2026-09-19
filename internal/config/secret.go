package config

import "log/slog"

// Redacted is the placeholder printed wherever a Secret would otherwise appear.
const Redacted = "[redacted]"

// Secret is a string that refuses to print itself. Every path that could turn
// a value into text — fmt verbs, slog attributes, JSON encoding — yields
// Redacted instead. The only way to obtain the raw value is Reveal, which
// keeps each use greppable.
type Secret string

// String implements fmt.Stringer and hides the value.
func (Secret) String() string { return Redacted }

// GoString implements fmt.GoStringer so %#v also hides the value.
func (Secret) GoString() string { return Redacted }

// LogValue implements slog.LogValuer and hides the value.
func (Secret) LogValue() slog.Value { return slog.StringValue(Redacted) }

// MarshalJSON hides the value.
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + Redacted + `"`), nil }

// MarshalText hides the value.
func (Secret) MarshalText() ([]byte, error) { return []byte(Redacted), nil }

// Reveal returns the raw value. Callers must never log or return it.
func (s Secret) Reveal() string { return string(s) }
