package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestSerializarDecisionSolicitudLigadaV2PostgreSQLConservaCanonesExactos(t *testing.T) {
	t.Parallel()
	decision, motivo := decisionSolicitudLigadaV2PostgreSQLPrueba(t)
	decisionCanonica, motivoCanonico, err := serializarDecisionSolicitudLigadaV2PostgreSQL(
		decision,
		motivo,
	)
	if err != nil {
		t.Fatalf("serializar decision V2: %v", err)
	}
	esperadaDecision, err := ports.RepresentacionCanonicaDecisionAutorizacionReforzadaV2(decision)
	if err != nil || string(decisionCanonica) != string(esperadaDecision) {
		t.Fatalf("canon de decision divergente: %v", err)
	}
	esperadoMotivo, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(motivo)
	if err != nil || string(motivoCanonico) != string(esperadoMotivo) {
		t.Fatalf("canon de motivo divergente: %v", err)
	}
	var documento map[string]json.RawMessage
	if err = json.Unmarshal(decisionCanonica, &documento); err != nil || len(documento) != 34 {
		t.Fatalf("documento V2 no cerrado: claves=%d err=%v", len(documento), err)
	}
	for _, clave := range []string{
		"esquema_huella_solicitud", "solicitud_huella_sha256",
		"esquema_huella_motivo", "motivo_huella_sha256",
	} {
		if _, existe := documento[clave]; !existe {
			t.Fatalf("falta el compromiso V2 %q", clave)
		}
	}
	sumaMotivo := sha256.Sum256(motivoCanonico)
	if hex.EncodeToString(sumaMotivo[:]) != decision.MotivoHuellaSHA256 {
		t.Fatal("el motivo durable no coincide con el compromiso de la decision")
	}
}

func TestSerializarDecisionSolicitudLigadaV2PostgreSQLRechazaV1YMotivoAjeno(t *testing.T) {
	t.Parallel()
	decision, motivo := decisionSolicitudLigadaV2PostgreSQLPrueba(t)
	decisionV1 := decision
	decisionV1.EsquemaHuellaSolicitud = ""
	decisionV1.SolicitudHuellaSHA256 = ""
	decisionV1.EsquemaHuellaMotivo = ""
	decisionV1.MotivoHuellaSHA256 = ""
	decisionV1.CorrelacionRef = "correlacion:postgresql:autorizacion:v1"
	if _, _, err := serializarDecisionSolicitudLigadaV2PostgreSQL(
		decisionV1,
		motivo,
	); !errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("decision V1 aceptada por el registro V2: %v", err)
	}
	motivoAjeno := motivo
	motivoAjeno.EntradaClave = "motivo_ffffffffffffffffffffffffffffffff"
	if _, _, err := serializarDecisionSolicitudLigadaV2PostgreSQL(
		decision,
		motivoAjeno,
	); !errors.Is(err, ports.ErrEvidenciaUsoDecisionAutorizacionInvalida) {
		t.Fatalf("motivo ajeno aceptado: %v", err)
	}
}

func TestRegistroDecisionSolicitudLigadaV2PostgreSQLFallaCerradoSinDependencia(t *testing.T) {
	t.Parallel()
	decision, motivo := decisionSolicitudLigadaV2PostgreSQLPrueba(t)
	orden, err := ports.NuevaOrdenRegistroDecisionAutorizacionSolicitudLigadaV2(decision, motivo)
	if err != nil {
		t.Fatalf("crear orden: %v", err)
	}
	var almacen *AlmacenAutorizacion
	if err = almacen.RegistrarDecisionSolicitudLigadaV2SiInstantaneaVigente(
		context.Background(),
		orden,
	); !errors.Is(err, ports.ErrRegistroDecisionNoDisponible) {
		t.Fatalf("almacen nulo no fallo cerrado: %v", err)
	}
}

func decisionSolicitudLigadaV2PostgreSQLPrueba(
	t *testing.T,
) (domain.DecisionAutorizacion, domain.ReferenciaEntradaCatalogo) {
	t.Helper()
	decision := decisionAutorizacionPostgreSQLPrueba(t)
	motivo := domain.ReferenciaEntradaCatalogo{
		CatalogoID:           "motivos_autorizacion",
		CatalogoVersion:      3,
		CatalogoHuellaSHA256: strings.Repeat("9", 64),
		EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(motivo)
	if err != nil {
		t.Fatalf("huella de motivo: %v", err)
	}
	decision.CorrelacionRef = "correlacion_0123456789abcdef0123456789abcdef"
	decision.EsquemaHuellaSolicitud = domain.EsquemaHuellaSolicitudAutorizacionV2
	decision.SolicitudHuellaSHA256 = strings.Repeat("8", 64)
	decision.EsquemaHuellaMotivo = domain.EsquemaHuellaMotivoAutorizacionV2
	decision.MotivoHuellaSHA256 = huellaMotivo
	if err = decision.ValidarEvidenciaInstantaneaSolicitudLigadaV2(); err != nil {
		t.Fatalf("decision V2 de prueba: %v", err)
	}
	return decision, motivo
}
