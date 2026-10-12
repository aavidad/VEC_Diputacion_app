package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const funcionConsultarPreferenciasBolsas = "vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1"
const funcionIntentoLecturaPreferenciasBolsas = "vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1"

var auditoriaLecturaPreferenciasRef = regexp.MustCompile(`^aud_v3_lp_[0-9a-f]{32}$`)
var auditoriaLecturaPreferenciasHuella = regexp.MustCompile(`^[0-9a-f]{64}$`)
var auditoriaIntentoPreferenciasRef = regexp.MustCompile(`^aud_v3_i_[0-9a-f]{32}$`)

type LecturaPreferenciasBolsasPostgreSQL struct{ pool iniciadorTransacciones }
type AuditoriaIntentosPreferenciasBolsasPostgreSQL struct{ pool iniciadorTransacciones }

var _ bolsa.LecturaPreferenciasBolsas = (*LecturaPreferenciasBolsasPostgreSQL)(nil)
var _ bolsa.AuditoriaIntentosPreferenciasBolsas = (*AuditoriaIntentosPreferenciasBolsasPostgreSQL)(nil)

func NuevaAuditoriaIntentosPreferenciasBolsasPostgreSQL(pool *pgxpool.Pool) (*AuditoriaIntentosPreferenciasBolsasPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &AuditoriaIntentosPreferenciasBolsasPostgreSQL{pool: pool}, nil
}

func (a *AuditoriaIntentosPreferenciasBolsasPostgreSQL) RegistrarIntentoLecturaPreferenciasBolsas(ctx context.Context, c bolsa.CapturaLecturaPreferencias) error {
	if ctx == nil || a == nil || valorNulo(a.pool) || c.CandidatoRef() == "" || c.CorrelacionRef() == "" ||
		len(c.ContextoCanonico()) == 0 || len(c.VinculoCanonico()) == 0 || len(c.OrdenJSON()) == 0 {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error { return a.registrarIntento(ctx, c) })
	if err != nil {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	return nil
}

func (a *AuditoriaIntentosPreferenciasBolsasPostgreSQL) registrarIntento(ctx context.Context, c bolsa.CapturaLecturaPreferencias) error {
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return err
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
 set_config('row_security','on',true),set_config('timezone','UTC',true),
 set_config('lock_timeout','2s',true),set_config('statement_timeout','10s',true),
 set_config('idle_in_transaction_session_timeout','15s',true)`); err != nil {
		return err
	}
	var ref, huella, correlacion string
	var secuencia string
	var fecha time.Time
	err = tx.QueryRow(ctx, `SELECT auditoria_ref,secuencia::text,huella_sha256,correlacion_ref,registrada_en FROM `+
		funcionIntentoLecturaPreferenciasBolsas+`($1::text,$2::bytea,$3::bytea,$4::jsonb)`,
		c.CandidatoRef(), c.ContextoCanonico(), c.VinculoCanonico(), []byte(c.OrdenJSON())).Scan(&ref, &secuencia, &huella, &correlacion, &fecha)
	if err != nil {
		return err
	}
	if !auditoriaIntentoPreferenciasRef.MatchString(ref) || secuencia == "" || !auditoriaLecturaPreferenciasHuella.MatchString(huella) ||
		correlacion != c.CorrelacionRef() || fecha.IsZero() || fecha.After(time.Now().UTC().Add(time.Second)) {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func NuevaLecturaPreferenciasBolsasPostgreSQL(pool *pgxpool.Pool) (*LecturaPreferenciasBolsasPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, bolsa.ErrPreferenciasNoDisponibles
	}
	return &LecturaPreferenciasBolsasPostgreSQL{pool: pool}, nil
}

func (a *LecturaPreferenciasBolsasPostgreSQL) LeerPreferenciasBolsas(ctx context.Context, c bolsa.CapturaLecturaPreferencias) (bolsa.EstadoPreferenciasBolsas, error) {
	vacia := bolsa.EstadoPreferenciasBolsas{}
	if ctx == nil || a == nil || valorNulo(a.pool) || c.CandidatoRef() == "" || c.CorrelacionRef() == "" ||
		len(c.ContextoCanonico()) == 0 || len(c.VinculoCanonico()) == 0 || len(c.OrdenJSON()) == 0 || !c.ValidaHasta().After(time.Now().UTC()) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	var estado bolsa.EstadoPreferenciasBolsas
	err := postgresql.RepetirTrasCarreraSerializable(ctx, func() error {
		var err error
		estado, err = a.leerIntento(ctx, c)
		return err
	})
	if err != nil {
		return vacia, errorLecturaPreferencias(ctx, err)
	}
	return estado, nil
}

func (a *LecturaPreferenciasBolsasPostgreSQL) leerIntento(ctx context.Context, c bolsa.CapturaLecturaPreferencias) (bolsa.EstadoPreferenciasBolsas, error) {
	vacia := bolsa.EstadoPreferenciasBolsas{}
	tx, err := a.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return vacia, err
	}
	defer revertir(tx)
	if _, err = tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
 set_config('row_security','on',true),set_config('timezone','UTC',true),
 set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true),
 set_config('idle_in_transaction_session_timeout','20s',true)`); err != nil {
		return vacia, err
	}
	var contenido []byte
	err = tx.QueryRow(ctx, `SELECT `+funcionConsultarPreferenciasBolsas+`($1::text,$2::bytea,$3::bytea,$4::jsonb)`,
		c.CandidatoRef(), c.ContextoCanonico(), c.VinculoCanonico(), []byte(c.OrdenJSON())).Scan(&contenido)
	if err != nil {
		return vacia, err
	}
	defer borrarBytesPostgreSQL(contenido)
	estado, err := decodificarLecturaPreferencias(contenido, c.CorrelacionRef())
	if err != nil {
		return vacia, err
	}
	if err := ctx.Err(); err != nil {
		return vacia, err
	}
	if err := tx.Commit(ctx); err != nil {
		return vacia, err
	}
	return estado, nil
}

