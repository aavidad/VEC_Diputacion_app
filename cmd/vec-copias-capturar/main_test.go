package main

import (
	"bytes"
	"context"
	"encoding/json"
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
		{"PGHOST": "/socket/inexistente,db.example.test"},
		{"PGHOST": "/socket/uno,/socket/dos"},
		{"PGHOST": "/socket/inexistente,"},
		{"PGHOST": "/socket/inexistente,,db.example.test"},
		{"PGHOST": "127.0.0.1,db.example.test"},
		{"PGHOST": "::1,127.0.0.1"},
		{"PGHOST": "127.0.0.1", "PGPASSWORD": "no-imprimir"},
		{"PGHOST": "127.0.0.1", "PGSERVICE": "permite-otro-host"},
		{"PGHOST": "127.0.0.1", "LD_PRELOAD": "inyectar"},
	} {
		if _, ok := entorno(configuracion{EntornoPG: env}); ok {
			t.Fatal("entorno admitido")
		}
	}
}

func TestCLIRechazaListaHostsAntesDelInventarioYLosComandos(t *testing.T) {
	for _, host := range []string{"/socket/inexistente,db.example.test", "/socket/uno,/socket/dos"} {
		t.Run(host, func(t *testing.T) {
			dir := t.TempDir()
			marca := filepath.Join(dir, "comando-ejecutado")
			captura := filepath.Join(dir, "captura")
			// A canary command would leave a marker if reached. The unavailable
			// descriptor also proves rejection precedes inventory observation.
			comando := map[string]any{"ejecutable": "/usr/bin/touch", "argumentos": []string{marca}}
			config := map[string]any{
				"formato_version": 1, "ensayo_local_sintetico": true,
				"limite_proceso_segundos": 1, "limite_liberacion_segundos": 1,
				"entorno_pg": map[string]string{"PGHOST": host},
				"descriptor": filepath.Join(dir, "no-leer.json"), "directorio_captura": captura,
				"escritores": []any{map[string]any{"id": "writer:sintetico", "cerrar": comando, "drenar": comando, "comprobar": comando, "reabrir": comando}},
			}
			b, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			ruta := filepath.Join(dir, "config.json")
			if os.WriteFile(ruta, b, 0600) != nil {
				t.Fatal("config")
			}
			var out, diag bytes.Buffer
			if run(context.Background(), []string{"-config", ruta}, &out, &diag) != 1 || diag.String() != "{\"error_clave\":\"copias_captura_configuracion\"}\n" || out.Len() != 0 {
				t.Fatalf("out=%q diag=%q", out.String(), diag.String())
			}
			for _, p := range []string{marca, captura} {
				if _, err := os.Stat(p); !os.IsNotExist(err) {
					t.Fatal("alcanzó efectos")
				}
			}
		})
	}
}

func TestEntornoAdmiteUnSoloDestinoLocal(t *testing.T) {
	for _, host := range []string{"/var/run/postgresql", "127.0.0.1", "::1"} {
		if _, ok := entorno(configuracion{EntornoPG: map[string]string{"PGHOST": host}}); !ok {
			t.Fatalf("host local bloqueado: %s", host)
		}
	}
}
