// Package confianzaatestacionv2 carga la lista positiva VEC-AD-2 desde una
// autoridad PostgreSQL aislada. No firma, no emite capacidades y no abre una
// via alternativa a la funcion SQL gobernada.
package confianzaatestacionv2

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrCargaConfianzaAtestacionV2NoDisponible = errors.New(
	"vec: carga PostgreSQL de confianza de atestacion V2 no disponible",
)

const (
	// RolLectorAutoridadPostgreSQL es un rol NOLOGIN sin herencias. El LOGIN
	// aislado del proceso debe poder asumirlo, pero no se acepta como autoridad.
	RolLectorAutoridadPostgreSQL = "vec_confianza_atestacion_v2_lector_autoridad"

	maximasRaicesConfiables = 64
	tiempoMaximoRollback    = 2 * time.Second
)

const consultaConfianzaActual = `
	SELECT revision, huella_configuracion_sha256,
	       configuracion_publicada_en, configuracion_expira_en,
	       configuracion_estado, configuracion_revocada_en,
	       clave_id, algoritmo_cose, suite, audiencia_despliegue,
	       clave_publica_spki, huella_clave_spki_sha256,
	       raiz_valida_desde, raiz_valida_hasta,
	       raiz_estado, raiz_revocada_en
	  FROM vec_confianza_atestacion_v2.obtener_confianza_actual()`

const sentenciaBloqueoGobierno = `
	SELECT pg_catalog.pg_advisory_xact_lock_shared(
		pg_catalog.hashtextextended(
			'vec_confianza_atestacion_v2:gobierno:v1', 0
		)
	)`

type iniciadorTransaccion interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// CargarConfiguracionActual reconstruye la revision completa mediante un pool
// concreto. El llamante no puede inyectar una consulta, una identidad ni una
// fuente alternativa en este constructor productivo.
func CargarConfiguracionActual(
	ctx context.Context,
	pool *pgxpool.Pool,
) (confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV2, error) {
	if pool == nil {
		return confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV2{},
			ErrCargaConfianzaAtestacionV2NoDisponible
	}
	return cargarConfiguracionActual(ctx, pool)
}

// NuevoServicioActual carga una unica instantanea autoritativa y construye el
// verificador con el conector de tiempo elegido por la composicion. La fuente
// PostgreSQL sigue siendo concreta y no admite repositorios alternativos.
func NuevoServicioActual(
	ctx context.Context,
	pool *pgxpool.Pool,
	reloj ports.Reloj,
) (*confianzaatestacion.ServicioConfianzaAtestacionAutorizacionV2, error) {
	if pool == nil || valorNulo(reloj) {
		return nil, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	return nuevoServicioActual(ctx, pool, reloj)
}

func nuevoServicioActual(
	ctx context.Context,
	iniciador iniciadorTransaccion,
	reloj ports.Reloj,
) (*confianzaatestacion.ServicioConfianzaAtestacionAutorizacionV2, error) {
	if valorNulo(reloj) {
		return nil, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	configuracion, err := cargarConfiguracionActual(ctx, iniciador)
	if err != nil {
		return nil, err
	}
	servicio, err := confianzaatestacion.NuevoServicioConfianzaAtestacionAutorizacionV2(
		configuracion,
		reloj,
	)
	if err != nil {
		return nil, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	return servicio, nil
}

func cargarConfiguracionActual(
	ctx context.Context,
	iniciador iniciadorTransaccion,
) (confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV2, error) {
	vacia := confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV2{}
	if ctx == nil || valorNulo(iniciador) {
		return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	tx, err := iniciador.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.NotDeferrable,
	})
	if err != nil || valorNulo(tx) {
		return vacia, errorCarga(ctx, err)
	}
	defer cancelarTransaccion(tx)

	if err = prepararTransaccion(ctx, tx); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if err = adquirirBloqueoGobierno(ctx, tx); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	instanteConsulta, err := comprobarIdentidad(ctx, tx)
	if err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}

	filas, err := tx.Query(ctx, consultaConfianzaActual)
	if err != nil || valorNulo(filas) {
		return vacia, errorCarga(ctx, err)
	}
	defer filas.Close()

	var base *baseConfiguracion
	raices := make([]confianzaatestacion.RaizPublicaAtestacionAutorizacionV2, 0, 4)
	tieneRaizActivaVigente := false
	for filas.Next() {
		if err = ctx.Err(); err != nil {
			return vacia, errorCarga(ctx, err)
		}
		if len(raices) >= maximasRaicesConfiables {
			return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
		}
		fila, errFila := escanearFilaConfianza(filas)
		if errFila != nil {
			return vacia, errorCarga(ctx, errFila)
		}
		if fila.validar(instanteConsulta) != nil {
			return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
		}
		actual := fila.base()
		if base == nil {
			copia := actual
			base = &copia
		} else if !base.igual(actual) {
			return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
		}
		raiz, errRaiz := fila.construirRaiz()
		if errRaiz != nil {
			return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
		}
		raices = append(raices, raiz)
		if fila.raizEstado == string(confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Activa) &&
			!instanteConsulta.Before(fila.raizValidaDesde) &&
			instanteConsulta.Before(fila.raizValidaHasta) {
			tieneRaizActivaVigente = true
		}
	}
	filas.Close()
	if err = filas.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if err = ctx.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if base == nil || len(raices) == 0 || !tieneRaizActivaVigente {
		return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
	}

	configuracion, err := confianzaatestacion.NuevaConfiguracionConfianzaAtestacionAutorizacionV2(
		base.revision,
		base.publicadaEn,
		base.expiraEn,
		raices...,
	)
	if err != nil || configuracion.ValidarHuellaSHA256Esperada(base.huellaSHA256) != nil {
		return vacia, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	if err = ctx.Err(); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return vacia, errorCarga(ctx, err)
	}
	return configuracion, nil
}

func prepararTransaccion(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `SET LOCAL ROLE `+RolLectorAutoridadPostgreSQL); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `
		SELECT pg_catalog.set_config('search_path', 'pg_catalog', true),
		       pg_catalog.set_config('row_security', 'on', true),
		       pg_catalog.set_config('timezone', 'UTC', true),
		       pg_catalog.set_config('lock_timeout', '2s', true),
		       pg_catalog.set_config('statement_timeout', '8s', true),
		       pg_catalog.set_config('idle_in_transaction_session_timeout', '10s', true)`)
	return err
}

