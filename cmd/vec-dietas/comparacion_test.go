package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/dietas/application/preparacionliquidacion"
	"vec-diputacion-granada/internal/modules/dietas/domain"
)

func entradaComparacionCLI(t *testing.T) []byte {
	t.Helper()
	var salida bytes.Buffer
	if ejecutarPreparacion(bytes.NewReader(fixtureLiquidacion(t)), &salida) != 0 {
		t.Fatal(salida.String())
	}
	var s domain.InstantaneaLiquidacionPropuesta
	if err := json.Unmarshal(salida.Bytes(), &s); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(preparacionliquidacion.EntradaComparacion{Esquema: preparacionliquidacion.EsquemaComparacionLiquidacion, Anterior: s, Propuesta: s})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLICompararLiquidaciones(t *testing.T) {
	b := entradaComparacionCLI(t)
	var out bytes.Buffer
	if ejecutarConArgumentos([]string{"--comparar-liquidaciones"}, bytes.NewReader(b), &out) != 0 {
		t.Fatal(out.String())
	}
	var c preparacionliquidacion.ComparacionLiquidacion
	if err := json.Unmarshal(out.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	if c.Liquidable || c.Procedencia != "comparacion_local_sin_registrar" || c.Anterior != c.Propuesta || c.Totales.DiferenciaReconocidoCentimos != 0 || c.Totales.DiferenciaRechazadoCentimos != 0 || len(c.Lineas) != 2 {
		t.Fatalf("comparación discordante: %+v", c)
	}
	for _, l := range c.Lineas {
		if l.DiferenciaReconocidoCentimos != 0 || l.DiferenciaRechazadoCentimos != 0 || l.CambioRegla || l.CambioMotivo {
			t.Fatal("igualdad discordante")
		}
	}
	if ejecutarComparacion(bytes.NewReader(b), failingIO{}) != 1 || ejecutarComparacion(failingIO{}, io.Discard) != 2 {
		t.Fatal("fallos de E/S")
	}
	limite := append(bytes.Clone(b), bytes.Repeat([]byte(" "), limiteEntrada-len(b))...)
	if ejecutarComparacion(bytes.NewReader(limite), io.Discard) != 0 || ejecutarComparacion(bytes.NewReader(append(limite, ' ')), io.Discard) != 2 {
		t.Fatal("límite de entrada")
	}
}

func TestCLIComparacionRechazaJSONAmbiguoSinEmitirDatos(t *testing.T) {
	b := string(entradaComparacionCLI(t))
	casos := []string{
		"", "null", b + "{}", b + " false",
		strings.Replace(b, `"anterior":`, `"Anterior":`, 1),
		strings.Replace(b, `"anterior":`, `"anterior":null,"anterior":`, 1),
		strings.Replace(b, `"liquidable":false`, `"liquidable":null`, 1),
		strings.Replace(b, `"liquidable":false`, `"liquidable":false,"liquidable":false`, 1),
		strings.Replace(b, `"liquidable":false`, `"liquidable":false,"aprobado":true`, 1),
		strings.Replace(b, `"comision_version":2`, `"comision_version":"2"`, 1),
		strings.Replace(b, `"procedencia":"propuesta_sin_registrar",`, "", 1),
		"{\"esquema\":\"\xff\"}",
	}
	for _, entrada := range casos {
		var out bytes.Buffer
		if ejecutarConArgumentos([]string{"--comparar-liquidaciones"}, strings.NewReader(entrada), &out) != 2 || out.String() != "{\"codigo\":\"entrada_json_invalida\"}\n" {
			t.Fatalf("entrada ambigua aceptada o filtrada: %s", out.String())
		}
	}
}

func TestCLIComparacionRevalidaYRechazaSustitucion(t *testing.T) {
	for nombre, mutar := range map[string]func(*preparacionliquidacion.EntradaComparacion){
		"esquema": func(e *preparacionliquidacion.EntradaComparacion) { e.Esquema = "otro" },
		"ajena": func(e *preparacionliquidacion.EntradaComparacion) {
			e.Propuesta.ComisionRef = "dco_comision_ajena_20261001"
			e.Propuesta.SnapshotSHA256 = ""
			e.Propuesta.SnapshotSHA256, _ = domain.HuellaDatosLiquidacion(e.Propuesta)
		},
		"manipulada": func(e *preparacionliquidacion.EntradaComparacion) {
			e.Anterior.Totales.RechazadoCentimos++
			e.Anterior.SnapshotSHA256 = ""
			e.Anterior.SnapshotSHA256, _ = domain.HuellaDatosLiquidacion(e.Anterior)
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			var entrada preparacionliquidacion.EntradaComparacion
			if err := json.Unmarshal(entradaComparacionCLI(t), &entrada); err != nil {
				t.Fatal(err)
			}
			mutar(&entrada)
			b, err := json.Marshal(entrada)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			codigo := "comparacion_liquidacion_invalida"
			if nombre == "manipulada" {
				codigo = "preparacion_liquidacion_invalida"
			}
			if ejecutarComparacion(bytes.NewReader(b), &out) != 2 || out.String() != "{\"codigo\":\""+codigo+"\"}\n" {
				t.Fatalf("sustitución o datos emitidos: %s", out.String())
			}
		})
	}
	var out bytes.Buffer
	if ejecutarConArgumentos([]string{"--comparar-liquidaciones", "--informe"}, bytes.NewReader(entradaComparacionCLI(t)), &out) != 2 || out.String() != "{\"codigo\":\"argumentos_no_admitidos\"}\n" {
		t.Fatal("argumentos admitidos", out.String())
	}
}

func entradaComparacionConCambio(t *testing.T) []byte {
	t.Helper()
	datos, err := os.ReadFile("testdata/preparacion_liquidacion_gastos.json")
	if err != nil {
		t.Fatal(err)
	}
	var entrada preparacionliquidacion.Entrada
	if err := json.Unmarshal(datos, &entrada); err != nil {
		t.Fatal(err)
	}
	anterior, err := preparacionliquidacion.Preparar(entrada)
	if err != nil {
		t.Fatal(err)
	}
	entrada.Revisiones[2].ReconocidoPropuestoCentimos = 1800
	entrada.Revisiones[2].MotivoCodigo = ""
	propuesta, err := preparacionliquidacion.Preparar(entrada)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(preparacionliquidacion.EntradaComparacion{
		Esquema:  preparacionliquidacion.EsquemaComparacionLiquidacion,
		Anterior: anterior.Instantanea(), Propuesta: propuesta.Instantanea(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLIComparacionInformeLocalConCambioReal(t *testing.T) {
	entrada := entradaComparacionConCambio(t)
	for _, caso := range []struct{ idioma, totalAntes, totalDespues string }{
		{"es", "59,70", "62,70"}, {"en", "59.70", "62.70"},
	} {
		t.Run(caso.idioma, func(t *testing.T) {
			args := []string{"--comparar-liquidaciones", "--informe", "--textos", "../../web/static/textos/" + caso.idioma + "/dietas-comparacion-liquidacion-informe.json", "--tema", "../../web/static/comun/tema-vec.css"}
			var out bytes.Buffer
			if codigo := ejecutarConArgumentos(args, bytes.NewReader(entrada), &out); codigo != 0 {
				t.Fatal("informe no generado", codigo, out.String())
			}
			for _, esperado := range []string{`<!doctype html>`, `<html lang="` + caso.idioma + `">`, caso.totalAntes, caso.totalDespues, "3"} {
				if !strings.Contains(out.String(), esperado) {
					t.Errorf("falta %q", esperado)
				}
			}
			if strings.Contains(out.String(), "comparacion_local_sin_registrar") || strings.Contains(out.String(), "<script") {
				t.Fatal("estado interno o script visible")
			}
		})
	}
}

func TestCLIComparacionInformeFallaAntesDeEmitirHTML(t *testing.T) {
	args := []string{"--comparar-liquidaciones", "--informe", "--textos", "../../web/static/textos/es/dietas-comparacion-liquidacion-informe.json", "--tema", "../../web/static/comun/tema-vec.css"}
	var out bytes.Buffer
	if codigo := ejecutarConArgumentos(args, bytes.NewReader([]byte(`{"esquema":"ajeno"}`)), &out); codigo != 2 || !strings.HasPrefix(out.String(), `{"codigo":`) || strings.Contains(out.String(), "<!doctype") {
		t.Fatal("entrada inválida produjo HTML", codigo, out.String())
	}
	args[3] = "ruta-privada-inexistente"
	out.Reset()
	if codigo := ejecutarConArgumentos(args, bytes.NewReader(entradaComparacionConCambio(t)), &out); codigo != 2 || out.String() != "{\"codigo\":\"catalogo_informe_no_disponible\"}\n" || strings.Contains(out.String(), args[3]) {
		t.Fatal("ruta filtrada o HTML parcial", codigo, out.String())
	}
	args[3] = "../../web/static/textos/es/dietas-comparacion-liquidacion-informe.json"
	if codigo := ejecutarConArgumentos(args, bytes.NewReader(entradaComparacionConCambio(t)), salidaCortaInforme{}); codigo != 1 {
		t.Fatal("salida parcial anunciada como completa", codigo)
	}
}
