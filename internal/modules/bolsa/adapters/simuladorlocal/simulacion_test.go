package simuladorlocal

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/application/simulacionbaremo"
)

func TestCanalLocalConservaResultadoDelServicioCLI(t *testing.T) {
	for _, e := range (Motor{}).Ejemplos() {
		t.Run(e.Modo, func(t *testing.T) {
			// El navegador reordena claves y añade espacios: el adaptador restaura
			// el canon del motor sin perder precisión de las cantidades.
			var objeto map[string]json.RawMessage
			if err := json.Unmarshal(e.Reglas, &objeto); err != nil {
				t.Fatal(err)
			}
			reglas, err := json.MarshalIndent(objeto, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			local, err := (Motor{}).Simular(Solicitud{e.Modo, e.Referencia, reglas})
			if err != nil {
				t.Fatal(err)
			}
			var r, entrada, esperado []byte
			if e.Modo == "experiencia" {
				r, entrada = simulacionbaremo.DatosEjemploExperiencia()
				v, err := (simulacionbaremo.Servicio{}).Simular(simulacionbaremo.Solicitud{ConjuntoCanonico: r, HuellaConjuntoSHA256: huella(r), EntradaCanonica: entrada, HuellaEntradaSHA256: huella(entrada)})
				if err != nil {
					t.Fatal(err)
				}
				esperado = v.RepresentacionCanonica()
			} else {
				r, entrada = simulacionbaremo.DatosEjemploMeritos()
				v, err := (simulacionbaremo.ServicioMeritos{}).SimularMeritos(simulacionbaremo.Solicitud{ConjuntoCanonico: r, HuellaConjuntoSHA256: huella(r), EntradaCanonica: entrada, HuellaEntradaSHA256: huella(entrada)})
				if err != nil {
					t.Fatal(err)
				}
				esperado = v.RepresentacionCanonica()
			}
			if !bytes.Equal(local, esperado) {
				t.Fatal("resultado distinto del servicio CLI")
			}
		})
	}
}

func TestJSONEstrictoYLimites(t *testing.T) {
	for _, b := range []string{
		`{"modo":"experiencia","modo":"meritos","reglas":{}}`,
		`{"modo":"experiencia","MODO":"meritos","reglas":{}}`,
		`{"reglas":{"verſion":1}}`,
		`{"modo":"experiencia","reglas":{"x":1,"x":2}}`,
		`{"modo":"experiencia","entrada":{},"reglas":{}}`,
		`{"reglas":{}} {}`, `{"reglas":null}`, `[]`,
		`{"reglas":{"a":[` + strings.Repeat("0,", 128) + `0]}}`,
		`{"reglas":` + strings.Repeat(`{"a":`, 34) + `0` + strings.Repeat(`}`, 34) + `}`,
		strings.Repeat(" ", MaximoBytes+1),
	} {
		if _, err := Decodificar([]byte(b)); err == nil {
			t.Fatal("aceptada entrada no admitida")
		}
	}
}

func TestEjemploAjenoNoAdmiteEntradaLibre(t *testing.T) {
	e := (Motor{}).Ejemplos()[0]
	for _, s := range []Solicitud{{"meritos", e.Referencia, e.Reglas}, {e.Modo, "../../persona.json", e.Reglas}, {e.Modo, e.Referencia, json.RawMessage(`{"persona":"real"}`)}} {
		if _, err := (Motor{}).Simular(s); err == nil {
			t.Fatal("entrada ajena admitida")
		}
	}
}
