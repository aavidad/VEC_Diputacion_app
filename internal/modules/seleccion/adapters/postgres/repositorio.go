// Package postgres traduce los puertos de Selección a las funciones de
// vec_seleccion (Selección 000001). Cada operación abre su propia
// transacción serializable y el consumo de la decisión V3 ocurre dentro de la
// misma función que escribe o lee. No lee tablas de otros módulos.
package postgres

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const configuracionTransaccion = `SELECT set_config('search_path','pg_catalog',true), set_config('row_security','on',true), set_config('timezone','UTC',true), set_config('lock_timeout','2s',true), set_config('statement_timeout','15s',true), set_config('idle_in_transaction_session_timeout','20s',true)`

// ConsultaMigraciones comprueba que el LOGIN puede ejecutar las funciones
// públicas de Selección 000001 y que AD3-89 y AD3-90 existen (catálogo).
const ConsultaMigraciones = `SELECT
 (SELECT count(*)=1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_solicitud_propia_seleccion_v3_atestada'),
 (SELECT count(*)=1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3' AND p.proname='registrar_y_consumir_consulta_solicitudes_seleccion_v3_atestada'),
 (SELECT coalesce(bool_and(pg_catalog.to_regprocedure(f) IS NOT NULL AND pg_catalog.has_function_privilege(pg_catalog.to_regprocedure(f),'EXECUTE')),false) FROM pg_catalog.unnest(ARRAY[
  'vec_seleccion.publicar_convocatoria_v1(text,text,timestamptz,timestamptz,jsonb,text,timestamptz)',
  'vec_seleccion.convocatorias_publicadas_v1()',
  'vec_seleccion.guardar_borrador_propio_v1(text,text,integer,integer,text,text,text,text,text,bytea,bytea,text,text,boolean,jsonb,jsonb,bigint,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_seleccion.presentar_solicitud_propia_v1(text,text,integer,text,text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_seleccion.listar_solicitudes_propias_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_seleccion.leer_borrador_propio_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_seleccion.listar_solicitudes_convocatoria_v1(text,bigint,integer,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_seleccion.leer_detalle_solicitud_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f)`

// EstadoMigraciones indica qué migración falta, en orden de instalación.
type EstadoMigraciones struct {
	AD389, AD390, Seleccion1 bool
}

// ComprobarMigraciones lee el estado de las migraciones con el LOGIN.
func ComprobarMigraciones(ctx context.Context, pool *pgxpool.Pool) (EstadoMigraciones, error) {
	var e EstadoMigraciones
	if ctx == nil || pool == nil {
		return e, ports.ErrNoDisponible
	}
	if err := pool.QueryRow(ctx, ConsultaMigraciones).Scan(&e.AD389, &e.AD390, &e.Seleccion1); err != nil {
		return e, errors.Join(ports.ErrNoDisponible, err)
	}
	return e, nil
}

// Repositorio implementa RepositorioSolicitudes y RegistroConvocatorias.
type Repositorio struct {
	pool *pgxpool.Pool
}

var (
	_ ports.RepositorioSolicitudes = (*Repositorio)(nil)
	_ ports.RegistroConvocatorias  = (*Repositorio)(nil)
)

// NuevoRepositorio usa el pool del LOGIN miembro de vec_seleccion_ejecutor.
func NuevoRepositorio(pool *pgxpool.Pool) (*Repositorio, error) {
	if pool == nil {
		return nil, ports.ErrNoDisponible
	}
	return &Repositorio{pool: pool}, nil
}

func (r *Repositorio) abrir(ctx context.Context) (pgx.Tx, error) {
	if ctx == nil || r == nil || r.pool == nil {
		return nil, ports.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, traducir(ctx, err)
	}
	if _, err = tx.Exec(ctx, configuracionTransaccion); err != nil {
		revertir(tx)
		return nil, traducir(ctx, err)
	}
	return tx, nil
}

func revertir(tx pgx.Tx) {
	if tx == nil {
		return
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

// traducir convierte los códigos de vec_seleccion en errores del puerto; lo
// demás es indisponibilidad (nunca éxito ni autorización).
func traducir(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "VSL01":
			return ports.ErrFueraDePlazo
		case "VSL02":
			return ports.ErrClaveReutilizada
		case "VSL03":
			return ports.ErrVersionObsoleta
		case "VSL04":
			return ports.ErrYaPresentada
		case "VSL05":
			return ports.ErrRequisitoNoCumplido
		case "VSL06":
			return ports.ErrConvocatoriaActualizada
		case "VSL07":
			return ports.ErrConvocatoriaNoDisponible
		case "VSL08":
			return ports.ErrSolicitudNoEncontrada
		case "VSL09":
			return ports.ErrDatosIncompletos
		case "42501":
			return errors.Join(dominiovec.ErrAutorizacionDenegada, ports.ErrDatosNoValidos)
		case "22023", "23514", "23503":
			return ports.ErrDatosNoValidos
		}
	}
	return errors.Join(ports.ErrNoDisponible, err)
}

