package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func TestCLIRecuperaInstantaneaEInformeIdenticos(t *testing.T) {
	for _, fixture := range []string{"preparacion_liquidacion.json", "preparacion_liquidacion_gastos.json"} {
		t.Run(fixture, func(t *testing.T) {
			entrada, err := os.ReadFile("testdata/" + fixture)
			if err != nil {
				t.Fatal(err)
			}
			var s, recuperada bytes.Buffer
			if ejecutarConArgumentos([]string{"--preparar-liquidacion"}, bytes.NewReader(entrada), &s) != 0 {
				t.Fatal(s.String())
			}
			args := []string{"--preparar-liquidacion", "--desde-instantanea"}
			if ejecutarConArgumentos(args, bytes.NewReader(s.Bytes()), &recuperada) != 0 || !bytes.Equal(s.Bytes(), recuperada.Bytes()) {
				t.Fatal("instantánea recuperada discordante", recuperada.String())
			}
			for _, idioma := range []string{"es", "en"} {
				informeArgs := []string{"--informe", "--textos", "../../web/static/textos/" + idioma + "/dietas-liquidacion-informe.json", "--tema", "../../web/static/comun/tema-vec.css"}
				var original, informe bytes.Buffer
				if ejecutarConArgumentos(append([]string{"--preparar-liquidacion"}, informeArgs...), bytes.NewReader(entrada), &original) != 0 {
					t.Fatal(original.String())
				}
				if ejecutarConArgumentos(append(args, informeArgs...), bytes.NewReader(s.Bytes()), &informe) != 0 || !bytes.Equal(original.Bytes(), informe.Bytes()) {
					t.Fatal("informe recuperado discordante", informe.String())
				}
			}
			if ejecutarRecuperacion(bytes.NewReader(s.Bytes()), failingIO{}) != 1 {
				t.Fatal("fallo de salida")
			}
			limite := append(bytes.Clone(s.Bytes()), bytes.Repeat([]byte(" "), limiteEntrada-s.Len())...)
			if ejecutarRecuperacion(bytes.NewReader(limite), io.Discard) != 0 || ejecutarRecuperacion(bytes.NewReader(append(limite, ' ')), io.Discard) != 2 {
				t.Fatal("límite de entrada")
			}
		})
	}
}

func TestCLIRecuperacionRechazaAntesDelInforme(t *testing.T) {
	var s bytes.Buffer
	if ejecutarPreparacion(bytes.NewReader(fixtureLiquidacion(t)), &s) != 0 {
		t.Fatal(s.String())
	}
	var instantanea domain.InstantaneaLiquidacionPropuesta
	if err := json.Unmarshal(s.Bytes(), &instantanea); err != nil {
		t.Fatal(err)
	}
	instantanea.Totales.RechazadoCentimos++
	instantanea.SnapshotSHA256 = ""
	instantanea.SnapshotSHA256, _ = domain.HuellaDatosLiquidacion(instantanea)
	manipulada, _ := json.Marshal(instantanea)
	args := []string{"--preparar-liquidacion", "--desde-instantanea", "--informe", "--textos", "ruta-inexistente", "--tema", "ruta-inexistente"}
	var out bytes.Buffer
	if ejecutarConArgumentos(args, bytes.NewReader(manipulada), &out) != 2 || out.String() != "{\"codigo\":\"preparacion_liquidacion_invalida\"}\n" {
		t.Fatal("no rechaza antes de abrir catálogos", out.String())
	}
	casos := []string{
		"null", "", s.String() + "{}",
		strings.Replace(s.String(), `"liquidable":false`, `"liquidable":false,"liquidable":false`, 1),
		strings.Replace(s.String(), `"liquidable":false`, `"Liquidable":false`, 1),
		strings.Replace(s.String(), `"liquidable":false`, `"liquidable":null`, 1),
		strings.Replace(s.String(), `"liquidable":false`, `"liquidable":false,"recibo":"inventado"`, 1),
		"{\"esquema\":\"\xff\"}",
	}
	for _, entrada := range casos {
		out.Reset()
		if ejecutarConArgumentos(args, strings.NewReader(entrada), &out) != 2 || out.String() != "{\"codigo\":\"entrada_json_invalida\"}\n" {
			t.Fatalf("entrada malformada: %s", out.String())
		}
	}
	if ejecutarRecuperacion(failingIO{}, io.Discard) != 2 {
		t.Fatal("fallo de lectura")
	}
}
