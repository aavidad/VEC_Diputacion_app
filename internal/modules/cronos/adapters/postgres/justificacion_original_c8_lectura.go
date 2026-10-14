package postgres

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

const (
	consultaDetalleOriginalC8 = `SELECT vec_cronos_v1.consultar_justificacion_original_c8_v1($1,$2,$3,$4::jsonb)`
	consultaBandejaOriginalC8 = `SELECT vec_cronos_v1.consultar_bandeja_justificaciones_c8_v1($1,$2,$3::jsonb)`
)

var huellaLecturaOriginalC8 = regexp.MustCompile(`^[0-9a-f]{64}$`)
var reciboLecturaOriginalC8 = regexp.MustCompile(`^recibo:cronos:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

type LectorOriginalC8PostgreSQL struct{ db *pgxpool.Pool }

func NuevoLectorOriginalC8PostgreSQL(db *pgxpool.Pool) (*LectorOriginalC8PostgreSQL, error) {
	if db == nil {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &LectorOriginalC8PostgreSQL{db: db}, nil
}

func capturaOriginalC8Valida(c ports.CapturaLecturaOriginalC8, accion, recurso string) (string, bool) {
	if len(c.ContextoCanonico) == 0 || len(c.ContextoCanonico) > 65536 ||
		len(c.VinculoCanonico) == 0 || len(c.VinculoCanonico) > 16384 ||
		len(c.Orden) == 0 || len(c.Orden) > 65536 || c.ValidaHasta.IsZero() || !time.Now().Before(c.ValidaHasta) {
		return "", false
	}
	var orden map[string]json.RawMessage
	if json.Unmarshal(c.Orden, &orden) != nil || orden["titular_persona_ref"] != nil || orden["empleado_objetivo_ref"] != nil {
		return "", false
	}
	var ref, act, correlacion string
	if json.Unmarshal(orden["recurso_ref"], &ref) != nil || ref != recurso ||
		json.Unmarshal(orden["accion"], &act) != nil || act != accion ||
		json.Unmarshal(orden["correlacion_ref"], &correlacion) != nil || correlacion == "" {
		return "", false
	}
	return correlacion, true
}

func (r *LectorOriginalC8PostgreSQL) leer(ctx context.Context, consulta string,
	validar func([]byte) error, args ...any) error {
	if r == nil || r.db == nil || ctx == nil || ctx.Err() != nil || validar == nil {
		return ports.ErrJustificacionNoDisponible
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return ports.ErrJustificacionNoDisponible
	}
	defer func() {
		rollback, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = tx.Rollback(rollback)
	}()
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('timezone','UTC',true),
    set_config('row_security','on',true),set_config('lock_timeout','2s',true),
    set_config('statement_timeout','10s',true),set_config('idle_in_transaction_session_timeout','15s',true)`); err != nil {
		return ports.ErrJustificacionNoDisponible
	}
	var bruto []byte
	if err = tx.QueryRow(ctx, consulta, args...).Scan(&bruto); err != nil {
		return errorOriginalC8(ctx, err)
	}
	defer clear(bruto)
	if len(bruto) == 0 || len(bruto) > 1<<20 {
		return ports.ErrJustificacionNoDisponible
	}
	if err = validar(bruto); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return ports.ErrJustificacionNoDisponible
	}
	return nil
}

