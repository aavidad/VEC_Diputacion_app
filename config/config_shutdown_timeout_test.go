package config

import (
	"testing"
	"time"
)

func TestShutdownTimeoutDesdeEntornoAcotado(t *testing.T) {
	for valor, esperado := range map[string]time.Duration{
		"": DefaultShutdownTimeout, "1s": time.Second, "60s": time.Minute,
		"500ms": DefaultShutdownTimeout, "61s": DefaultShutdownTimeout,
		"invalido": DefaultShutdownTimeout,
	} {
		t.Setenv(EnvHTTPShutdownTimeout, valor)
		if obtenido := Load().ShutdownTimeout; obtenido != esperado {
			t.Errorf("%q: %s, esperado %s", valor, obtenido, esperado)
		}
	}
	if obtenido := (Config{ShutdownTimeout: 2 * time.Hour}).Normalize().ShutdownTimeout; obtenido != DefaultShutdownTimeout {
		t.Fatalf("plazo no acotado: %s", obtenido)
	}
}
