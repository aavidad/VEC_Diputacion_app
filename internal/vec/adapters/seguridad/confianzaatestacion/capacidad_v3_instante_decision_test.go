package confianzaatestacion

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestInstanteDecisionCanonicoCapacidadV3ConservaSeisMicrosegundos(t *testing.T) {
	base := time.Date(2026, 7, 23, 9, 1, 30, 0, time.UTC)
	for _, caso := range []struct {
		nombre   string
		instante time.Time
		esperado string
	}{
		{"000000", base, "2026-07-23T09:01:30.000000Z"},
		{"120000", base.Add(120000 * time.Microsecond), "2026-07-23T09:01:30.120000Z"},
		{"123400", base.Add(123400 * time.Microsecond), "2026-07-23T09:01:30.123400Z"},
		{"123456", base.Add(123456 * time.Microsecond), "2026-07-23T09:01:30.123456Z"},
		{"offset_equivalente", time.Date(2026, 7, 23, 11, 1, 30, 123400000, time.FixedZone("UTC+02", 2*60*60)), "2026-07-23T09:01:30.123400Z"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			obtenido, valido := instanteDecisionCanonicoCapacidadV3(caso.instante)
			if !valido || obtenido != caso.esperado {
				t.Fatalf("canon de decisión = %q, válido=%v; se esperaba %q", obtenido, valido, caso.esperado)
			}
			parseado, err := parsearInstanteDecisionCapacidadV3(caso.esperado)
			if err != nil || !parseado.Equal(caso.instante) {
				t.Fatalf("parser de decisión no conserva instante: %v", err)
			}
		})
	}
	if obtenido, valido := instanteDecisionCanonicoCapacidadV3(base.Add(123456789 * time.Nanosecond)); valido || obtenido != "" {
		t.Fatalf("precisión inferior a microsegundo truncada: %q, válido=%v", obtenido, valido)
	}
	for _, historico := range []string{
		"2026-07-23T09:01:30Z", "2026-07-23T09:01:30.12Z",
		"2026-07-23T09:01:30.12345Z",
	} {
		if _, err := parsearInstanteDecisionCapacidadV3(historico); err != nil {
			t.Fatalf("canon RFC3339Nano histórico rechazado: %q", historico)
		}
	}
	for _, ajeno := range []string{
		"2026-07-23T09:01:30.1234560Z", "2026-07-23T09:01:30.1200000Z",
		"2026-07-23T09:01:30.123456789Z", "2026-07-23T11:01:30.123400+02:00",
		"2026-07-23T09:01:30.120000+00:00", "2026-07-23t09:01:30.120000z",
	} {
		if _, err := parsearInstanteDecisionCapacidadV3(ajeno); err == nil {
			t.Fatalf("forma temporal ajena aceptada: %q", ajeno)
		}
	}
}

func TestVectorHistoricoO205ConservaBytesYMAC(t *testing.T) {
	contenido, err := os.ReadFile(filepath.Join("testdata", "capacidad_v3_canonica_o2_05.json"))
	if err != nil {
		t.Fatal(err)
	}
	contenido = bytes.TrimSuffix(contenido, []byte{'\n'})
	documento, err := interpretarExportacionCapacidadV3(contenido)
	if err != nil {
		t.Fatalf("vector O205 histórico rechazado: %v", err)
	}
	if documento.DecisionValidaHasta != "2099-01-01T00:02:00Z" ||
		documento.MACSHA256 != strings.Repeat("d", 64) {
		t.Fatal("el vector histórico perdió instante o campo MAC")
	}
	recodificado, err := json.Marshal(documento)
	if err != nil || !bytes.Equal(recodificado, contenido) {
		t.Fatalf("el vector histórico cambió bytes firmados: %v", err)
	}
}

func TestEmisionCapacidadV3CopiaLiteralValidaHastaDeDecisionCanonica(t *testing.T) {
	escenario, prueba := escenarioYPruebaConfianzaV3(t)
	clave := claveCapacidadAtestacionV3Prueba(t,
		EstadoClaveHMACCapacidadAtestacionV3Emision, time.Time{}, bytes.Repeat([]byte{0x73}, 32))
	emisor, err := NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave,
		&relojConfianzaAtestacionV3Prueba{ahora: escenario.ahora})
	if err != nil {
		t.Fatal(err)
	}
	capacidad, err := emisor.Emitir(context.Background(), escenario.solicitud, escenario.decision,
		escenario.motivo, escenario.resultado, escenario.atestacion, prueba)
	if err != nil {
		t.Fatal(err)
	}
	exportacion, err := capacidad.ExportacionCanonicaParaConsumidor()
	if err != nil {
		t.Fatal(err)
	}
	documentoCapacidad, err := interpretarExportacionCapacidadV3(exportacion)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.RepresentacionCanonicaDecisionAutorizacionV3(escenario.decision)
	if err != nil {
		t.Fatal(err)
	}
	var documentoDecision struct {
		ValidaHasta string `json:"valida_hasta"`
	}
	if err := json.Unmarshal(decision, &documentoDecision); err != nil {
		t.Fatal(err)
	}
	if documentoDecision.ValidaHasta == "" || documentoCapacidad.DecisionValidaHasta != documentoDecision.ValidaHasta {
		t.Fatalf("atributo de capacidad difiere de decisión V3: capacidad=%q decisión=%q",
			documentoCapacidad.DecisionValidaHasta, documentoDecision.ValidaHasta)
	}
	material, err := NuevoMaterialConsumoAutorizacionAtestadaV3(
		escenario.solicitud, escenario.decision, escenario.motivo, escenario.resultado,
		escenario.atestacion, prueba, capacidad, escenario.raiz)
	if err != nil {
		t.Fatalf("material V3 no conserva el atributo exacto: %v", err)
	}
	entrega, err := material.ExportarMaterialParaConsumidor()
	if err != nil || !bytes.Equal(entrega.DecisionCanonica(), decision) ||
		!bytes.Equal(entrega.CapacidadCanonica(), exportacion) {
		t.Fatalf("material V3 alteró decisión o capacidad: %v", err)
	}
	parseado, err := parsearInstanteDecisionCapacidadV3(documentoCapacidad.DecisionValidaHasta)
	_, esperado, errVentana := escenario.decision.VentanaValidez()
	if err != nil || errVentana != nil || !parseado.Equal(esperado) {
		t.Fatalf("material V3 cambió la validez temporal: %v / %v", err, errVentana)
	}
}
