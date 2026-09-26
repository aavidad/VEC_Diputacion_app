package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	postgresbolsa "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	aplicacionbolsa "vec-diputacion-granada/internal/modules/bolsa/application"
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	puertosct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// Relevo de no incorporaciones (duda 12 de RRHH): lee la publicación de CT
// 000124 y la entrega a la bandeja de Bolsa 000042 con su propio LOGIN (grupo
// vec_bolsa_llamamientos_relevo_no_incorporacion, que es el único que puede
// entregar). La bandeja comprueba con CT el origen de cada evento, resuelve la
// consecuencia con la política que se publica aquí al arrancar desde el
// catálogo de Bolsa y, solo si la baja queda aplicada, habilita el siguiente
// llamamiento. En cada pasada reevalúa lo que aún no se pudo aplicar. Como el
// de contratos, solo compone puertos: ningún módulo lee tablas del otro.

const (
	rolRelevoNoIncorporacionBolsaDesarrollo = "vec_bolsa_llamamientos_relevo_no_incorporacion"
	// lotePendientesNoIncorporacion acota la reevaluación de cada pasada.
	lotePendientesNoIncorporacion = 50
)

var (
	ErrIncorporacionAcreditadaFaltaBolsa42 = fmt.Errorf("%w: falta Bolsa 000042 (bandeja de no incorporaciones)", ErrIncorporacionAcreditadaMigracionesNoDisponibles)
	errNoIncorporacionesSinCatalogoBolsa   = errors.New("bootstrap: la no incorporación exige el paquete de reglas de Bolsa (consecuencias b24)")
	errRelevoNoIncorporacionesSinConexion  = errors.New("bootstrap: el relevo de no incorporaciones exige su conexión propia (" +
		config.EnvBolsaRelevoNoIncorporacionDatabaseURL + ") con un LOGIN solo del grupo " + rolRelevoNoIncorporacionBolsaDesarrollo)
)

// consultaMigracionBolsa42 se ejecuta con el LOGIN del relevo.
const consultaMigracionBolsa42 = `SELECT bool_and(coalesce(pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE'),false))
 FROM unnest(ARRAY['vec_bolsa_llamamientos.registrar_no_incorporacion_bolsa_v1(jsonb,text,timestamptz,bigint,jsonb)',
  'vec_bolsa_llamamientos.cursor_no_incorporaciones_bolsa_v1()',
  'vec_bolsa_llamamientos.pendientes_no_incorporacion_bolsa_v1(integer)',
  'vec_bolsa_llamamientos.reevaluar_no_incorporacion_bolsa_v1(text,jsonb)']) f`

// catalogoNoIncorporacionBolsa es el catálogo de sanciones de Bolsa: calcula
// las fechas de cada evento y compone la política que se publica al arrancar.
type catalogoNoIncorporacionBolsa interface {
	puertosbolsa.ResolvedorSancionNoIncorporacion
	PoliticaNoIncorporacion(context.Context) (puertosbolsa.PoliticaNoIncorporacion, error)
}

type entregaNoIncorporacionesCTBolsa struct {
	lector   puertosct.LectorPublicacionNoIncorporacionesBolsa
	receptor *aplicacionbolsa.ServicioRecepcionNoIncorporaciones
	lote     int
}

