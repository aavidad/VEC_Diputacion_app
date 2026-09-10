package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dom "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	lector "vec-diputacion-granada/internal/modules/personal/adapters/lecturaincorporacion"
)

const consultaLocalizadorSolicitudAltaEjercicio = `SELECT solicitud_json::text, resultado_ref, recibo_ref, relacion_ref, ocupacion_ref, material_sha256 FROM vec_personal.localizar_solicitud_alta_ejercicio_v1($1::text,$2::text,$3::text)`
const ajustesLocalizadorSolicitudAltaEjercicio = `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','5s',true)`

var ErrLocalizadorSolicitudAltaNoDisponible = errors.New("personal: localizacion de solicitud no disponible")

// SolicitudLocalizadaAltaEjercicio contiene metadatos del propietario, no un
// RegistroPersonal acreditado. La lectura V2 con permiso propio sigue obligada.
type SolicitudLocalizadaAltaEjercicio struct {
	Solicitud ct.SolicitudAltaPersonalRPT
	Selector  lector.Selector
}

type LocalizadorSolicitudAltaEjercicioPostgreSQL struct {
	pool  iniciadorTransaccionAlta
	reloj ct.Reloj
}

// Pool dedicado del localizador: sin SELECT de tablas ni roles CT. El llamador
// comprueba el acceso actual antes de divulgar solicitud, referencias o hashes.
func NuevoLocalizadorSolicitudAltaEjercicioPostgreSQL(p *pgxpool.Pool, r ct.Reloj) (*LocalizadorSolicitudAltaEjercicioPostgreSQL, error) {
	return nuevoLocalizadorSolicitudAltaEjercicio(p, r)
}
func nuevoLocalizadorSolicitudAltaEjercicio(p iniciadorTransaccionAlta, r ct.Reloj) (*LocalizadorSolicitudAltaEjercicioPostgreSQL, error) {
	if dependenciaNulaAlta(p) || dependenciaNulaAlta(r) {
		return nil, ErrLocalizadorSolicitudAltaNoDisponible
	}
	return &LocalizadorSolicitudAltaEjercicioPostgreSQL{pool: p, reloj: r}, nil
}
func errorLocalizadorSolicitudAlta(ctx context.Context, e error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(e, context.Canceled) {
		return context.Canceled
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return context.DeadlineExceeded
	}
	return ErrLocalizadorSolicitudAltaNoDisponible
}

func (l *LocalizadorSolicitudAltaEjercicioPostgreSQL) Localizar(ctx context.Context, organizacion, expediente, solicitud string) (SolicitudLocalizadaAltaEjercicio, bool, error) {
	var cero SolicitudLocalizadaAltaEjercicio
	if l == nil || ctx == nil || dependenciaNulaAlta(l.pool) || dependenciaNulaAlta(l.reloj) {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, nil)
	}
	for _, ref := range []string{organizacion, expediente, solicitud} {
		if !dom.ReferenciaOpacaValida(ref) {
			return cero, false, errorLocalizadorSolicitudAlta(ctx, nil)
		}
	}
	var ultimo time.Time
	validar := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		ahora := l.reloj.Ahora()
		if err := ctx.Err(); err != nil {
			return err
		}
		if !dom.InstanteUTCCanonico(ahora) || ahora.Before(ultimo) {
			return ErrLocalizadorSolicitudAltaNoDisponible
		}
		ultimo = ahora
		return ctx.Err()
	}
	if err := validar(); err != nil {
		return cero, false, err
	}
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	confirmado := false
	if !dependenciaNulaAlta(tx) {
		defer func() {
			if !confirmado {
				revertirTransaccionAlta(tx)
			}
		}()
	}
	if err != nil || dependenciaNulaAlta(tx) {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	if _, err = tx.Exec(ctx, ajustesLocalizadorSolicitudAltaEjercicio); err != nil {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	rows, err := tx.Query(ctx, consultaLocalizadorSolicitudAltaEjercicio, organizacion, expediente, solicitud)
	if !dependenciaNulaAlta(rows) {
		defer rows.Close()
	}
	if err != nil || dependenciaNulaAlta(rows) {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
	}
	if err = validar(); err != nil {
		return cero, false, err
	}
	var resultado SolicitudLocalizadaAltaEjercicio
	encontrado := rows.Next()
	if encontrado {
		var raw []byte
		s := lector.Selector{OrganizacionRef: organizacion, ExpedienteRef: expediente, SolicitudRef: solicitud}
		if err = rows.Scan(&raw, &s.ResultadoRef, &s.ReciboRef, &s.RelacionRef, &s.OcupacionRef, &s.MaterialSHA256); err != nil {
			return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
		}
		if err = validar(); err != nil {
			return cero, false, err
		}
		original, e := decodificarSolicitudLocalizadaAlta(raw)
		if e != nil || original.ExpedienteRef != expediente || original.SolicitudRef != solicitud || !patronHuellaAlta.MatchString(s.MaterialSHA256) {
			return cero, false, ErrLocalizadorSolicitudAltaNoDisponible
		}
		for _, ref := range []string{s.ResultadoRef, s.ReciboRef, s.RelacionRef, s.OcupacionRef} {
			if !dom.ReferenciaOpacaValida(ref) {
				return cero, false, ErrLocalizadorSolicitudAltaNoDisponible
			}
		}
		s.VersionExpediente = original.VersionExpediente
		resultado = SolicitudLocalizadaAltaEjercicio{Solicitud: original, Selector: s}
		if rows.Next() {
			return cero, false, ErrLocalizadorSolicitudAltaNoDisponible
		}
	}
	if err = rows.Err(); err != nil {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
	}
	rows.Close()
	if err = validar(); err != nil {
		return cero, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return cero, false, errorLocalizadorSolicitudAlta(ctx, err)
	}
	confirmado = true
	if err = validar(); err != nil {
		return cero, false, err
	}
	return resultado, encontrado, nil
}

func decodificarSolicitudLocalizadaAlta(raw []byte) (ct.SolicitudAltaPersonalRPT, error) {
	var s ct.SolicitudAltaPersonalRPT
	if len(raw) == 0 || len(raw) > 16<<10 || !utf8.Valid(raw) {
		return s, ErrLocalizadorSolicitudAltaNoDisponible
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	if valorLecturaJSON(d, 0) != nil || exigirFinJSONAlta(d) != nil {
		return s, ErrLocalizadorSolicitudAltaNoDisponible
	}
	m, err := objetoLecturaJSON(raw, "esquema", "contrato_version", "solicitud_ref", "expediente_ref", "version_expediente", "capacidad_ref", "correlacion_ref", "idempotencia_ref", "fuente_rpt", "puesto_ref", "plaza_ref")
	if err != nil {
		return s, ErrLocalizadorSolicitudAltaNoDisponible
	}
	if _, err = objetoLecturaJSON(m["fuente_rpt"], "referencia", "version", "huella_sha256"); err != nil {
		return s, ErrLocalizadorSolicitudAltaNoDisponible
	}
	if json.Unmarshal(raw, &s) != nil || s.Validar() != nil {
		return ct.SolicitudAltaPersonalRPT{}, ErrLocalizadorSolicitudAltaNoDisponible
	}
	return s, nil
}