func adquirirBloqueoGobierno(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, sentenciaBloqueoGobierno)
	return err
}

type identidadTransaccion struct {
	sesionUsuario       string
	usuarioActual       string
	instanteConsulta    time.Time
	sesionPuedeLogin    bool
	sesionSuperusuario  bool
	sesionCreaRoles     bool
	sesionCreaBases     bool
	sesionReplica       bool
	sesionEvitaRLS      bool
	sesionMembresias    int64
	sesionMiembroLector bool
	actualPuedeLogin    bool
	actualSuperusuario  bool
	actualCreaRoles     bool
	actualCreaBases     bool
	actualReplica       bool
	actualEvitaRLS      bool
	actualSinMembresias bool
}

func comprobarIdentidad(ctx context.Context, tx pgx.Tx) (time.Time, error) {
	var identidad identidadTransaccion
	err := tx.QueryRow(ctx, `
		SELECT session_user::text, current_user::text,
		       pg_catalog.clock_timestamp(),
		       sesion.rolcanlogin, sesion.rolsuper, sesion.rolcreaterole,
		       sesion.rolcreatedb, sesion.rolreplication, sesion.rolbypassrls,
		       (SELECT count(*)
		          FROM pg_catalog.pg_auth_members AS membresia_sesion
		         WHERE membresia_sesion.member = sesion.oid),
		       EXISTS (
		           SELECT 1
		             FROM pg_catalog.pg_auth_members AS membresia_lector
		            WHERE membresia_lector.member = sesion.oid
		              AND membresia_lector.roleid = actual.oid
		       ),
		       actual.rolcanlogin, actual.rolsuper, actual.rolcreaterole,
		       actual.rolcreatedb, actual.rolreplication, actual.rolbypassrls,
		       NOT EXISTS (
		           SELECT 1
		             FROM pg_catalog.pg_auth_members AS membresia
		            WHERE membresia.member = actual.oid
		       )
		  FROM pg_catalog.pg_roles AS sesion
		  JOIN pg_catalog.pg_roles AS actual
		    ON actual.rolname = current_user
		 WHERE sesion.rolname = session_user`).Scan(
		&identidad.sesionUsuario,
		&identidad.usuarioActual,
		&identidad.instanteConsulta,
		&identidad.sesionPuedeLogin,
		&identidad.sesionSuperusuario,
		&identidad.sesionCreaRoles,
		&identidad.sesionCreaBases,
		&identidad.sesionReplica,
		&identidad.sesionEvitaRLS,
		&identidad.sesionMembresias,
		&identidad.sesionMiembroLector,
		&identidad.actualPuedeLogin,
		&identidad.actualSuperusuario,
		&identidad.actualCreaRoles,
		&identidad.actualCreaBases,
		&identidad.actualReplica,
		&identidad.actualEvitaRLS,
		&identidad.actualSinMembresias,
	)
	identidad.instanteConsulta = normalizarInstante(identidad.instanteConsulta)
	if err != nil || identidad.sesionUsuario == "" || !identidad.sesionPuedeLogin ||
		identidad.sesionSuperusuario || identidad.sesionCreaRoles || identidad.sesionCreaBases ||
		identidad.sesionReplica || identidad.sesionEvitaRLS || identidad.sesionMembresias != 1 ||
		!identidad.sesionMiembroLector ||
		identidad.usuarioActual != RolLectorAutoridadPostgreSQL || identidad.actualPuedeLogin ||
		identidad.actualSuperusuario || identidad.actualCreaRoles || identidad.actualCreaBases ||
		identidad.actualReplica || identidad.actualEvitaRLS || !identidad.actualSinMembresias ||
		!instanteCanonico(identidad.instanteConsulta) {
		return time.Time{}, ErrCargaConfianzaAtestacionV2NoDisponible
	}
	return identidad.instanteConsulta, nil
}

