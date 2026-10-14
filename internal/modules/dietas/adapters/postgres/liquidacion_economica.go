package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const registrarLiquidacionEconomicaSQL = `SELECT vec_dietas.registrar_liquidacion_economica_v1($1::text,$2::bytea,$3::bytea,$4::bytea,$5::bytea,$6::numeric,$7::numeric,$8::bytea,$9::bytea,$10::bytea,$11::bytea)`
const consultarLiquidacionEconomicaSQL = `SELECT vec_dietas.consultar_liquidacion_economica_v1($1::text,$2::bigint,$3::bytea,$4::bytea,$5::jsonb)`

var _ dietasports.RepositorioLiquidacionEconomica = (*RepositorioBorradorComisionPostgreSQL)(nil)
var _ dietasports.LectorLiquidacionEconomica = (*RepositorioBorradorComisionPostgreSQL)(nil)
var referenciaAuditoriaLiquidacion = regexp.MustCompile(`^aud_v3_ld_[0-9a-f]{32}$`)
var huellaAuditoriaLiquidacion = regexp.MustCompile(`^[0-9a-f]{64}$`)
var intentoLecturaLiquidacion = regexp.MustCompile(`^lectura_[0-9a-f]{32}$`)

// Registrar confirma un solo acto nominal. Un resultado no verificable causa
// ROLLBACK antes de COMMIT; ante un COMMIT incierto el cliente repite la misma
// clave y SQL devuelve la instantánea y el recibo originales.
func (r *RepositorioBorradorComisionPostgreSQL) Registrar(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudLiquidacionEconomica) (domain.InstantaneaLiquidacionEconomica, error) {
	var cero domain.InstantaneaLiquidacionEconomica
	if err := r.valido(ctx); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if solicitud.Preparacion == nil || identidad.ContextoRegistrado.Validar() != nil ||
		identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil ||
		identidad.Autorizacion.Accion != application.AccionRegistrarLiquidacionEconomica ||
		identidad.Autorizacion.Finalidad != application.FinalidadRegistrarLiquidacionEconomica ||
		identidad.Autorizacion.RecursoRef != solicitud.Decision.Referencia ||
		identidad.Autorizacion.Material.ValidarEstructura() != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	efecto, err := application.ConstruirEfectoAutorizacionLiquidacionEconomica(
		identidad.ContextoRegistrado, identidad.UnidadCompetenciaRef, solicitud)
	if err != nil || len(efecto.Material) == 0 || efecto.Recurso.Referencia != solicitud.Decision.Referencia {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	for intento := 0; intento < 3; intento++ {
		resultado, repetir, err := r.registrarLiquidacionUnaVez(ctx, identidad, solicitud, string(efecto.Material))
		if !repetir {
			return resultado, err
		}
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
	}
	return cero, dietasports.ErrResultadoCircuitoIncierto
}

func (r *RepositorioBorradorComisionPostgreSQL) registrarLiquidacionUnaVez(ctx context.Context, identidad dietasports.IdentidadEfectivaCircuito, solicitud dietasports.SolicitudLiquidacionEconomica, material string) (domain.InstantaneaLiquidacionEconomica, bool, error) {
	var cero domain.InstantaneaLiquidacionEconomica
	argumentos, err := argumentosAD3(identidad.Autorizacion.Material)
	if err != nil {
		return cero, false, dietasports.ErrAccesoCircuitoDenegado
	}
	defer argumentos.limpiar()
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	if _, err = tx.Exec(ctx, `SET LOCAL TIME ZONE 'UTC'`); err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	if _, err = tx.Exec(ctx, `SET LOCAL statement_timeout='15s'`); err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	if _, err = tx.Exec(ctx, `SET LOCAL idle_in_transaction_session_timeout='20s'`); err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	var bruto []byte
	err = tx.QueryRow(ctx, registrarLiquidacionEconomicaSQL, material,
		argumentos.capacidad, argumentos.decision, argumentos.motivo, argumentos.contexto,
		argumentos.personaVersion, argumentos.perfilVersion, argumentos.payload,
		argumentos.sobre, argumentos.evidencia, argumentos.raiz).Scan(&bruto)
	if err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	var instantanea domain.InstantaneaLiquidacionEconomica
	if len(bruto) == 0 || len(bruto) > 1<<20 || json.Unmarshal(bruto, &instantanea) != nil {
		return cero, false, dietasports.ErrResultadoCircuitoIncierto
	}
	liquidacion, err := domain.RecuperarLiquidacionEconomica(instantanea)
	if err != nil || instantanea.Constancia.DecisorPersonaRef != identidad.ContextoRegistrado.Contexto.PersonaRef ||
		instantanea.Constancia.UnidadRef != solicitud.Decision.UnidadRef ||
		instantanea.Constancia.VersionEntrada != solicitud.Decision.VersionEsperada ||
		instantanea.Constancia.VersionResultante != solicitud.Decision.VersionEsperada+1 ||
		!reflect.DeepEqual(instantanea.Economia, solicitud.Preparacion.Instantanea()) {
		return cero, false, dietasports.ErrResultadoCircuitoIncierto
	}
	if err = tx.Commit(ctx); err != nil {
		repetir, causa := errorReintentoLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	return liquidacion.Instantanea(), false, nil
}

func errorReintentoLiquidacion(ctx context.Context, err error) (bool, error) {
	if ctx != nil && ctx.Err() != nil {
		return false, ctx.Err()
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "40001", "40P01":
			return true, nil
		case "PD002":
			return false, dietasports.ErrConflictoIdempotencia
		case "PD003", "42501":
			return false, dietasports.ErrAccesoCircuitoDenegado
		case "PD004":
			return false, dietasports.ErrComisionNoEncontrada
		case "PD005":
			return false, dietasports.ErrEstadoCircuitoConflicto
		case "P7201":
			return false, dietasports.ErrRelacionNoDisponible
		}
	}
	return false, dietasports.ErrResultadoCircuitoIncierto
}

// Consultar usa la evaluación de permisos actual capturada por el montaje.
// PostgreSQL añade unidad y titular de su fila y asienta la lectura antes de
// revelar la instantánea. La validación del resultado precede al COMMIT.
func (r *RepositorioBorradorComisionPostgreSQL) Consultar(ctx context.Context, captura dietasports.CapturaLecturaLiquidacionEconomica, consulta dietasports.ConsultaLiquidacionEconomica) (dietasports.ResultadoLecturaLiquidacionEconomica, error) {
	var cero dietasports.ResultadoLecturaLiquidacionEconomica
	if err := r.valido(ctx); err != nil {
		if ctx != nil && ctx.Err() != nil {
			return cero, ctx.Err()
		}
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if !cursorBandeja.MatchString(consulta.Referencia) || (consulta.Version != 0 && consulta.Version < 2) ||
		consulta.Version > math.MaxInt64 ||
		len(captura.ContextoCanonico) == 0 || len(captura.ContextoCanonico) > 65536 ||
		len(captura.VinculoCanonico) == 0 || len(captura.VinculoCanonico) > 16384 ||
		len(captura.Orden) == 0 || len(captura.Orden) > 65536 ||
		captura.ValidaHasta.IsZero() || !time.Now().Before(captura.ValidaHasta) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	var orden map[string]json.RawMessage
	if json.Unmarshal(captura.Orden, &orden) != nil || len(orden) != 33 ||
		orden["unidad_ref"] != nil || orden["titular_persona_ref"] != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	var recurso, accion, correlacion, intentoRef string
	if json.Unmarshal(orden["recurso_ref"], &recurso) != nil || recurso != consulta.Referencia ||
		json.Unmarshal(orden["accion"], &accion) != nil || accion != "dietas.liquidacion_economica.consultar" ||
		json.Unmarshal(orden["correlacion_ref"], &correlacion) != nil || correlacion == "" ||
		json.Unmarshal(orden["intento_ref"], &intentoRef) != nil || !intentoLecturaLiquidacion.MatchString(intentoRef) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	for intento := 0; intento < 3; intento++ {
		resultado, repetir, err := r.consultarLiquidacionUnaVez(ctx, captura, consulta, correlacion, intentoRef)
		if !repetir {
			return resultado, err
		}
		if ctx.Err() != nil {
			return cero, ctx.Err()
		}
	}
	return cero, dietasports.ErrCircuitoNoDisponible
}

func (r *RepositorioBorradorComisionPostgreSQL) consultarLiquidacionUnaVez(ctx context.Context, captura dietasports.CapturaLecturaLiquidacionEconomica, consulta dietasports.ConsultaLiquidacionEconomica, correlacion, intentoRef string) (dietasports.ResultadoLecturaLiquidacionEconomica, bool, error) {
	var cero dietasports.ResultadoLecturaLiquidacionEconomica
	version := consulta.Version
	if version > math.MaxInt64 {
		return cero, false, dietasports.ErrAccesoCircuitoDenegado
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		repetir, causa := errorReintentoLecturaLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	defer func() {
		rollbackCtx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}()
	for _, sentencia := range []string{`SET LOCAL TIME ZONE 'UTC'`, `SET LOCAL statement_timeout='10s'`, `SET LOCAL idle_in_transaction_session_timeout='15s'`} {
		if _, err = tx.Exec(ctx, sentencia); err != nil {
			repetir, causa := errorReintentoLecturaLiquidacion(ctx, err)
			return cero, repetir, causa
		}
	}
	var bruto []byte
	err = tx.QueryRow(ctx, consultarLiquidacionEconomicaSQL, consulta.Referencia,
		int64(version), captura.ContextoCanonico, captura.VinculoCanonico,
		[]byte(captura.Orden)).Scan(&bruto)
	if err != nil {
		repetir, causa := errorReintentoLecturaLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	var respuesta struct {
		Esquema     string                                           `json:"esquema"`
		Liquidacion domain.InstantaneaLiquidacionEconomica           `json:"liquidacion"`
		Auditoria   dietasports.AuditoriaLecturaLiquidacionEconomica `json:"auditoria"`
	}
	if len(bruto) == 0 || len(bruto) > 1<<20 ||
		json.Unmarshal(bruto, &respuesta) != nil || respuesta.Esquema != "vec.dietas.consulta-liquidacion-economica.v1" ||
		respuesta.Liquidacion.Economia.ComisionRef != consulta.Referencia ||
		respuesta.Liquidacion.Constancia.VersionResultante < 2 ||
		(consulta.Version != 0 && respuesta.Liquidacion.Constancia.VersionResultante != consulta.Version) ||
		respuesta.Auditoria.CorrelacionRef != correlacion ||
		!referenciaAuditoriaLiquidacion.MatchString(respuesta.Auditoria.Referencia) ||
		respuesta.Auditoria.Referencia != "aud_v3_ld_"+intentoRef[len("lectura_"):] ||
		!huellaAuditoriaLiquidacion.MatchString(respuesta.Auditoria.HuellaSHA256) ||
		respuesta.Auditoria.Secuencia == 0 {
		return cero, false, dietasports.ErrCircuitoNoDisponible
	}
	fecha, err := instanteReciboCircuito(respuesta.Auditoria.RegistradaEn)
	if err != nil || fecha.IsZero() {
		return cero, false, dietasports.ErrCircuitoNoDisponible
	}
	liquidacion, err := domain.RecuperarLiquidacionEconomica(respuesta.Liquidacion)
	if err != nil {
		return cero, false, dietasports.ErrCircuitoNoDisponible
	}
	if err = tx.Commit(ctx); err != nil {
		repetir, causa := errorReintentoLecturaLiquidacion(ctx, err)
		return cero, repetir, causa
	}
	return dietasports.ResultadoLecturaLiquidacionEconomica{Liquidacion: liquidacion.Instantanea(), Auditoria: respuesta.Auditoria}, false, nil
}

func errorReintentoLecturaLiquidacion(ctx context.Context, err error) (bool, error) {
	repetir, causa := errorReintentoLiquidacion(ctx, err)
	if repetir {
		return true, nil
	}
	if errors.Is(causa, dietasports.ErrResultadoCircuitoIncierto) {
		return false, dietasports.ErrCircuitoNoDisponible
	}
	if errors.Is(causa, dietasports.ErrComisionNoEncontrada) {
		return false, dietasports.ErrAccesoCircuitoDenegado
	}
	return false, causa
}
