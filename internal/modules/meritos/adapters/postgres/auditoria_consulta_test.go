package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func intentoConsultaPrueba() (vec.AuditEntry, vec.AuditEntry) {
	e := vec.AuditEntry{ActorID: "persona:prueba", ActorProfile: "perfil:prueba",
		Action: application.AccionConsultaPropia, ModuleID: "meritos", Purpose: application.FinalidadConsultaPropia,
		SubjectRef: "hecho:prueba", RuleRef: "motivo:prueba", AuthorizationRef: "decision:prueba",
		CorrelationRef: "corr_" + strings.Repeat("1", 32), Result: "denegada",
		OccurredAt: time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)}
	r := e
	suma := sha256.Sum256([]byte(e.AuthorizationRef + "|" + e.CorrelationRef))
	r.ID = "auditoria_intento_consulta:" + hex.EncodeToString(suma[:])
	r.Metadata = map[string]string{"tipo_evento": "intento_consumo_consulta_propia", "pdp_resultado": "concedida",
		"origen_resultado": "observacion_registrador", "registro_contexto_ref": "contexto:prueba",
		"contexto_actor_huella_sha256": strings.Repeat("b", 64), "concesion_huella_sha256": strings.Repeat("c", 64)}
	return e, r
}

func TestAuditoriaConsultaConfirmaYRecuperaElMismoIntento(t *testing.T) {
	entrada, esperado := intentoConsultaPrueba()
	tx := &consultaTxPrueba{raw: consultaJSONPrueba(t, esperado)}
	pool := &consultaPoolPrueba{tx: tx}
	audit, err := nuevaAuditoriaConsulta(pool)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		out, err := audit.AppendAudit(context.Background(), entrada)
		if err != nil || !reflect.DeepEqual(out, esperado) {
			t.Fatal("el intento o su recuperación no confirman el resultado exacto", err)
		}
	}
	if tx.commits != 2 || tx.rollbacks != 2 || pool.opciones.IsoLevel != pgx.Serializable ||
		pool.opciones.AccessMode != pgx.ReadWrite || tx.consulta != registrarIntentoConsulta ||
		!reflect.DeepEqual(tx.argumentos, []any{entrada.AuthorizationRef, entrada.CorrelationRef, entrada.Result}) {
		t.Fatal("la auditoría transporta identidad libre o no confirma la transacción nominal")
	}
}

func TestAuditoriaConsultaEntradaAjenaNoAlcanzaSQL(t *testing.T) {
	for _, caso := range []string{"accion", "finalidad", "correlacion", "resultado", "documento", "metadata", "identidad_extra"} {
		t.Run(caso, func(t *testing.T) {
			e, _ := intentoConsultaPrueba()
			switch caso {
			case "accion":
				e.Action = "meritos.hecho.declarar"
			case "finalidad":
				e.Purpose = "otra"
			case "correlacion":
				e.CorrelationRef = "corr_invalida"
			case "resultado":
				e.Result = "concedida"
			case "documento":
				e.DocumentRef = "documento:privado"
			case "metadata":
				e.Metadata = map[string]string{"dato": "privado"}
			case "identidad_extra":
				e.RepresentedSubjectID = "persona:otra"
			}
			pool := &consultaPoolPrueba{tx: &consultaTxPrueba{}}
			audit, _ := nuevaAuditoriaConsulta(pool)
			out, err := audit.AppendAudit(context.Background(), e)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || pool.llamadas != 0 || !reflect.DeepEqual(out, vec.AuditEntry{}) {
				t.Fatal("entrada no nominal alcanza SQL", err)
			}
		})
	}
}

func TestAuditoriaConsultaNoConfirmaIdentidadOConcesionSustituida(t *testing.T) {
	for _, caso := range []string{"id", "actor", "perfil", "hecho", "decision", "correlacion", "pdp_denegado", "material_extra", "documento", "resultado", "fecha", "trailing", "exceso"} {
		t.Run(caso, func(t *testing.T) {
			e, r := intentoConsultaPrueba()
			switch caso {
			case "id":
				r.ID = "auditoria_intento_consulta:otra"
			case "actor":
				r.ActorID = "persona:otra"
			case "perfil":
				r.ActorProfile = "perfil:otro"
			case "hecho":
				r.SubjectRef = "hecho:otro"
			case "decision":
				r.AuthorizationRef = "decision:otra"
			case "correlacion":
				r.CorrelationRef = "corr_" + strings.Repeat("2", 32)
			case "pdp_denegado":
				r.Metadata["pdp_resultado"] = "denegada"
			case "material_extra":
				r.Metadata["material_v3"] = "privado"
			case "documento":
				r.DocumentRef = "documento:privado"
			case "resultado":
				r.Result = "no_confirmado"
			case "fecha":
				r.OccurredAt = time.Time{}
			}
			raw, _ := json.Marshal(r)
			if caso == "trailing" {
				raw = append(raw, []byte(" {}")...)
			}
			if caso == "exceso" {
				raw = []byte(strings.Repeat(" ", 8193))
			}
			tx := &consultaTxPrueba{raw: raw}
			audit, _ := nuevaAuditoriaConsulta(&consultaPoolPrueba{tx: tx})
			out, err := audit.AppendAudit(context.Background(), e)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || tx.commits != 0 || tx.rollbacks != 1 || !reflect.DeepEqual(out, vec.AuditEntry{}) {
				t.Fatal("respuesta alterada de auditoría confirmada", err)
			}
		})
	}
}

func TestAuditoriaConsultaRechazoSQLYCommitInciertoNoAfirmanRegistro(t *testing.T) {
	for _, caso := range []string{"roles_mixtos", "decision_ausente", "replay_divergente", "commit"} {
		t.Run(caso, func(t *testing.T) {
			e, r := intentoConsultaPrueba()
			tx := &consultaTxPrueba{raw: consultaJSONPrueba(t, r)}
			if caso == "commit" {
				tx.errCommit = errors.New("diagnóstico privado")
			} else {
				codigo := "42501"
				if caso == "replay_divergente" {
					codigo = "23505"
				}
				tx.errConsulta = &pgconn.PgError{Code: codigo, Message: "diagnóstico privado"}
			}
			audit, _ := nuevaAuditoriaConsulta(&consultaPoolPrueba{tx: tx})
			out, err := audit.AppendAudit(context.Background(), e)
			if !errors.Is(err, ports.ErrConsultaNoDisponible) || tx.rollbacks != 1 || !reflect.DeepEqual(out, vec.AuditEntry{}) {
				t.Fatal("fallo afirma auditoría confirmada", err)
			}
		})
	}
}
