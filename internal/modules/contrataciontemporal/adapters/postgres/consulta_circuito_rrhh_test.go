package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCanonConsultaCircuitoRRHHConservaMaterialPactadoConSQL(t *testing.T) {
	s := ports.SolicitudConsultaCircuitoRRHH{
		AutenticacionRef: "aut_aaaaaaaaaaaaaaaaaaaaaaaa",
		SesionRef:        "ses_bbbbbbbbbbbbbbbbbbbbbbbb",
		PerfilRef:        "prf_cccccccccccccccccccccccc",
		OrganizacionRef:  "organizacion:dipgra:circuito:001",
		ExpedienteRef:    "expediente:ct:circuito:001",
		VersionObservada: 3,
	}
	contenido, err := canonConsultaCircuitoRRHH(s)
	esperado := `{"expediente_ref":"expediente:ct:circuito:001","organizacion_ref":"organizacion:dipgra:circuito:001","version_expediente":3}`
	if err != nil || string(contenido) != esperado {
		t.Fatalf("material SQL divergente: %s (%v)", contenido, err)
	}
	recurso, err := RecursoConsultaCircuitoRRHH(s)
	if err != nil || recurso.Tipo != TipoRecursoConsultaCircuitoRRHH ||
		recurso.Referencia != s.ExpedienteRef || recurso.Ambitos["organizacion_ref"] != s.OrganizacionRef ||
		len(recurso.Atributos["material_sha256"]) != 64 {
		t.Fatalf("recurso V3 divergente: %+v (%v)", recurso, err)
	}
	materialHuella := sha256.Sum256(contenido)
	contexto := `{"ambitos":{"organizacion_ref":"organizacion:dipgra:circuito:001"},"atributos":{"material_sha256":"` + hex.EncodeToString(materialHuella[:]) + `"}}`
	contextoHuella := sha256.Sum256([]byte(contexto))
	calculada, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil || calculada != hex.EncodeToString(contextoHuella[:]) {
		t.Fatalf("huella de recurso V3 distinta de SQL: %s (%v)", calculada, err)
	}
}
