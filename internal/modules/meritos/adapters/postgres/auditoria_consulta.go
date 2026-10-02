package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/meritos/application"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

const registrarIntentoConsulta = `SELECT vec_meritos.registrar_intento_consulta_propia_v1($1,$2,$3)`

var correlacionIntento = regexp.MustCompile(`^corr_[0-9a-f]{32}$`)
var huellaIntento = regexp.MustCompile(`^[0-9a-f]{64}$`)

// AuditoriaConsulta usa exclusivamente el pool del registrador nominal. SQL
// reconstruye la identidad desde la concesión original; no recibe actor libre.
type AuditoriaConsulta struct{ pool iniciador }

var _ ports.AuditoriaIntentos = (*AuditoriaConsulta)(nil)

func NuevaAuditoriaConsulta(pool *pgxpool.Pool) (*AuditoriaConsulta, error) {
	return nuevaAuditoriaConsulta(pool)
}

func nuevaAuditoriaConsulta(pool iniciador) (*AuditoriaConsulta, error) {
	if nulo(pool) {
		return nil, ports.ErrConsultaNoDisponible
	}
	return &AuditoriaConsulta{pool: pool}, nil
}

func entradaIntentoValida(e vec.AuditEntry) bool {
	return e.Action == application.AccionConsultaPropia && e.ModuleID == "meritos" &&
		e.Purpose == application.FinalidadConsultaPropia && (e.Result == "denegada" || e.Result == "no_confirmado") &&
		domain.ReferenciaValida(e.AuthorizationRef) && domain.ReferenciaValida(e.ActorID) &&
		domain.ReferenciaValida(e.ActorProfile) && domain.ReferenciaValida(e.SubjectRef) &&
		domain.ReferenciaValida(e.RuleRef) && correlacionIntento.MatchString(e.CorrelationRef) &&
		!e.OccurredAt.IsZero() && e.ID == "" && e.Seq == 0 && e.ObjectVersion == 0 &&
		len(e.ActorRoles) == 0 && len(e.Metadata) == 0 && e.RepresentedSubjectID == "" &&
		e.AuthMethod == "" && e.AuthAssurance == "" && e.ExpedienteRef == "" && e.DocumentRef == "" &&
		e.Reason == "" && e.BeforeHash == "" && e.AfterHash == "" && e.Signature == "" &&
		e.PrevSignature == "" && e.IntegrityAlgorithm == ""
}

func (a *AuditoriaConsulta) AppendAudit(ctx context.Context, entrada vec.AuditEntry) (vec.AuditEntry, error) {
	if ctx == nil || a == nil || nulo(a.pool) || !entradaIntentoValida(entrada) {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vec.AuditEntry{}, err
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || nulo(tx) {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','1s',true),set_config('statement_timeout','2s',true),set_config('idle_in_transaction_session_timeout','3s',true)`); err != nil {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	var raw []byte
	if err = tx.QueryRow(ctx, registrarIntentoConsulta, entrada.AuthorizationRef, entrada.CorrelationRef, entrada.Result).Scan(&raw); err != nil {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	resultado, err := leerIntentoConsulta(raw, entrada)
	if err != nil {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vec.AuditEntry{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	// Después de intentar COMMIT no reinterpretar el resultado durable por una
	// cancelación local. Un reintento exacto recupera la misma referencia SQL.
	return resultado, nil
}

func leerIntentoConsulta(raw []byte, esperado vec.AuditEntry) (vec.AuditEntry, error) {
	var e vec.AuditEntry
	if len(raw) == 0 || len(raw) > 8192 {
		return e, ports.ErrConsultaNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	suma := sha256.Sum256([]byte(esperado.AuthorizationRef + "|" + esperado.CorrelationRef))
	referencia := "auditoria_intento_consulta:" + hex.EncodeToString(suma[:])
	if d.Decode(&e) != nil || d.Decode(new(any)) != io.EOF ||
		e.ID != referencia ||
		e.ActorID != esperado.ActorID || e.ActorProfile != esperado.ActorProfile || e.SubjectRef != esperado.SubjectRef ||
		e.AuthorizationRef != esperado.AuthorizationRef || e.CorrelationRef != esperado.CorrelationRef ||
		e.RuleRef != esperado.RuleRef || e.Result != esperado.Result || e.Action != esperado.Action ||
		e.ModuleID != esperado.ModuleID || e.Purpose != esperado.Purpose || e.OccurredAt.IsZero() ||
		e.OccurredAt.Location() != time.UTC || e.OccurredAt.Nanosecond()%1000 != 0 ||
		len(e.Metadata) != 6 || e.Metadata["tipo_evento"] != "intento_consumo_consulta_propia" ||
		e.Metadata["pdp_resultado"] != "concedida" || e.Metadata["origen_resultado"] != "observacion_registrador" ||
		!domain.ReferenciaValida(e.Metadata["registro_contexto_ref"]) ||
		!huellaIntento.MatchString(e.Metadata["contexto_actor_huella_sha256"]) ||
		!huellaIntento.MatchString(e.Metadata["concesion_huella_sha256"]) {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	// No aceptar identidad adicional, documentos, diagnósticos o firma inventada.
	limpia := e
	limpia.ID, limpia.Metadata = "", nil
	if !entradaIntentoValida(limpia) {
		return vec.AuditEntry{}, ports.ErrConsultaNoDisponible
	}
	return e, nil
}
