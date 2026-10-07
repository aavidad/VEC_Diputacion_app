package bootstrap

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

// El aviso nombra variable y dependencia ausente, nunca el valor puesto.
func TestAvisarDependenciasSelectoresBolsaCTSinValores(t *testing.T) {
	var salida bytes.Buffer
	registro := slog.New(slog.NewTextHandler(&salida, nil))
	const secreto = "postgresql://usuario:clave-secreta@host/base"
	entorno := map[string]string{
		config.EnvBolsaPortalCandidatoEnabled:  "true",
		config.EnvBolsaLlamamientosDatabaseURL: secreto,
	}
	n := avisarDependenciasSelectoresBolsaCT(registro, func(k string) string { return entorno[k] })
	texto := salida.String()
	if n == 0 || strings.Count(texto, "level=WARN") != n {
		t.Fatalf("avisos=%d, registro:\n%s", n, texto)
	}
	if !strings.Contains(texto, "variable="+config.EnvBolsaPortalCandidatoEnabled) ||
		!strings.Contains(texto, "falta="+config.EnvBolsaReglasSourcePath) {
		t.Fatalf("aviso incompleto:\n%s", texto)
	}
	if strings.Contains(texto, "clave-secreta") || strings.Contains(texto, "postgresql://") {
		t.Fatalf("el aviso no debe mostrar valores:\n%s", texto)
	}
	if avisarDependenciasSelectoresBolsaCT(registro, func(string) string { return "" }) != 0 {
		t.Fatal("sin variables activas no debe avisar")
	}
	if avisarDependenciasSelectoresBolsaCT(nil, nil) != 0 {
		t.Fatal("sin registro no debe avisar")
	}
}
