package ports_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestParejaSubidaLocalBolsaNoCruzaModulos(t *testing.T) {
	if !ports.ParejaActoSubidaLocalValida("bolsa", "justificante_renuncia",
		"custodiar_justificante_renuncia_bolsa",
		ports.AccionBolsaJustificanteRenunciaSubirLocal,
		ports.AccionBolsaJustificanteRenunciaDescargarLocal) {
		t.Fatal("pareja nominal Bolsa denegada")
	}
	if ports.ParejaActoSubidaLocalValida("meritos", "hecho", "custodiar_evidencia_meritos",
		ports.AccionBolsaJustificanteRenunciaSubirLocal,
		ports.AccionBolsaJustificanteRenunciaDescargarLocal) ||
		ports.ParejaActoSubidaLocalValida("bolsa", "justificante_renuncia",
			"custodiar_justificante_renuncia_bolsa",
			ports.AccionBolsaJustificanteRenunciaSubirLocal,
			ports.AccionBolsaJustificanteRenunciaDescargarRRHH) {
		t.Fatal("acción ajena o RRHH aceptada como descarga propia")
	}
}

func TestPreimagenRRHHLigaActorTitularYParticipacionSinCargaConocida(t *testing.T) {
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		time.Date(2026, 10, 15, 8, 0, 0, 0, time.UTC), "per_0123456789abcdefghijkl",
		"prf_0123456789abcdefghijkl", vecdomain.AuthMethodCertificate, vecdomain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.SolicitudOriginalSubidaLocalRRHH{
		ActorRegistradoV2: actor, VinculoV2: vinculo,
		DocumentoID: "ref:" + strings.Repeat("1", 64), HuellaPDFSHA256: strings.Repeat("a", 64),
		RecursoRef: "ref:" + strings.Repeat("2", 64), PersonaTitularRef: "per_abcdefghijklmnopqrstuvwxyz",
		BolsaRef: "bolsa_123", ParticipacionRef: "part_123",
		CorrelacionRef: "correlacion_" + strings.Repeat("3", 32),
	}
	pre, err := ports.PreimagenConsultaSubidaLocalRRHH(q)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if json.Unmarshal(pre, &m) != nil || len(m) != 13 || m["carga_id"] != "" ||
		m["accion"] != ports.AccionBolsaJustificanteRenunciaDescargarRRHH ||
		m["fase"] != "abrir_rrhh" || m["persona_ref"] != q.PersonaTitularRef ||
		m["actor_ref"] != actor.Contexto.PersonaRef || m["participacion_ref"] != q.ParticipacionRef {
		t.Fatalf("preimagen RRHH: %s", pre)
	}
	q.ParticipacionRef = ""
	if _, err := ports.PreimagenConsultaSubidaLocalRRHH(q); err == nil {
		t.Fatal("se aceptó relación Bolsa incompleta")
	}
}
