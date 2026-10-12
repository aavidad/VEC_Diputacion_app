package ports_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

func TestPreimagenCargaAdmitidaCustodiaSeparaFaseYObjetoExacto(t *testing.T) {
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(
		time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC),
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.ConsultaCargaAdmitidaCustodiaV3{
		Fase: "consulta", CargaID: "ref:" + strings.Repeat("1", 64),
		RecursoRef: "ref:" + strings.Repeat("2", 64), PersonaRef: actor.Contexto.PersonaRef,
		VersionEsperada: 5, ActorRegistradoV2: actor, VinculoV2: vinculo,
		Finalidad: "custodiar_evidencia_meritos", CorrelacionRef: "corr:0123456789abcdef",
	}
	consulta, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q)
	if err != nil {
		t.Fatal(err)
	}
	q.Fase, q.PlanRef = "abrir", "plan:ingesta:0123456789abcdef"
	q.HuellaCargaEsperadaSHA256 = strings.Repeat("a", 64)
	q.ObjetoRef, q.ObjetoVersion = "objeto:admitido:uno", "version:1"
	q.HuellaContenidoSHA256, q.Tamano, q.MIME = strings.Repeat("b", 64), 100, "application/pdf"
	q.EvidenciaAnalisisRef = "evidencia:antivirus:uno"
	abrir, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q)
	if err != nil || bytes.Equal(consulta, abrir) {
		t.Fatal("la fase de apertura reutilizó la preimagen de consulta")
	}
	q.ObjetoVersion = "version:ajena"
	otraVersion, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q)
	if err != nil || bytes.Equal(abrir, otraVersion) {
		t.Fatal("la versión S3 no quedó ligada")
	}
	q.ObjetoVersion = "version:1"
	q.PersonaRef = "per_otra_persona_0123456789"
	if _, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q); err == nil {
		t.Fatal("se aceptó identidad ajena")
	}
	q.PersonaRef = actor.Contexto.PersonaRef
	q.PlanRef = ""
	if _, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q); err == nil {
		t.Fatal("la fase abrir no exigió plan")
	}
}

func TestConsultaCargaAdmitidaCustodiaDeniegaV3AusenteCaducadaYPersonaAjena(t *testing.T) {
	ahora := time.Date(2026, 10, 11, 8, 0, 0, 0, time.UTC)
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(ahora,
		"per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl",
		domain.AuthMethodCertificate, domain.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	q := ports.ConsultaCargaAdmitidaCustodiaV3{Fase: "consulta", CargaID: "ref:" + strings.Repeat("1", 64),
		RecursoRef: "ref:" + strings.Repeat("2", 64), PersonaRef: actor.Contexto.PersonaRef,
		VersionEsperada: 5, ActorRegistradoV2: actor, VinculoV2: vinculo,
		Finalidad: "custodiar_evidencia_meritos", CorrelacionRef: "corr:0123456789abcdef"}
	pre, err := ports.PreimagenConsultaCargaAdmitidaCustodiaV3(q)
	if err != nil || q.Validar(ahora) == nil {
		t.Fatal("consulta sin V3 admitida")
	}
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:custodia:0123456789abcdef", strings.Repeat("1", 64), strings.Repeat("2", 64),
		"contexto:custodia:0123456789abcdef", strings.Repeat("3", 64),
		ports.AccionCargaAdmitidaConsultarCustodiaV3, q.CargaID, ports.HuellaPreimagenCargaDocumentalV3(pre),
		ports.AudienciaCargaAdmitidaConsultarCustodiaV3, ahora.Add(-4*time.Second), ahora.Add(-time.Second))
	if err != nil {
		t.Fatal(err)
	}
	spki, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	material, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("c", 512)), resumen, []byte("decision"), []byte("motivo"),
		[]byte("contexto"), 3, 4, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki)
	if err != nil {
		t.Fatal(err)
	}
	q.AutorizacionV3 = ports.AutorizacionCargaDocumentalV3{Material: material,
		Accion: ports.AccionCargaAdmitidaConsultarCustodiaV3, RecursoRef: q.CargaID, AmbitoRef: q.RecursoRef,
		PersonaRef: q.PersonaRef, PerfilActivoRef: actor.Contexto.PerfilActivoRef,
		Finalidad: q.Finalidad, CorrelacionRef: q.CorrelacionRef}
	if q.Validar(ahora) == nil {
		t.Fatal("decisión caducada admitida")
	}
	q.AutorizacionV3.PersonaRef = "per_ajena0123456789abcdefghijkl"
	if q.Validar(ahora.Add(-2*time.Second)) == nil {
		t.Fatal("decisión de otra persona admitida")
	}
}