type baseConfiguracion struct {
	revision     string
	huellaSHA256 string
	publicadaEn  time.Time
	expiraEn     time.Time
	estado       string
	revocadaEn   pgtype.Timestamptz
}

func (b baseConfiguracion) igual(otra baseConfiguracion) bool {
	return b.revision == otra.revision && b.huellaSHA256 == otra.huellaSHA256 &&
		b.publicadaEn.Equal(otra.publicadaEn) && b.expiraEn.Equal(otra.expiraEn) &&
		b.estado == otra.estado && mismoInstanteNulo(b.revocadaEn, otra.revocadaEn)
}

type filaConfianza struct {
	revision                  string
	huellaConfiguracionSHA256 string
	configuracionPublicadaEn  time.Time
	configuracionExpiraEn     time.Time
	configuracionEstado       string
	configuracionRevocadaEn   pgtype.Timestamptz
	claveID                   string
	algoritmoCOSE             string
	suite                     string
	audienciaDespliegue       string
	clavePublicaSPKI          []byte
	huellaClaveSPKISHA256     string
	raizValidaDesde           time.Time
	raizValidaHasta           time.Time
	raizEstado                string
	raizRevocadaEn            pgtype.Timestamptz
}

func escanearFilaConfianza(filas pgx.Rows) (filaConfianza, error) {
	var fila filaConfianza
	err := filas.Scan(
		&fila.revision,
		&fila.huellaConfiguracionSHA256,
		&fila.configuracionPublicadaEn,
		&fila.configuracionExpiraEn,
		&fila.configuracionEstado,
		&fila.configuracionRevocadaEn,
		&fila.claveID,
		&fila.algoritmoCOSE,
		&fila.suite,
		&fila.audienciaDespliegue,
		&fila.clavePublicaSPKI,
		&fila.huellaClaveSPKISHA256,
		&fila.raizValidaDesde,
		&fila.raizValidaHasta,
		&fila.raizEstado,
		&fila.raizRevocadaEn,
	)
	if err != nil {
		return filaConfianza{}, err
	}
	fila.configuracionPublicadaEn = normalizarInstante(fila.configuracionPublicadaEn)
	fila.configuracionExpiraEn = normalizarInstante(fila.configuracionExpiraEn)
	fila.raizValidaDesde = normalizarInstante(fila.raizValidaDesde)
	fila.raizValidaHasta = normalizarInstante(fila.raizValidaHasta)
	fila.clavePublicaSPKI = bytes.Clone(fila.clavePublicaSPKI)
	if fila.configuracionRevocadaEn.Valid {
		fila.configuracionRevocadaEn.Time = normalizarInstante(fila.configuracionRevocadaEn.Time)
	}
	if fila.raizRevocadaEn.Valid {
		fila.raizRevocadaEn.Time = normalizarInstante(fila.raizRevocadaEn.Time)
	}
	return fila, nil
}

func (f filaConfianza) base() baseConfiguracion {
	return baseConfiguracion{
		revision: f.revision, huellaSHA256: f.huellaConfiguracionSHA256,
		publicadaEn: f.configuracionPublicadaEn, expiraEn: f.configuracionExpiraEn,
		estado: f.configuracionEstado, revocadaEn: f.configuracionRevocadaEn,
	}
}

