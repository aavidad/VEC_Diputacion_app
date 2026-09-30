package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/config"
)

func TestConfiguracionAvisosNoSeInfiereAlDesactivar(t *testing.T) {
	t.Setenv(envAvisosExternos, "false")
	cerrar, err := componerBuzonAvisosExternos(context.Background(), config.Config{}, nil, nil, nil)
	if err != nil || cerrar == nil {
		t.Fatal("el receptor apagado no debe requerir material ni conexiones")
	}
	cerrar()
}
func TestConfiguracionAvisosFallaAnteActivacionMalformada(t *testing.T) {
	t.Setenv(envAvisosExternos, "TRUE")
	if _, err := componerBuzonAvisosExternos(context.Background(), config.Config{}, nil, nil, nil); err == nil {
		t.Fatal("un selector malformado no puede arrancar")
	}
}
func TestConfiguracionAvisosUsaSuMaterialYLimites(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "usuarios"), 0700); err != nil {
		t.Fatal(err)
	}
	c := configuracionAvisosExternos{Esquema: esquemaAvisosExternos, Version: 1, ProductorRef: "productor:sintetico", Lote: 32, Intervalo: "5s", Idioma: "es", URLPersonal: "https://portal.example/area-personal/"}
	escribir := func(raw []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, "usuarios", "avisos-externos.json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	escribir(raw)
	recibida, err := leerConfiguracionAvisosExternos(config.Config{DevelopmentMaterialDir: root}, "usuarios/avisos-externos.json")
	if err != nil || recibida != c {
		t.Fatal("no recupera configuración propia", err)
	}
	if _, err = recibida.intervaloValido(); err != nil {
		t.Fatal(err)
	}
	recibida.Idioma = "../es"
	if _, err = recibida.intervaloValido(); err == nil {
		t.Fatal("un idioma no puede salir del catálogo")
	}
	recibida = c
	recibida.Intervalo = "1ms"
	if _, err = recibida.intervaloValido(); err == nil {
		t.Fatal("sondeo sin límite inferior")
	}
	escribir([]byte(strings.TrimSuffix(string(raw), "}") + `,"correo":"externo@example.test"}`))
	if _, err = leerConfiguracionAvisosExternos(config.Config{DevelopmentMaterialDir: root}, "usuarios/avisos-externos.json"); err == nil {
		t.Fatal("la configuración no admite destino de correo")
	}
}
