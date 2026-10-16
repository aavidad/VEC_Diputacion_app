package ports_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestReservaSubidaLocalComprometePDFYActoPropio(t *testing.T) {
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		time.Date(2026, 10, 14, 8, 0, 0, 0, time.UTC), "per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.ReservaSubidaLocal{
		ActorRegistradoV2: actor, VinculoV2: vinculo,
		Acto: ports.ActoSubidaLocal{
			Recurso: vecdomain.RecursoAutorizable{Referencia: "ref:" + strings.Repeat("1", 64),
				ModuloID: "meritos", Tipo: "hecho"},
			PersonaRef: actor.Contexto.PersonaRef, Finalidad: "custodiar_evidencia_meritos",
			Clasificacion: "personal", VersionDueno: actor.Contexto.Instantanea.PersonaVersion,
		},
		AccionSubir:     ports.AccionMeritosComprobanteSubirLocal,
		AccionDescargar: ports.AccionMeritosComprobanteDescargarLocal,
		CargaID:         "ref:" + strings.Repeat("2", 64), CorrelacionRef: "correlacion_" + strings.Repeat("3", 32),
		ClaveHMAC:           "hmac-sha256:local:" + strings.Repeat("4", 64),
		HuellaSolicitudHMAC: "hmac-sha256:local:" + strings.Repeat("5", 64),
		HuellaPDFSHA256:     strings.Repeat("6", 64), Tamano: 4096, MIME: "application/pdf",
	}
	pre, err := ports.PreimagenReservaSubidaLocal(q)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(pre, &m) != nil || len(m) != 18 ||
		string(m["accion"]) != `"meritos.cartera.comprobante.subir_propio"` ||
		string(m["fase"]) != `"reservar"` || m["documento_id"] != nil {
		t.Fatalf("preimagen de reserva: %s", pre)
	}
	q.HuellaPDFSHA256 = strings.Repeat("7", 64)
	otra, err := ports.PreimagenReservaSubidaLocal(q)
	if err != nil || bytes.Equal(pre, otra) ||
		ports.HuellaPreimagenSubidaLocal(pre) == ports.HuellaPreimagenSubidaLocal(otra) {
		t.Fatal("se conservó la autorización al sustituir el PDF")
	}
	q.Acto.Recurso.ModuloID = "dietas"
	if _, err := ports.PreimagenReservaSubidaLocal(q); err == nil {
		t.Fatal("acto de otro módulo admitido")
	}
}