// entregar hace una pasada completa desde el cursor de la bandeja y después
// reevalúa lo pendiente. Un evento inválido, que CT no publicó o repetido se
// registra y no bloquea a los demás; una indisponibilidad (también del
// catálogo) detiene la pasada.
func (e *entregaNoIncorporacionesCTBolsa) entregar(ctx context.Context) (resultadoEntregaContratosCT, error) {
	var r resultadoEntregaContratosCT
	if e == nil || e.lector == nil || e.receptor == nil || e.lote < 1 || e.lote > puertosct.LimiteLecturaContratosBolsa || ctx == nil {
		return r, puertosbolsa.ErrContratosParticipacionNoDisponible
	}
	cursor, hay, err := e.receptor.Cursor(ctx)
	if err != nil {
		return r, err
	}
	var desde puertosct.CursorPublicacionContratosBolsa
	if hay {
		desde = puertosct.CursorPublicacionContratosBolsa{Posicion: cursor.Posicion, OrigenRef: cursor.OrigenRef}
	}
	for pagina := 0; pagina < maximoPaginasEntregaContratosCT; pagina++ {
		eventos, err := e.lector.LeerNoIncorporacionesBolsa(ctx, desde, e.lote)
		if err != nil {
			return r, err
		}
		for _, evento := range eventos {
			res, err := e.receptor.Recibir(ctx, evento.Contenido, evento.HuellaSHA256, evento.OrigenCreadaEn, evento.OrigenPosicion)
			switch {
			case errors.Is(err, dominiobolsa.ErrEventoContratoParticipacionInvalido), errors.Is(err, puertosbolsa.ErrEventoContratoDivergente):
				r.rechazados++
				slog.Warn("no incorporación de CT en cuarentena o rechazada por la bandeja de Bolsa", "evento_ref", evento.EventoRef, "causa", err)
			case err != nil:
				return r, err
			case res.Reutilizado:
				r.reentregas++
			default:
				r.nuevos++
				if res.Estado != puertosbolsa.EstadoNoIncorporacionAplicada {
					slog.Warn("no incorporación de CT registrada en Bolsa sin baja; queda para revisión de RRHH", "evento_ref", evento.EventoRef, "estado", res.Estado)
				}
			}
		}
		if len(eventos) < e.lote {
			break
		}
		ultimo := eventos[len(eventos)-1]
		desde = puertosct.CursorPublicacionContratosBolsa{Posicion: ultimo.OrigenPosicion, OrigenRef: ultimo.OrigenRef}
	}
	aplicadas, err := e.receptor.ReevaluarPendientes(ctx, lotePendientesNoIncorporacion)
	if err != nil {
		return r, err
	}
	if aplicadas > 0 {
		slog.Info("no incorporaciones pendientes aplicadas en Bolsa tras reevaluarlas", "aplicadas", aplicadas)
	}
	return r, nil
}

// iniciarEntregaNoIncorporacionesCTBolsaDesarrollo publica la política de no
// incorporación con la cuenta de ejecución de Bolsa y arranca el relevo con
// su conexión propia y la misma cadencia que el de contratos. Con la
// incorporación acreditada encendida, la falta de Bolsa 000042, del catálogo
// de Bolsa o de la conexión del relevo impide arrancar.
func iniciarEntregaNoIncorporacionesCTBolsaDesarrollo(ctx context.Context, cfg config.Config, ejecucionCT, bolsa *pgxpool.Pool,
	catalogo catalogoNoIncorporacionBolsa) (func(), error) {
	nada := func() {}
	if dependenciaEsNulaContratacionTemporalDesarrollo(catalogo) {
		return nada, errNoIncorporacionesSinCatalogoBolsa
	}
	if bolsa == nil || ejecucionCT == nil || ctx == nil {
		return nada, ErrIncorporacionAcreditadaFaltaBolsa42
	}
	politica, err := catalogo.PoliticaNoIncorporacion(ctx)
	if err != nil {
		return nada, errors.Join(errNoIncorporacionesSinCatalogoBolsa, err)
	}
	publicador, err := postgresbolsa.NuevaPoliticaNoIncorporacionPostgreSQL(bolsa)
	if err != nil {
		return nada, err
	}
	version, err := publicador.PublicarPoliticaNoIncorporacion(ctx, politica)
	if err != nil {
		return nada, errors.Join(ErrIncorporacionAcreditadaFaltaBolsa42, err)
	}
	slog.Info("bolsa: política de no incorporación del catálogo publicada", "version", version)
	opciones, err := cfg.BolsaContratosCT.Resolver()
	if err != nil {
		return nada, err
	}
	if !opciones.Activa {
		slog.Warn("entrega de no incorporaciones CT a Bolsa desactivada por configuración: la baja y el siguiente esperan al relevo")
		return nada, nil
	}
	dsn, err := cfg.DSNBolsaRelevoNoIncorporacionSeparado()
	if err != nil {
		return nada, errors.Join(errRelevoNoIncorporacionesSinConexion, err)
	}
	pool, err := abrirPoolRelevoNoIncorporacionesDesarrollo(ctx, dsn)
	if err != nil {
		return nada, errors.Join(errRelevoNoIncorporacionesSinConexion, err)
	}
	var instalada bool
	if err := pool.QueryRow(ctx, consultaMigracionBolsa42).Scan(&instalada); err != nil || !instalada {
		pool.Close()
		return nada, ErrIncorporacionAcreditadaFaltaBolsa42
	}
	lector, err := postgresct.NuevoLectorPublicacionContratosBolsaPostgreSQL(ejecucionCT)
	if err != nil {
		pool.Close()
		return nada, err
	}
	buzon, err := postgresbolsa.NuevoBuzonNoIncorporacionesPostgreSQL(pool)
	if err != nil {
		pool.Close()
		return nada, err
	}
	receptor, err := aplicacionbolsa.NuevoServicioRecepcionNoIncorporaciones(buzon, catalogo)
	if err != nil {
		pool.Close()
		return nada, err
	}
	relevo := &entregaNoIncorporacionesCTBolsa{lector: lector, receptor: receptor, lote: opciones.Lote}
	detener := mantenerEntregaCTBolsa("no incorporaciones", relevo.entregar, opciones.Intervalo, esperarTemporizadorCTDesarrollo)
	return func() {
		detener()
		pool.Close()
	}, nil
}

