package config

import (
	"testing"
	"time"
)

func TestIdleTimeoutDesdeEntornoAcotado(t *testing.T) {
	casos := map[string]time.Duration{
		"":       DefaultIdleTimeout,
		"60m":    60 * time.Minute,
		"30s":    30 * time.Second,
		"10s":    DefaultIdleTimeout,
		"3h":     DefaultIdleTimeout,
		"basura": DefaultIdleTimeout,
	}
	for valor, esperado := range casos {
		t.Setenv(EnvHTTPIdleTimeout, valor)
		if obtenido := idleTimeoutDesdeEntorno(); obtenido != esperado {
			t.Errorf("%q: obtenido %v, esperado %v", valor, obtenido, esperado)
		}
	}
}
