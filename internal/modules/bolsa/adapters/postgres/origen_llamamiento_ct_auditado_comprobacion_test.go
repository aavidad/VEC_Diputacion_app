package postgres

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func sobreOrigenConComprobacionPrueba(t *testing.T, sobre []byte, estado json.RawMessage) []byte {
	t.Helper()
	var exterior map[string]json.RawMessage
	if err := json.Unmarshal(sobre, &exterior); err != nil {
		t.Fatal(err)
	}
	var origen map[string]json.RawMessage
	if err := json.Unmarshal(exterior["origen"], &origen); err != nil {
		t.Fatal(err)
	}
	var entradas []map[string]json.RawMessage
	if err := json.Unmarshal(origen["entradas"], &entradas); err != nil || len(entradas) != 1 {
		t.Fatalf("entradas=%d error=%v", len(entradas), err)
	}
	entradas[0]["comprobacion_ofrecimiento"] = estado
	var err error
	origen["entradas"], err = json.Marshal(entradas)
	if err != nil {
		t.Fatal(err)
	}
	exterior["origen"], err = json.Marshal(origen)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(exterior)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOrigenCTAuditadoDistingueProyeccionHistoricaAusenciaYActoObsoleto(t *testing.T) {
	_, sobre := origenAuditadoCanonicoPrueba(t)
	historica, err := decodificarRespuestaOrigenCTAuditado(sobre)
	if err != nil || historica.Origen.Entradas[0].ComprobacionOfrecimiento != nil {
		t.Fatalf("historia: %v", err)
	}
	ausencia, err := decodificarRespuestaOrigenCTAuditado(sobreOrigenConComprobacionPrueba(t, sobre, json.RawMessage(`{"version_esperada":0}`)))
	if err != nil || ausencia.Origen.Entradas[0].ComprobacionOfrecimiento == nil || ausencia.Origen.Entradas[0].ComprobacionOfrecimiento.VersionEsperada != 0 || validarLecturaOrigenCT(ausencia.Origen) != nil {
		t.Fatalf("ausencia acreditada: %v", err)
	}
	ref := "comprobacion:" + strings.Repeat("a", 64)
	acto := `{"version_esperada":2,"comprobacion_ref":"` + ref + `","recibo_ref":"recibo:` + ref + `","registrada_en":"2026-10-18T08:00:00+00:00","version_politica_ofertas":1,"resultado":"pendiente","motivo_clave":"antecedentes_incompletos","vigente":false}`
	salida, err := decodificarRespuestaOrigenCTAuditado(sobreOrigenConComprobacionPrueba(t, sobre, json.RawMessage(acto)))
	if err != nil || validarLecturaOrigenCT(salida.Origen) != nil {
		t.Fatalf("acto obsoleto: %v", err)
	}
	e := salida.Origen.Entradas[0].ComprobacionOfrecimiento
	if e == nil || e.VersionEsperada != 2 || e.Vigente == nil || *e.Vigente || e.ReciboRef != "recibo:"+ref {
		t.Fatalf("última versión histórica perdida: %+v", e)
	}
	paraVigente := strings.Replace(acto, `"vigente":false`, `"vigente":true`, 1)
	salida, err = decodificarRespuestaOrigenCTAuditado(sobreOrigenConComprobacionPrueba(t, sobre, json.RawMessage(paraVigente)))
	if err != nil || salida.Origen.Entradas[0].ComprobacionOfrecimiento == nil || !*salida.Origen.Entradas[0].ComprobacionOfrecimiento.Vigente {
		t.Fatalf("acto vigente: %v", err)
	}
}

func TestOrigenCTAuditadoRechazaEstadoNoMinimo(t *testing.T) {
	_, sobre := origenAuditadoCanonicoPrueba(t)
	for nombre, estado := range map[string]string{
		"nulo":                `null`,
		"ausencia con motivo": `{"version_esperada":0,"motivo_clave":"antecedentes_incompletos"}`,
		"dato personal":       `{"version_esperada":0,"actor_ref":"persona:uno"}`,
		"acto incompleto":     `{"version_esperada":1,"vigente":true}`,
	} {
		t.Run(nombre, func(t *testing.T) {
			_, err := decodificarRespuestaOrigenCTAuditado(sobreOrigenConComprobacionPrueba(t, sobre, json.RawMessage(estado)))
			if !errors.Is(err, ports.ErrOrigenLlamamientoCTNoDisponible) {
				t.Fatalf("estado no mínimo: %v", err)
			}
		})
	}
}
