package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func TestCLIProduceRegistroSinAprobacion(t *testing.T) {
	datos, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	var out, diagnostic bytes.Buffer
	if codigo := ejecutar(context.Background(), nil, bytes.NewReader(datos), &out, &diagnostic); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, diagnostic.String())
	}
	var r struct {
		Estado string `json:"estado"`
		Huella string `json:"huella_material_sha256"`
		Notas  []struct {
			Estado string `json:"estado"`
		} `json:"notas"`
		Aprobada bool `json:"aprobada"`
	}
	if err := json.Unmarshal(out.Bytes(), &r); err != nil || r.Estado != "borrador_pendiente_validacion" || r.Notas[1].Estado != "pendiente" || r.Aprobada || len(r.Huella) != 64 {
		t.Fatalf("salida inesperada: %v %s", err, out.String())
	}
}

func BenchmarkCLI128Notas(b *testing.B) {
	datos, err := os.ReadFile("testdata/material.json")
	if err != nil {
		b.Fatal(err)
	}
	var material domain.MaterialCalificacionesEjercicio
	if err := json.Unmarshal(datos, &material); err != nil {
		b.Fatal(err)
	}
	material.Notas = make([]domain.NotaEjercicio, 128)
	for i := range material.Notas {
		material.Notas[i].SolicitudRef = fmt.Sprintf("aspirante_%03d", i)
		if i%2 == 0 {
			puntos := int64(7_000_000)
			material.Notas[i].PuntosMicropuntos = &puntos
			material.Notas[i].FuenteRef = fmt.Sprintf("correccion_%03d", i)
		}
	}
	datos, err = json.Marshal(material)
	if err != nil {
		b.Fatal(err)
	}
	muestras := make([]int64, 0, 1024)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inicio := time.Now()
		if codigo := ejecutar(context.Background(), nil, bytes.NewReader(datos), io.Discard, io.Discard); codigo != 0 {
			b.Fatalf("código %d", codigo)
		}
		if len(muestras) < cap(muestras) {
			muestras = append(muestras, time.Since(inicio).Nanoseconds())
		}
	}
	b.StopTimer()
	sort.Slice(muestras, func(i, j int) bool { return muestras[i] < muestras[j] })
	if len(muestras) > 0 {
		indice := (95*len(muestras)+99)/100 - 1
		b.ReportMetric(float64(muestras[indice]), "p95-ns/op")
		b.ReportMetric(float64(len(muestras)), "p95-samples")
	}
}

func TestCLIRechazaClavesDuplicadasYDesconocidas(t *testing.T) {
	base, err := os.ReadFile("testdata/material.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, datos := range []string{
		`{"esquema":"a","esquema":"b"}`,
		`{"campo_desconocido":1}`,
		string(bytes.Replace(base, []byte(`"esquema"`), []byte(`"Esquema"`), 1)),
		string(bytes.Replace(base, []byte(`"esquema"`), []byte(`"esquema":"vec.seleccion.calificaciones-ejercicio.v1","Esquema"`), 1)),
		string(bytes.Replace(base, []byte(`"version"`), []byte(`"Version"`), 1)),
		string(bytes.Replace(base, []byte(`"version"`), []byte(`"version":2,"Version"`), 1)),
		string(bytes.Replace(base, []byte(`"solicitud_ref"`), []byte(`"Solicitud_ref"`), 1)),
		strings.Repeat(" ", maximoEntrada+1),
	} {
		var out, diagnostico bytes.Buffer
		if codigo := ejecutar(context.Background(), nil, strings.NewReader(datos), &out, &diagnostico); codigo != 1 || out.Len() != 0 {
			t.Fatalf("código %d, salida %q", codigo, out.String())
		}
	}
}