func (f filaConfianza) validar(instanteConsulta time.Time) error {
	if f.configuracionEstado != "activa" || f.configuracionRevocadaEn.Valid ||
		!instanteCanonico(f.configuracionPublicadaEn) ||
		!instanteCanonico(f.configuracionExpiraEn) ||
		instanteConsulta.Before(f.configuracionPublicadaEn) ||
		!instanteConsulta.Before(f.configuracionExpiraEn) ||
		f.algoritmoCOSE != confianzaatestacion.AlgoritmoCOSEAtestacionAutorizacionV2EdDSA ||
		f.suite != confianzaatestacion.SuiteAtestacionAutorizacionV2COSEEdDSA ||
		!huellaSHA256Valida(f.huellaClaveSPKISHA256) || len(f.clavePublicaSPKI) == 0 ||
		huellaBytes(f.clavePublicaSPKI) != f.huellaClaveSPKISHA256 ||
		!instanteCanonico(f.raizValidaDesde) || !instanteCanonico(f.raizValidaHasta) {
		return ErrCargaConfianzaAtestacionV2NoDisponible
	}
	if f.raizRevocadaEn.Valid && f.raizRevocadaEn.InfinityModifier != pgtype.Finite {
		return ErrCargaConfianzaAtestacionV2NoDisponible
	}
	switch confianzaatestacion.EstadoClaveAtestacionAutorizacionV2(f.raizEstado) {
	case confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Activa:
		if f.raizRevocadaEn.Valid {
			return ErrCargaConfianzaAtestacionV2NoDisponible
		}
	case confianzaatestacion.EstadoClaveAtestacionAutorizacionV2Revocada:
		if !f.raizRevocadaEn.Valid || !instanteCanonico(f.raizRevocadaEn.Time) {
			return ErrCargaConfianzaAtestacionV2NoDisponible
		}
	default:
		return ErrCargaConfianzaAtestacionV2NoDisponible
	}
	return nil
}

func (f filaConfianza) construirRaiz() (
	confianzaatestacion.RaizPublicaAtestacionAutorizacionV2,
	error,
) {
	claveCruda, err := x509.ParsePKIXPublicKey(f.clavePublicaSPKI)
	if err != nil {
		return confianzaatestacion.RaizPublicaAtestacionAutorizacionV2{}, err
	}
	clavePublica, correcta := claveCruda.(ed25519.PublicKey)
	if !correcta || len(clavePublica) != ed25519.PublicKeySize {
		return confianzaatestacion.RaizPublicaAtestacionAutorizacionV2{},
			ErrCargaConfianzaAtestacionV2NoDisponible
	}
	spkiCanonico, err := x509.MarshalPKIXPublicKey(clavePublica)
	if err != nil || !bytes.Equal(spkiCanonico, f.clavePublicaSPKI) {
		return confianzaatestacion.RaizPublicaAtestacionAutorizacionV2{},
			ErrCargaConfianzaAtestacionV2NoDisponible
	}
	revocadaEn := time.Time{}
	if f.raizRevocadaEn.Valid {
		revocadaEn = f.raizRevocadaEn.Time
	}
	return confianzaatestacion.NuevaRaizPublicaAtestacionAutorizacionV2EdDSA(
		f.claveID,
		bytes.Clone(clavePublica),
		f.audienciaDespliegue,
		confianzaatestacion.EstadoClaveAtestacionAutorizacionV2(f.raizEstado),
		f.raizValidaDesde,
		f.raizValidaHasta,
		revocadaEn,
	)
}

func cancelarTransaccion(tx pgx.Tx) {
	ctx, cancelar := context.WithTimeout(context.Background(), tiempoMaximoRollback)
	defer cancelar()
	_ = tx.Rollback(ctx)
}

func errorCarga(ctx context.Context, causa error) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrCargaConfianzaAtestacionV2NoDisponible, err)
		}
	}
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		if errors.Is(causa, context.Canceled) {
			return errors.Join(ErrCargaConfianzaAtestacionV2NoDisponible, context.Canceled)
		}
		return errors.Join(ErrCargaConfianzaAtestacionV2NoDisponible, context.DeadlineExceeded)
	}
	return ErrCargaConfianzaAtestacionV2NoDisponible
}

func normalizarInstante(instante time.Time) time.Time {
	if instante.IsZero() {
		return time.Time{}
	}
	return instante.UTC()
}

func instanteCanonico(instante time.Time) bool {
	return !instante.IsZero() && instante.Location() == time.UTC &&
		instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func huellaBytes(datos []byte) string {
	suma := sha256.Sum256(datos)
	return hex.EncodeToString(suma[:])
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) {
		return false
	}
	contenido, err := hex.DecodeString(valor)
	return err == nil && len(contenido) == sha256.Size
}

func mismoInstanteNulo(primero, segundo pgtype.Timestamptz) bool {
	if primero.Valid != segundo.Valid || primero.InfinityModifier != segundo.InfinityModifier {
		return false
	}
	return !primero.Valid || primero.Time.Equal(segundo.Time)
}

func valorNulo(valor any) bool {
	if valor == nil {
		return true
	}
	reflejado := reflect.ValueOf(valor)
	switch reflejado.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflejado.IsNil()
	default:
		return false
	}
}