func (r *LectorOriginalC8PostgreSQL) ConsultarOriginal(ctx context.Context,
	c ports.CapturaLecturaOriginalC8, solicitudRef string) (ports.LecturaOriginalC8, error) {
	var cero ports.LecturaOriginalC8
	if !domain.SolicitudPermisoRefValida(solicitudRef) {
		return cero, domain.ErrJustificacionInvalida
	}
	var orden map[string]json.RawMessage
	if json.Unmarshal(c.Orden, &orden) != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	var accion string
	if json.Unmarshal(orden["accion"], &accion) != nil ||
		accion != "cronos.justificacion.propia.consultar" && accion != "cronos.justificacion.rrhh.consultar" {
		return cero, ports.ErrJustificacionNoDisponible
	}
	correlacion, ok := capturaOriginalC8Valida(c, accion, solicitudRef)
	if !ok {
		return cero, ports.ErrJustificacionNoDisponible
	}
	var sql struct {
		SolicitudRef        string                           `json:"solicitud_ref"`
		EmpleadoRef         string                           `json:"empleado_ref"`
		PersonaTitularRef   string                           `json:"persona_titular_ref"`
		EstadoPermiso       domain.EstadoSolicitudPermiso    `json:"estado_permiso"`
		VersionPermiso      int64                            `json:"version_permiso"`
		PendienteJustificar bool                             `json:"pendiente_justificar"`
		JustificanteExigido bool                             `json:"justificante_exigido"`
		Historia            []ports.HechoOriginalC8          `json:"historia"`
		Auditoria           ports.AuditoriaLecturaOriginalC8 `json:"auditoria"`
	}
	err := r.leer(ctx, consultaDetalleOriginalC8, func(bruto []byte) error {
		if decodificarEstricto(bruto, &sql) != nil || sql.SolicitudRef != solicitudRef || sql.EmpleadoRef == "" ||
			sql.PersonaTitularRef == "" || sql.EstadoPermiso != domain.EstadoPermisoConcedido ||
			sql.VersionPermiso < 1 || !sql.PendienteJustificar || !sql.JustificanteExigido || sql.Historia == nil ||
			len(sql.Historia) > 500 || !auditoriaOriginalC8Valida(sql.Auditoria, correlacion) {
			return ports.ErrJustificacionNoDisponible
		}
		for i, h := range sql.Historia {
			if h.Version != int64(i+1) || !domain.RefDocumentoJustificacionValida(h.DocumentoID) ||
				h.DocumentoVersion == 0 || !domain.HuellaEfectosValida(h.SHA256) ||
				h.ReciboDocumentoRef == "" || h.CustodioID == "" || !reciboLecturaOriginalC8.MatchString(h.ReciboRef) ||
				h.InstanteUTC.IsZero() || h.InstanteUTC.Nanosecond()%1000 != 0 ||
				(h.Estado != domain.JustificacionPendiente && h.Estado != domain.JustificacionAceptada && h.Estado != domain.JustificacionRechazada) {
				return ports.ErrJustificacionNoDisponible
			}
		}
		return nil
	}, solicitudRef, c.ContextoCanonico, c.VinculoCanonico, string(c.Orden))
	if err != nil {
		return cero, err
	}
	return ports.LecturaOriginalC8{SolicitudRef: sql.SolicitudRef, EmpleadoRef: sql.EmpleadoRef,
		PersonaTitularRef: sql.PersonaTitularRef, EstadoPermiso: sql.EstadoPermiso, VersionPermiso: sql.VersionPermiso,
		PendienteJustificar: sql.PendienteJustificar, JustificanteExigido: sql.JustificanteExigido,
		Historia: sql.Historia, Auditoria: sql.Auditoria}, nil
}

func (r *LectorOriginalC8PostgreSQL) ConsultarBandejaOriginal(ctx context.Context,
	c ports.CapturaLecturaOriginalC8) (ports.BandejaOriginalC8, error) {
	var cero ports.BandejaOriginalC8
	correlacion, ok := capturaOriginalC8Valida(c, "cronos.justificacion.bandeja.consultar", "bandeja:cronos:justificaciones")
	if !ok {
		return cero, ports.ErrJustificacionNoDisponible
	}
	var sql struct {
		Pendientes []ports.PendienteRevisionOriginalC8 `json:"pendientes"`
		Auditoria  ports.AuditoriaLecturaOriginalC8    `json:"auditoria"`
	}
	err := r.leer(ctx, consultaBandejaOriginalC8, func(bruto []byte) error {
		if decodificarEstricto(bruto, &sql) != nil || sql.Pendientes == nil || len(sql.Pendientes) > 500 ||
			!auditoriaOriginalC8Valida(sql.Auditoria, correlacion) {
			return ports.ErrJustificacionNoDisponible
		}
		for _, p := range sql.Pendientes {
			if !domain.SolicitudPermisoRefValida(p.SolicitudRef) || !domain.RefDocumentoJustificacionValida(p.DocumentoID) ||
				p.DocumentoVersion == 0 || !domain.HuellaEfectosValida(p.SHA256) ||
				p.Estado != domain.JustificacionPendiente || p.Version < 1 || p.RegistradaEn.IsZero() {
				return ports.ErrJustificacionNoDisponible
			}
		}
		return nil
	}, c.ContextoCanonico, c.VinculoCanonico, string(c.Orden))
	if err != nil {
		return cero, err
	}
	return ports.BandejaOriginalC8{Pendientes: sql.Pendientes, Auditoria: sql.Auditoria}, nil
}

func auditoriaOriginalC8Valida(a ports.AuditoriaLecturaOriginalC8, correlacion string) bool {
	fecha, err := time.Parse(time.RFC3339Nano, a.RegistradaEn)
	return err == nil && !fecha.IsZero() && a.Referencia != "" &&
		huellaLecturaOriginalC8.MatchString(a.HuellaSHA256) && a.CorrelacionRef == correlacion
}
