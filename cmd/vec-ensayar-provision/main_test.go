package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/domain"
)

func TestCLIConsumidorRealYJSONEstricto(t *testing.T) {
	var entrada, salida, errores bytes.Buffer
	if rc := ejecutar([]string{"--ejemplo", "--emitir-entrada"}, strings.NewReader(""), &entrada, &errores); rc != 0 {
		t.Fatal(rc, errores.String())
	}
	if rc := ejecutar(nil, &entrada, &salida, &errores); rc != 0 {
		t.Fatal(rc, errores.String())
	}
	var ciclo domain.CicloEnsayado
	if err := json.Unmarshal(salida.Bytes(), &ciclo); err != nil {
		t.Fatal(err)
	}
	if ciclo.SchemaVersion != domain.VersionCiclo || len(ciclo.Valoraciones) != 2 || ciclo.Resolucion.Estado != "borrador" || ciclo.Resolucion.EfectoOficial {
		t.Fatal("contrato", salida.String())
	}
}

func TestCLIRechazaDocumentosHostilesYSinFiltrarEntrada(t *testing.T) {
	var ejemplo, diagnostico bytes.Buffer
	if rc := ejecutar([]string{"--ejemplo", "--emitir-entrada"}, strings.NewReader(""), &ejemplo, &diagnostico); rc != 0 {
		t.Fatal(rc)
	}
	datos := ejemplo.String()
	casos := []string{
		datos + datos,
		strings.Replace(datos, `"schema_version":"provision.ciclo.ensayo.v1"`, `"schema_version":"provision.ciclo.ensayo.v1","schema_version":"otra"`, 1),
		strings.Replace(datos, `"schema_version":"provision.ciclo.ensayo.v1"`, `"Schema_Version":"provision.ciclo.ensayo.v1"`, 1),
		strings.Replace(datos, `"alcance":`, `"alcance":`, 1) + `{"secreto":"no_emitir_este_contenido"}`,
		strings.Replace(datos, `"version_esperada":1`, `"version_esperada":null`, 1),
		strings.Replace(datos, `"version_esperada":1`, `"version_esperada":-1`, 1),
		`{"schema_version":"provision.ciclo.ensayo.v1","extra":"no_emitir_este_contenido"}`,
	}
	for _, in := range casos {
		var out, err bytes.Buffer
		if rc := ejecutar(nil, strings.NewReader(in), &out, &err); rc != 2 || out.Len() != 0 || strings.Contains(err.String(), "no_emitir_este_contenido") {
			t.Fatalf("rc=%d out=%s err=%s", rc, out.String(), err.String())
		}
	}
}

func TestCLIArgumentos(t *testing.T) {
	for _, args := range [][]string{{"--emitir-entrada"}, {"--ejemplo", "sobrante"}, {"--desconocido"}} {
		var out, err bytes.Buffer
		if rc := ejecutar(args, strings.NewReader(""), &out, &err); rc != 2 || out.Len() != 0 {
			t.Fatal(args, rc)
		}
	}
}

func TestCLIDiagnosticoNoReflejaIDDeReglaValida(t *testing.T) {
	p, err := simulacion.EjemploCiclo()
	if err != nil {
		t.Fatal(err)
	}
	const marcador = "/ruta/sintetica/no_emitir"
	regla := &p.Configuracion.Reglas[0]
	regla.ID = marcador
	regla.MinDiferencia, regla.MaxDiferencia = 0, 0
	regla.Tramos = []domain.Tramo{{ID: "igual", MinDiferencia: 0, MaxDiferencia: 0, Coeficiente: regla.Maximo, Maximo: regla.Maximo}}
	if err := domain.ValidarConfiguracion(p.Configuracion); err != nil {
		t.Fatal("la configuración debe llegar al cálculo", err)
	}
	datos, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	// Grado 24 y puesto 22 producen una diferencia fuera de la tabla válida.
	var salida, diagnostico bytes.Buffer
	if rc := ejecutar(nil, bytes.NewReader(datos), &salida, &diagnostico); rc != 2 || salida.Len() != 0 {
		t.Fatal("no rechazó el cálculo", rc, salida.String())
	}
	if strings.Contains(diagnostico.String(), marcador) {
		t.Fatal("el diagnóstico refleja un identificador de la entrada")
	}
	var nominal domain.Error
	if err := json.Unmarshal(diagnostico.Bytes(), &nominal); err != nil {
		t.Fatal(err)
	}
	if nominal.Codigo != "diferencia_fuera_tabla" || nominal.Campo != "ciclo" {
		t.Fatal("diagnóstico", nominal)
	}
}
