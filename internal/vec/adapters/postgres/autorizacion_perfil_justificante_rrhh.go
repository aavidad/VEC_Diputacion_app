package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type falloSeleccionPerfilJustificante struct {
	etapa string
	causa error
}

// DiagnosticoSelectorPerfilJustificante expone solo etapa y SQLSTATE, nunca
// Message, Detail, Hint ni la cadena de conexión del error original.
func (f falloSeleccionPerfilJustificante) DiagnosticoSelectorPerfilJustificante() (string, string) {
	var pgErr *pgconn.PgError
	if errors.As(f.causa, &pgErr) && len(pgErr.Code) == 5 {
		return f.etapa, "sqlstate_" + pgErr.Code
	}
	return f.etapa, "error_operacion"
}

func (f falloSeleccionPerfilJustificante) Error() string {
	return fmt.Sprintf("vec: selector de perfil documental no disponible (%s)", f.etapa)
}
func (f falloSeleccionPerfilJustificante) Unwrap() []error {
	if f.causa == nil {
		return []error{ports.ErrFuenteAutorizacionNoDisponible}
	}
	return []error{ports.ErrFuenteAutorizacionNoDisponible, f.causa}
}

func huellaPerfilJustificanteValida(v string) bool {
	if len(v) != 64 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil && strings.ToLower(v) == v
}

// SeleccionarPerfilJustificanteBolsaRRHH usa el mismo LOGIN de fuente AUT.
// La fachada solo devuelve un perfil propio único y su apunte común.
func (a *AlmacenAutorizacion) SeleccionarPerfilJustificanteBolsaRRHH(
	ctx context.Context, q ports.SolicitudPerfilJustificanteBolsaRRHH,
) (ports.SeleccionPerfilJustificanteBolsaRRHH, error) {
	vacio := ports.SeleccionPerfilJustificanteBolsaRRHH{}
	if a == nil || a.externo || valorNuloPostgreSQL(a.pool) || ctx == nil || ctx.Err() != nil {
		return vacio, falloSeleccionPerfilJustificante{etapa: "dependencias"}
	}
	if !strings.HasPrefix(q.AutenticacionRef, "aut_") ||
		!strings.HasPrefix(q.SesionRef, "ses_") ||
		!strings.HasPrefix(q.CuentaConfirmadaRef, "cta_") ||
		!strings.HasPrefix(q.ControlSesionRef, "cse_") ||
		q.ControlSesionRevision == 0 ||
		!huellaPerfilJustificanteValida(q.ControlSesionSHA256) ||
		!huellaPerfilJustificanteValida(q.AutenticacionSHA256) ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(q.CorrelacionRef) {
		return vacio, domain.ErrAutorizacionDenegada
	}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacio, falloSeleccionPerfilJustificante{etapa: "transaccion", causa: err}
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return vacio, falloSeleccionPerfilJustificante{etapa: "configuracion", causa: err}
	}
	var raw []byte
	err = tx.QueryRow(ctx, `SELECT vec_autorizacion.seleccionar_perfil_justificante_rrhh_v1(
		$1,$2,$3,$4,$5::numeric,$6,$7,$8)`, q.AutenticacionRef, q.SesionRef,
		q.CuentaConfirmadaRef, q.ControlSesionRef, q.ControlSesionRevision,
		q.ControlSesionSHA256, q.AutenticacionSHA256, q.CorrelacionRef).Scan(&raw)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			return vacio, domain.ErrAutorizacionDenegada
		}
		return vacio, falloSeleccionPerfilJustificante{etapa: "consulta", causa: err}
	}
	var r struct {
		Esquema         string `json:"esquema"`
		PerfilActivoRef string `json:"perfil_activo_ref"`
		AuditoriaRef    string `json:"auditoria_ref"`
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) == 0 || len(raw) > 1024 || d.Decode(&r) != nil {
		return vacio, falloSeleccionPerfilJustificante{etapa: "respuesta"}
	}
	var extra any
	if d.Decode(&extra) != io.EOF || r.Esquema != "vec.autorizacion.perfil-justificante-rrhh.v1" ||
		!strings.HasPrefix(r.AuditoriaRef, "aud_v3_b_") ||
		(r.PerfilActivoRef != "" && !strings.HasPrefix(r.PerfilActivoRef, "prf_")) {
		return vacio, falloSeleccionPerfilJustificante{etapa: "acuse"}
	}
	if err = tx.Commit(ctx); err != nil {
		return vacio, falloSeleccionPerfilJustificante{etapa: "confirmacion", causa: err}
	}
	return ports.SeleccionPerfilJustificanteBolsaRRHH{
		PerfilActivoRef: r.PerfilActivoRef, AuditoriaRef: r.AuditoriaRef,
	}, nil
}

var _ ports.SeleccionadorPerfilJustificanteBolsaRRHH = (*AlmacenAutorizacion)(nil)
