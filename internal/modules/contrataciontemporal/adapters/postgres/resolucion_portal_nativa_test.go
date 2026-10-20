package postgres

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestResolucionPortalFirmaDoceCamposYConservaErroresNominales(t *testing.T) {
	s := ports.SolicitudResolucionPortal{
		ClaveIdempotencia: "718f47a6-5d2b-4c10-9a11-1234567890ab",
		OrganizacionRef:   "organizacion:prueba", ExpedienteRef: "expediente:prueba",
		BolsaRef: "bolsa:prueba", LlamamientoRef: "llamamiento:prueba",
		ReciboEmisionRef: "recibo:emision:prueba", ParticipacionRef: "participacion:prueba",
		ReciboRespuestaRef: "recibo:respuesta:prueba", Respuesta: ports.RespuestaLlamamientoAceptada,
		RevisionRespuestaRRHH: true, RevisionPlazoRRHH: true, CriterioValidacionRef: "politica:revision:prueba",
	}
	m := MaterialResolucionPortal{Solicitud: s, Politica: ports.ReferenciaGobernadaComunicacionLlamamiento{
		Referencia: s.CriterioValidacionRef, Version: 1, HuellaSHA256: strings.Repeat("a", 64),
	}}
	r, canon, err := recursoResolucionPortal(m)
	var material map[string]map[string]json.RawMessage
	if err != nil || json.Unmarshal(canon, &material) != nil || len(material) != 2 ||
		len(material["Solicitud"]) != 12 || len(material["Politica"]) != 3 ||
		r.Referencia != s.ExpedienteRef || r.Ambitos["organizacion_ref"] != s.OrganizacionRef ||
		len(r.Atributos["material_sha256"]) != 64 {
		t.Fatal("material firmado divergente", err)
	}
	for _, campo := range []string{"BolsaRef", "LlamamientoRef", "ReciboEmisionRef", "ParticipacionRef", "ReciboRespuestaRef"} {
		if _, ok := material["Solicitud"][campo]; !ok {
			t.Fatal("falta vínculo nominal en material:", campo)
		}
	}
	for codigo, esperado := range map[string]error{
		"42883": ErrPersistenciaComunicacionLlamamientoNoDisponible,
		"P2490": ports.ErrSolicitudComunicacionLlamamientoInvalida,
		"P2491": ports.ErrClaveComunicacionLlamamientoUsada,
		"P2492": ports.ErrVersionComunicacionLlamamientoEnConflicto,
		"P2493": ports.ErrOperacionComunicacionLlamamientoDenegada,
	} {
		if err := normalizarErrorPortal(t.Context(), &pgconn.PgError{Code: codigo, Message: "dato privado"}); !errors.Is(err, esperado) ||
			strings.Contains(err.Error(), "dato privado") {
			t.Fatal("error SQL expuesto o mal clasificado:", codigo, err)
		}
	}
}