type estadoLecturaPreferenciasJSON struct {
	Version           uint64     `json:"version"`
	OrdenGuardado     []string   `json:"orden_guardado"`
	BolsasActuales    []string   `json:"bolsas_actuales"`
	BolsasNuevas      []string   `json:"bolsas_nuevas"`
	BolsasRetiradas   []string   `json:"bolsas_retiradas"`
	RevisionRequerida bool       `json:"revision_requerida"`
	ReciboRef         *string    `json:"recibo_ref"`
	RegistradoEn      *time.Time `json:"registrado_en"`
}
type lecturaPreferenciasJSON struct {
	Estado                estadoLecturaPreferenciasJSON `json:"estado"`
	AuditoriaRef          string                        `json:"auditoria_ref"`
	AuditoriaHuellaSHA256 string                        `json:"auditoria_huella_sha256"`
	CorrelacionRef        string                        `json:"correlacion_ref"`
	AuditadaEn            time.Time                     `json:"auditada_en"`
}

func decodificarLecturaPreferencias(contenido []byte, correlacion string) (bolsa.EstadoPreferenciasBolsas, error) {
	vacia := bolsa.EstadoPreferenciasBolsas{}
	if len(contenido) == 0 || len(contenido) > 131072 {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	if err := validarClavesJSONPreferencias(contenido, []string{"estado", "auditoria_ref", "auditoria_huella_sha256", "correlacion_ref", "auditada_en"}); err != nil {
		return vacia, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	var envoltura struct {
		Estado json.RawMessage `json:"estado"`
	}
	if err := json.Unmarshal(contenido, &envoltura); err != nil {
		return vacia, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	if err := validarClavesJSONPreferencias(envoltura.Estado, bolsa.CamposLecturaPreferenciasBolsas()); err != nil {
		return vacia, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var p lecturaPreferenciasJSON
	if err := dec.Decode(&p); err != nil {
		return vacia, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		if err != nil {
			return vacia, errors.Join(bolsa.ErrPreferenciasNoDisponibles, err)
		}
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	if !auditoriaLecturaPreferenciasRef.MatchString(p.AuditoriaRef) || !auditoriaLecturaPreferenciasHuella.MatchString(p.AuditoriaHuellaSHA256) ||
		p.CorrelacionRef != correlacion || p.AuditadaEn.IsZero() || p.AuditadaEn.After(time.Now().UTC().Add(time.Second)) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	e := p.Estado
	if e.OrdenGuardado == nil || e.BolsasActuales == nil || e.BolsasNuevas == nil || e.BolsasRetiradas == nil ||
		e.Version == 0 && (e.ReciboRef != nil || e.RegistradoEn != nil) ||
		e.Version > 0 && (e.ReciboRef == nil || *e.ReciboRef == "" || e.RegistradoEn == nil || e.RegistradoEn.IsZero() || e.RegistradoEn.After(p.AuditadaEn)) {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	proyectado, err := aplicacionbolsa.ProyectarEstadoPreferencias(e.Version, e.OrdenGuardado, e.BolsasActuales)
	if err != nil {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	nuevas := slices.Clone(proyectado.BolsasNuevas)
	retiradas := slices.Clone(proyectado.BolsasRetiradas)
	sort.Strings(nuevas)
	sort.Strings(retiradas)
	if !slices.Equal(nuevas, e.BolsasNuevas) || !slices.Equal(retiradas, e.BolsasRetiradas) || e.RevisionRequerida != proyectado.RevisionRequerida {
		return vacia, bolsa.ErrPreferenciasNoDisponibles
	}
	proyectado.BolsasNuevas = e.BolsasNuevas
	proyectado.BolsasRetiradas = e.BolsasRetiradas
	if e.ReciboRef != nil {
		proyectado.ReciboRef = *e.ReciboRef
	}
	proyectado.RegistradoEn = e.RegistradoEn
	return proyectado, nil
}

// Comprueba las claves y los duplicados antes del DTO. json.Unmarshal por sí
// solo aceptaría la última aparición de una clave repetida.
func validarClavesJSONPreferencias(contenido []byte, esperadas []string) error {
	d := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := d.Token()
	if err != nil {
		return err
	}
	if inicio != json.Delim('{') {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	permitidas := make(map[string]bool, len(esperadas))
	for _, k := range esperadas {
		permitidas[k] = true
	}
	vistas := make(map[string]bool, len(esperadas))
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return err
		}
		clave, ok := token.(string)
		if !ok || !permitidas[clave] || vistas[clave] {
			return bolsa.ErrPreferenciasNoDisponibles
		}
		var valor json.RawMessage
		if err := d.Decode(&valor); err != nil {
			return err
		}
		vistas[clave] = true
	}
	cierre, err := d.Token()
	if err != nil {
		return err
	}
	if cierre != json.Delim('}') || len(vistas) != len(esperadas) {
		return bolsa.ErrPreferenciasNoDisponibles
	}
	_, err = d.Token()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return bolsa.ErrPreferenciasNoDisponibles
}

func errorLecturaPreferencias(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42501" &&
		(strings.HasPrefix(pgErr.Message, "B104: lectura nominal inválida") ||
			strings.HasPrefix(pgErr.Message, "AD246: lectura sin emisor acreditado") ||
			strings.HasPrefix(pgErr.Message, "AD246: vínculo histórico de lectura inválido") ||
			strings.HasPrefix(pgErr.Message, "AD246: sesión o candidato externo denegado") ||
			strings.HasPrefix(pgErr.Message, "AD246: autoridad actual de lectura denegada")) {
		return dominiovec.ErrAutorizacionDenegada
	}
	return bolsa.ErrPreferenciasNoDisponibles
}
