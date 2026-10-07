package jsonio

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/formacion/application"
)

func TestEntradaLimitadaYEstricta(t *testing.T) {
	for _, in := range []string{strings.Repeat(" ", MaxEntrada+1), `{"alcance":"preparacion_sintetica","alcance":"real"}`, `{} {}`, `{"dni":"inventado"}`, strings.Repeat("[", 14) + strings.Repeat("]", 14)} {
		if _, err := Leer(strings.NewReader(in)); err == nil {
			t.Fatal("entrada ambigua aceptada")
		}
	}
}

func TestCamposMayusculasNoSustituyenClaveCanonica(t *testing.T) {
	if err := documentoUnico([]byte(`{"alcance":"real","Alcance":"preparacion_sintetica"}`)); err == nil {
		t.Fatal("aceptó alias de mayúsculas para el mismo campo")
	}
}

func TestConfiguracionAusenteSeProyectaComoListasVacias(t *testing.T) {
	p, err := Proyectar(application.Preparacion{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Plan.Configuracion.Modalidades == nil || p.Plan.Configuracion.Prioridades == nil {
		t.Fatal("proyectó null en listas de configuración")
	}
}