func materialValido(m ports.MaterialConsumoV3) bool {
	return m != nil && len(m.CapacidadCanonica()) != 0 && len(m.DecisionCanonica()) != 0 && len(m.ContextoActorCanonico()) != 0
}

// argumentosMaterial son los diez argumentos de consumo, en el orden de las
// fachadas AD3.
func argumentosMaterial(m ports.MaterialConsumoV3) []any {
	return []any{m.CapacidadCanonica(), m.DecisionCanonica(), m.MotivoCanonico(), m.ContextoActorCanonico(),
		int64(m.PersonaVersion()), int64(m.PerfilVersion()), m.PayloadVECAD3(), m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI()}
}

const parametrosMaterial = `$%d::bytea,$%d::bytea,$%d::bytea,$%d::bytea,$%d::numeric,$%d::numeric,$%d::bytea,$%d::bytea,$%d::bytea,$%d::bytea`

// ejecutar abre la transacción, ejecuta una función que devuelve una fila y
// confirma.
func (r *Repositorio) ejecutar(ctx context.Context, sql string, args []any, destino ...any) error {
	tx, err := r.abrir(ctx)
	if err != nil {
		return err
	}
	defer revertir(tx)
	if err := tx.QueryRow(ctx, sql, args...).Scan(destino...); err != nil {
		return traducir(ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return traducir(ctx, err)
	}
	return nil
}

func puntos(micropuntos int64) (baremacion.Puntos, error) {
	p, err := baremacion.PuntosDesdeMicropuntos(micropuntos)
	if err != nil {
		return baremacion.Puntos{}, ports.ErrNoDisponible
	}
	return p, nil
}

// PublicarConvocatoria publica la convocatoria con su huella y su instante
// fijo; la base reutiliza la versión vigente si el contenido coincide.
func (r *Repositorio) PublicarConvocatoria(ctx context.Context, c domain.Convocatoria) (int, bool, error) {
	contenido, err := c.ContenidoJSON()
	if err != nil {
		return 0, false, ports.ErrDatosNoValidos
	}
	huella, err := c.HuellaSHA256()
	if err != nil {
		return 0, false, ports.ErrDatosNoValidos
	}
	var version int
	var nueva bool
	err = r.ejecutar(ctx, `SELECT version, nueva FROM vec_seleccion.publicar_convocatoria_v1($1::text,$2::text,$3::timestamptz,$4::timestamptz,$5::jsonb,$6::text,$7::timestamptz)`,
		[]any{c.Ref, c.Titulo, c.AbreEn.UTC(), c.CierraEn.UTC(), string(contenido), huella, c.PublicadaEn.UTC()}, &version, &nueva)
	if err != nil {
		return 0, false, err
	}
	return version, nueva, nil
}

type convocatoriaFila struct {
	Ref       string          `json:"convocatoria_ref"`
	Version   int             `json:"version"`
	Titulo    string          `json:"titulo"`
	AbreEn    time.Time       `json:"abre_en"`
	CierraEn  time.Time       `json:"cierra_en"`
	Abierta   bool            `json:"abierta"`
	Publicada time.Time       `json:"publicada_en"`
	Contenido json.RawMessage `json:"contenido"`
}

// ConvocatoriasVigentes lee la versión vigente de cada convocatoria con su
// plazo evaluado por el reloj de la base.
func (r *Repositorio) ConvocatoriasVigentes(ctx context.Context) ([]domain.ConvocatoriaPublicada, error) {
	var bruto []byte
	if err := r.ejecutar(ctx, `SELECT vec_seleccion.convocatorias_publicadas_v1()`, nil, &bruto); err != nil {
		return nil, err
	}
	var filas []convocatoriaFila
	if json.Unmarshal(bruto, &filas) != nil {
		return nil, ports.ErrNoDisponible
	}
	resultado := make([]domain.ConvocatoriaPublicada, 0, len(filas))
	for _, f := range filas {
		c, err := domain.ConvocatoriaDesdeContenido(f.Ref, f.Titulo, f.AbreEn, f.CierraEn, f.Publicada, f.Contenido)
		if err != nil || f.Version < 1 {
			return nil, ports.ErrNoDisponible
		}
		resultado = append(resultado, domain.ConvocatoriaPublicada{Convocatoria: c, Version: f.Version, Abierta: f.Abierta})
	}
	return resultado, nil
}

func decodificarSobre(clave, nonce, cifrado string) (ports.SobreDatos, error) {
	n, e1 := base64.StdEncoding.DecodeString(nonce)
	c, e2 := base64.StdEncoding.DecodeString(cifrado)
	if e1 != nil || e2 != nil || clave == "" || len(n) == 0 || len(c) == 0 {
		return ports.SobreDatos{}, ports.ErrNoDisponible
	}
	return ports.SobreDatos{ClaveRef: clave, Nonce: n, Cifrado: c}, nil
}
