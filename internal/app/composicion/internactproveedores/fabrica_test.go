package internactproveedores

import (
	"crypto/tls"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestTLSVerificadoRechazaRutaDeFallbackSinVerificacion(t *testing.T) {
	cfg := &pgconn.Config{Host: "db.interno", TLSConfig: &tls.Config{ServerName: "db.interno", MinVersion: tls.VersionTLS12}}
	if !tlsVerificado(cfg) {
		t.Fatal("configuracion TLS verificada rechazada")
	}
	cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: "db.interno"}}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin TLS admitido")
	}
	cfg.Fallbacks[0].TLSConfig = &tls.Config{ServerName: "db.interno", InsecureSkipVerify: true}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin verificacion admitido")
	}
	cfg.Fallbacks = nil
	cfg.TLSConfig.ServerName = "otro.interno"
	if tlsVerificado(cfg) {
		t.Fatal("nombre TLS distinto admitido")
	}
}