// abrirPoolRelevoNoIncorporacionesDesarrollo abre la conexión del relevo y
// exige, al nacer cada conexión física, que su LOGIN solo pertenezca al grupo
// del relevo: ni el ejecutor general de Bolsa ni ningún rol de CT.
func abrirPoolRelevoNoIncorporacionesDesarrollo(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	configuracion, err := pgxpool.ParseConfig(dsn)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		validarTLSPostgreSQLBorradores(&configuracion.ConnConfig.Config, true) != nil {
		return nil, falloPostgreSQLCTDesarrollo(err)
	}
	configuracion.MaxConns, configuracion.MinConns = 2, 0
	configuracion.ConnConfig.ConnectTimeout = 5 * time.Second
	if configuracion.ConnConfig.RuntimeParams == nil {
		configuracion.ConnConfig.RuntimeParams = make(map[string]string)
	}
	parametros := configuracion.ConnConfig.RuntimeParams
	parametros["application_name"] = "vec-bolsa-relevo-no-incorporaciones"
	parametros["timezone"] = "UTC"
	parametros["search_path"] = "pg_catalog,pg_temp"
	parametros["statement_timeout"] = "15s"
	parametros["lock_timeout"] = "3s"
	parametros["idle_in_transaction_session_timeout"] = "20s"
	configuracion.AfterConnect = func(ctx context.Context, conexion *pgx.Conn) error {
		if conexion == nil {
			return falloPostgreSQLCTDesarrollo(nil)
		}
		return comprobarIdentidadRelevoNoIncorporacionesDesarrollo(ctx, conexion)
	}
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil {
		return nil, falloPostgreSQLCTDesarrollo(err)
	}
	if err := comprobarIdentidadRelevoNoIncorporacionesDesarrollo(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func comprobarIdentidadRelevoNoIncorporacionesDesarrollo(ctx context.Context, consultador interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || consultador == nil {
		return falloPostgreSQLCTDesarrollo(nil)
	}
	var valido bool
	err := consultador.QueryRow(ctx, `
		WITH RECURSIVE membresias_efectivas(rol_id, admin_option) AS (
			SELECT directa.roleid, directa.admin_option FROM pg_catalog.pg_auth_members AS directa WHERE directa.member = session_user::regrole
			UNION
			SELECT siguiente.roleid, previa.admin_option OR siguiente.admin_option FROM pg_catalog.pg_auth_members AS siguiente JOIN membresias_efectivas AS previa ON previa.rol_id = siguiente.member
		)
		SELECT session_user = current_user AND identidad.rolcanlogin AND identidad.rolinherit
		       AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb AND NOT identidad.rolcreaterole
		       AND NOT identidad.rolreplication AND NOT identidad.rolbypassrls
		       AND pg_catalog.pg_has_role(session_user, $1::regrole, 'USAGE')
		       AND NOT grupo.rolcanlogin AND NOT grupo.rolsuper AND NOT grupo.rolbypassrls
		       AND NOT EXISTS (SELECT 1 FROM membresias_efectivas WHERE rol_id <> $1::regrole OR admin_option)
		  FROM pg_catalog.pg_roles AS identidad
		 CROSS JOIN pg_catalog.pg_roles AS grupo
		 WHERE identidad.rolname = session_user AND grupo.oid = $1::regrole`, rolRelevoNoIncorporacionBolsaDesarrollo).Scan(&valido)
	if err != nil || !valido {
		return falloPostgreSQLCTDesarrollo(err)
	}
	return nil
}

// componerAvisosNoIncorporacionBolsaDesarrollo lleva a los avisos de Bolsa
// las no incorporaciones que esperan revisión de RRHH (bandeja v3, Bolsa
// 000042). Solo con la incorporación acreditada encendida; entonces la falta
// de la migración impide arrancar, como el relevo.
func componerAvisosNoIncorporacionBolsaDesarrollo(ctx context.Context, cfg config.Config, fuente *fuenteConstituidaRRHHDesarrollo) error {
	if fuente == nil || fuente.consultaAvisos == nil || !incorporacionAcreditadaSolicitada(cfg) {
		return nil
	}
	if err := fuente.consultaAvisos.ActivarNoIncorporaciones(ctx); err != nil {
		return errors.Join(ErrIncorporacionAcreditadaFaltaBolsa42, err)
	}
	return nil
}
