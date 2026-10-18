package postgres

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestNormalizarInstantesEmisionLlamamientoConservaInstanteYPrecision(t *testing.T) {
	instante := time.Date(2026, 10, 16, 0, 37, 15, 121014000, time.UTC)
	for nombre, ubicacion := range map[string]*time.Location{
		"offset_cero":      time.FixedZone("UTC+0", 0),
		"offset_dos_horas": time.FixedZone("CEST", 2*60*60),
	} {
		t.Run(nombre, func(t *testing.T) {
			emision := ports.EmisionLlamamiento{
				EmitidoEn: instante.In(ubicacion),
				OrigenCT:  &ports.ReciboOrigenEmisionCT{EmitidoEn: instante.In(ubicacion)},
			}
			normalizarInstantesEmisionLlamamiento(&emision)
			for campo, obtenido := range map[string]time.Time{
				"emision": emision.EmitidoEn,
				"origen":  emision.OrigenCT.EmitidoEn,
			} {
				if obtenido.Location() != time.UTC || !obtenido.Equal(instante) || obtenido.Nanosecond() != 121014000 {
					t.Fatalf("%s: cambió la ubicación, el instante o los microsegundos: %v", campo, obtenido)
				}
			}
		})
	}
	var ausente *ports.EmisionLlamamiento
	normalizarInstantesEmisionLlamamiento(ausente)
	normalizarInstanteOrigenCT(nil)
}

func TestDecodificarReservaLlamamientoCTNormalizaFechaPostgreSQL(t *testing.T) {
	origen, _ := origenAuditadoCanonicoPrueba(t)
	instante := time.Date(2026, 10, 16, 0, 37, 15, 121014000, time.UTC)
	emision := ports.EmisionLlamamiento{
		LlamamientoRef: "llamamiento:ct:prueba", ReciboRef: "recibo:ct:prueba",
		BolsaRef: origen.BolsaRef, Participaciones: []string{origen.Entradas[0].ParticipacionRef},
		EmitidoEn: instante,
		OrigenCT: &ports.ReciboOrigenEmisionCT{
			OrigenEmisionCT: ports.OrigenEmisionCT{
				ExpedienteRef: "expediente:ct:prueba", ClaveCT: "clave:ct:prueba",
				UnidadRef: "unidad:ct:prueba", BolsaRef: origen.BolsaRef,
				ParticipacionRef: origen.Entradas[0].ParticipacionRef, Lectura: origen,
			},
			LlamamientoRef: "llamamiento:ct:prueba", ReciboRef: "recibo:ct:prueba",
			EmitidoEn: instante,
		},
	}
	raw, err := json.Marshal(emision)
	if err != nil {
		t.Fatal(err)
	}
	// PostgreSQL serializa timestamptz UTC con +00:00; Go emite Z.
	raw = bytes.ReplaceAll(raw, []byte(`"2026-10-16T00:37:15.121014Z"`), []byte(`"2026-10-16T00:37:15.121014+00:00"`))
	if bytes.Count(raw, []byte(`"2026-10-16T00:37:15.121014+00:00"`)) != 2 {
		t.Fatal("la prueba no reprodujo ambas fechas PostgreSQL")
	}
	decodificada, err := decodificarReservaLlamamientoCT(raw, emision.OrigenCT.ExpedienteRef, emision.OrigenCT.ClaveCT)
	if err != nil {
		t.Fatalf("reserva CT válida rechazada: %v", err)
	}
	if !decodificada.Reutilizada || decodificada.EmitidoEn.Location() != time.UTC ||
		decodificada.OrigenCT.EmitidoEn.Location() != time.UTC ||
		!decodificada.EmitidoEn.Equal(instante) || !decodificada.OrigenCT.EmitidoEn.Equal(instante) ||
		decodificada.EmitidoEn.Nanosecond() != instante.Nanosecond() ||
		!bytes.Equal(decodificada.OrigenCT.Lectura.BolsaCanonica, origen.BolsaCanonica) ||
		!bytes.Equal(decodificada.OrigenCT.Lectura.InstantaneaCanonica, origen.InstantaneaCanonica) {
		t.Fatal("la reserva no conservó instante UTC, microsegundos o canónicos")
	}
}
