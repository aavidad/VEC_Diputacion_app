package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInformePreparacionConsumeCatalogosReales(t *testing.T) {
	entrada, err := os.ReadFile("testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct{ idioma, total string }{{"es", "44,70"}, {"en", "44.70"}} {
		t.Run(caso.idioma, func(t *testing.T) {
			args := []string{"--preparar-liquidacion", "--informe", "--textos", "../../web/static/textos/" + caso.idioma + "/dietas-liquidacion-informe.json", "--tema", "../../web/static/comun/tema-vec.css"}
			var out bytes.Buffer
			if code := ejecutarConArgumentos(args, bytes.NewReader(entrada), &out); code != 0 {
				t.Fatalf("code=%d: %s", code, out.String())
			}
			if !strings.HasPrefix(out.String(), "<!doctype html>") || !strings.Contains(out.String(), `<html lang="`+caso.idioma+`">`) || !strings.Contains(out.String(), caso.total) {
				t.Fatal("no devolvió el informe localizado de la preparación")
			}
		})
	}
}

func TestInformePreparacionNoExponeRutaYLimitaCatalogo(t *testing.T) {
	entrada, err := os.ReadFile("testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "catalogo-sintetico.json")
	if err := os.WriteFile(ruta, bytes.Repeat([]byte("x"), 65537), 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := ejecutarInformePreparacion(bytes.NewReader(entrada), &out, ruta, ""); code != 2 || strings.Contains(out.String(), ruta) || !strings.Contains(out.String(), "catalogo_informe_no_disponible") {
		t.Fatalf("respuesta incorrecta: code=%d %s", code, out.String())
	}
}

type salidaCortaInforme struct{}

func (salidaCortaInforme) Write(datos []byte) (int, error) {
	return len(datos) - 1, nil
}

func TestInformePreparacionRechazaSalidaIncompleta(t *testing.T) {
	entrada, err := os.ReadFile("testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	if code := ejecutarInformePreparacion(bytes.NewReader(entrada), salidaCortaInforme{}, "../../web/static/textos/es/dietas-liquidacion-informe.json", "../../web/static/comun/tema-vec.css"); code != 1 {
		t.Fatalf("salida incompleta anunciada como correcta: %d", code)
	}
}

type salidaFallaUnaVezInforme struct {
	llamadas int
}

func (s *salidaFallaUnaVezInforme) Write(datos []byte) (int, error) {
	s.llamadas++
	if s.llamadas == 1 {
		return 3, errors.New("causa_sintetica_no_visible")
	}
	return len(datos), nil
}

func TestInformePreparacionNoMezclaJSONConSalidaParcial(t *testing.T) {
	entrada, err := os.ReadFile("testdata/preparacion_liquidacion.json")
	if err != nil {
		t.Fatal(err)
	}
	salida := &salidaFallaUnaVezInforme{}
	if code := ejecutarInformePreparacion(bytes.NewReader(entrada), salida, "../../web/static/textos/es/dietas-liquidacion-informe.json", "../../web/static/comun/tema-vec.css"); code != 1 || salida.llamadas != 1 {
		t.Fatalf("fallo de salida no conservado: code=%d, escrituras=%d", code, salida.llamadas)
	}
}
