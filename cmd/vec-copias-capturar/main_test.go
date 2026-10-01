package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfiguracionNoAdmiteInterpretacionesAmbiguas(t *testing.T) {
	for _, documento := range []string{
		`{"formato_version":1,"formato_version":2}`,
		`{"formato_version":1,"FORMATO_VERSION":2}`,
		`{"formato_version":null}`,
		`{"formato_version":1,"desconocido":true}`,
		`{"formato_version":1} {}`,
		`{"entorno_pg":{"PGHOST":"127.0.0.1","PGHOST":"otro"}}`,
	} {
		if _, err := leerConfiguracion(strings.NewReader(documento)); err == nil {
			t.Fatal(documento)
		}
	}
}
func TestCLIRechazaProduccionSinExponerConfiguracion(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(ruta, []byte(`{"formato_version":1,"ensayo_local_sintetico":false,"origen_ref":"privado-no-imprimir"}`), 0600); err != nil {
		t.Fatal(err)
	}
	var out, diag bytes.Buffer
	if run(context.Background(), []string{"-config", ruta}, &out, &diag) != 1 {
		t.Fatal("admitió producción")
	}
	if strings.Contains(diag.String(), "privado") || strings.Contains(diag.String(), ruta) || out.Len() != 0 {
		t.Fatal("filtra configuración")
	}
}
func TestEntornoNoPermiteSecretosNiDestinoExterno(t *testing.T) {
	for _, env := range []map[string]string{
		{"PGHOST": "servidor-ejemplo"},
		{"PGHOST": "127.0.0.1", "PGPASSWORD": "no-imprimir"},
		{"PGHOST": "127.0.0.1", "PGSERVICE": "permite-otro-host"},
		{"PGHOST": "127.0.0.1", "LD_PRELOAD": "inyectar"},
	} {
		if _, ok := entorno(configuracion{EntornoPG: env}); ok {
			t.Fatal("entorno admitido")
		}
	}
}
