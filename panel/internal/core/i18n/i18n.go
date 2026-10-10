// Package i18n is a minimal translation layer; English is the only locale for
// now, with the bundle in place so more locales slot in later.
package i18n

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/en.json
var enJSON []byte

type Bundle struct {
	m map[string]string
}

func New(lang string) *Bundle {
	b := &Bundle{m: map[string]string{}}
	// Only en exists today; ignore lang until more locales are added.
	_ = lang
	_ = json.Unmarshal(enJSON, &b.m)
	return b
}

// T returns the translation for key, falling back to the key itself.
func (b *Bundle) T(key string, args ...any) string {
	s, ok := b.m[key]
	if !ok {
		s = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}
