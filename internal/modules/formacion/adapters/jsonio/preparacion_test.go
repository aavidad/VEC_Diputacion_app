package jsonio

import (
	"bytes"
	"errors"
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

func TestFuentesCorporativasExigenEnlacesInequivocos(t *testing.T) {
	validas := `{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"https://formacion.dipgra.es/"}`
	if _, err := leerFuentes([]byte(validas)); err != nil {
		t.Fatal(err)
	}
	for _, entrada := range []string{
		`{"plan_url":"https://www.dipgra.es/plan/"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":""}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plan_url":"https://otro.example/","plataforma_url":"https://formacion.dipgra.es/"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"http://formacion.dipgra.es/"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"https://usuario@formacion.dipgra.es/"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"https://formacion.dipgra.es/#seccion"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"https://formacion.dipgra.es/#"}`,
		`{"plan_url":"https://www.dipgra.es/plan/","plataforma_url":"https://formacion.dipgra.es/","tercero":"https://otro.example/"}`,
	} {
		if _, err := leerFuentes([]byte(entrada)); !errors.Is(err, ErrConfiguracion) {
			t.Fatalf("aceptó configuración ambigua: %s", entrada)
		}
	}
}

type escritorConError struct{ err error }

func (e escritorConError) Write([]byte) (int, error) { return 0, e.err }

func TestEscribirDistingueErrorDelEscritor(t *testing.T) {
	err := Escribir(escritorConError{ErrConfiguracion}, application.Preparacion{})
	if !errors.Is(err, ErrSalida) || !errors.Is(err, ErrConfiguracion) {
		t.Fatalf("perdió origen o causa del error de salida: %v", err)
	}
}

func TestEscribirRechazaFuentesInvalidasSinEmitirBorrador(t *testing.T) {
	anterior := fuentesJSON
	fuentesJSON = []byte(`{"plan_url":"https://www.dipgra.es/plan/"}`)
	t.Cleanup(func() { fuentesJSON = anterior })
	var salida bytes.Buffer
	if err := Escribir(&salida, application.Preparacion{}); !errors.Is(err, ErrConfiguracion) || salida.Len() != 0 {
		t.Fatalf("escritura con fuentes invalidas: error=%v bytes=%d", err, salida.Len())
	}
}
