package postgres

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const rolResolutorMotivosRRHHPostgreSQL = "vec_autorizacion_motivos_rrhh_resolutor"

var patronLoginResolutorMotivosRRHH = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

type origenPoolResolucionMotivosRRHH interface {
	Configuracion() *pgxpool.Config
	Sello() *selloPoolResolucionMotivosRRHH
	VincularSello(*selloPoolResolucionMotivosRRHH) bool
	Adquirir(context.Context) (conexionPoolResolucionMotivosRRHH, error)
	Cerrar()
}

type conexionPoolResolucionMotivosRRHH interface {
	Configuracion() *pgx.ConnConfig
	Sello() *selloPoolResolucionMotivosRRHH
	QueryRow(context.Context, string, ...any) pgx.Row
	BeginTx(
		context.Context,
		pgx.TxOptions,
	) (transaccionPoolResolucionMotivosRRHH, error)
	Liberar()
}

type transaccionPoolResolucionMotivosRRHH interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Commit(context.Context) error
	Rollback(context.Context) error
	Sello() *selloPoolResolucionMotivosRRHH
}

type origenPoolResolucionMotivosRRHHPostgreSQL struct {
	pool  *pgxpool.Pool
	sello *selloPoolResolucionMotivosRRHH
}

type conexionPoolResolucionMotivosRRHHPostgreSQL struct {
	conexion *pgxpool.Conn
	sello    *selloPoolResolucionMotivosRRHH
}

type transaccionPoolResolucionMotivosRRHHPostgreSQL struct {
	pgx.Tx
	sello *selloPoolResolucionMotivosRRHH
}

type creadorOrigenPoolResolucionMotivosRRHH func(
	context.Context,
	*pgxpool.Config,
) (origenPoolResolucionMotivosRRHH, error)

// PoolResolucionMotivosRRHHPostgreSQL es el único contenedor de conexiones
// admisible para las dos resoluciones nominales. No expone el pool subyacente.
type PoolResolucionMotivosRRHHPostgreSQL struct {
	origen     origenPoolResolucionMotivosRRHH
	login      string
	modoTLS    modoTLSAcreditacionPoolO405
	oidCuadro  uint32
	oidDetalle uint32
	sello      *selloPoolResolucionMotivosRRHH
	cierre     *cierrePoolResolucionMotivosRRHH
}

type selloPoolResolucionMotivosRRHH struct {
	dependencia              *PoolResolucionMotivosRRHHPostgreSQL
	login                    string
	modo                     modoTLSAcreditacionPoolO405
	oidCuadro                uint32
	oidDetalle               uint32
	callbacksPredeterminados bool
}

type cierrePoolResolucionMotivosRRHH struct {
	unaVez sync.Once
	cerrar func()
}

// NuevoPoolResolucionMotivosRRHHPostgreSQL crea, acredita y entrega un pool
// exclusivo. La identidad de la DSN y el LOGIN declarado han de coincidir.
func NuevoPoolResolucionMotivosRRHHPostgreSQL(
	ctx context.Context,
	cadenaConexion string,
	loginNominal string,
) (*PoolResolucionMotivosRRHHPostgreSQL, error) {
	return nuevoPoolResolucionMotivosRRHHPostgreSQL(
		ctx, cadenaConexion, loginNominal, modoTLSAcreditacionPoolO405Produccion,
	)
}

func nuevoPoolResolucionMotivosRRHHPostgreSQL(
	ctx context.Context,
	cadenaConexion string,
	loginNominal string,
	modo modoTLSAcreditacionPoolO405,
) (*PoolResolucionMotivosRRHHPostgreSQL, error) {
	if dependenciaNula(ctx) || !loginResolutorMotivosRRHHValido(loginNominal) {
		return nil, ports.ErrMotivoConsultaRRHHNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	configuracion, err := pgxpool.ParseConfig(cadenaConexion)
	if err != nil || configuracion == nil || configuracion.ConnConfig == nil ||
		configuracion.ConnConfig.User != loginNominal ||
		!endurecerTLSFabricaPoolO405(&configuracion.ConnConfig.Config, modo) ||
		!configuracionPoolAcreditacionO405Valida(configuracion, modo) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	return construirPoolResolucionMotivosRRHH(
		ctx, configuracion, loginNominal, modo,
		crearOrigenPoolResolucionMotivosRRHHPostgreSQL,
	)
}

func construirPoolResolucionMotivosRRHH(
	ctx context.Context,
	configuracion *pgxpool.Config,
	loginNominal string,
	modo modoTLSAcreditacionPoolO405,
	crear creadorOrigenPoolResolucionMotivosRRHH,
) (resultado *PoolResolucionMotivosRRHHPostgreSQL, errResultado error) {
	var origen origenPoolResolucionMotivosRRHH
	transferido := false
	defer func() {
		panico := recover()
		if !transferido && !dependenciaNula(origen) {
			cerrarOrigenResolucionMotivosRRHH(origen)
		}
		if panico != nil {
			resultado, errResultado = nil, errorPoolResolucionMotivosRRHH(ctx)
		}
	}()
	if dependenciaNula(ctx) || crear == nil ||
		!loginResolutorMotivosRRHHValido(loginNominal) ||
		!configuracionPoolResolucionMotivosRRHHValida(
			configuracion, loginNominal, modo,
		) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	var err error
	origen, err = crear(ctx, configuracion)
	if err != nil || dependenciaNula(origen) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	resultado = &PoolResolucionMotivosRRHHPostgreSQL{
		origen: origen, login: loginNominal, modoTLS: modo,
		cierre: &cierrePoolResolucionMotivosRRHH{
			cerrar: origen.Cerrar,
		},
	}
	resultado.sello = &selloPoolResolucionMotivosRRHH{
		dependencia: resultado, login: loginNominal, modo: modo,
		callbacksPredeterminados: true,
	}
	if !origen.VincularSello(resultado.sello) ||
		origen.Sello() != resultado.sello {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	oidCuadro, oidDetalle, err := resultado.acreditarInicial(ctx)
	if err != nil || oidCuadro == 0 || oidDetalle == 0 || oidCuadro == oidDetalle {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	resultado.oidCuadro, resultado.oidDetalle = oidCuadro, oidDetalle
	resultado.sello.oidCuadro, resultado.sello.oidDetalle = oidCuadro, oidDetalle
	if !selloPoolResolucionMotivosRRHHValido(resultado.sello, resultado, true) {
		return nil, errorPoolResolucionMotivosRRHH(ctx)
	}
	transferido = true
	return resultado, nil
}

func crearOrigenPoolResolucionMotivosRRHHPostgreSQL(
	ctx context.Context,
	configuracion *pgxpool.Config,
) (origenPoolResolucionMotivosRRHH, error) {
	pool, err := pgxpool.NewWithConfig(ctx, configuracion)
	if err != nil || pool == nil {
		return nil, err
	}
	return &origenPoolResolucionMotivosRRHHPostgreSQL{pool: pool}, nil
}

func (o *origenPoolResolucionMotivosRRHHPostgreSQL) Configuracion() *pgxpool.Config {
	if o == nil || o.pool == nil {
		return nil
	}
	return o.pool.Config()
}

func (o *origenPoolResolucionMotivosRRHHPostgreSQL) Sello() *selloPoolResolucionMotivosRRHH {
	if o == nil {
		return nil
	}
	return o.sello
}

func (o *origenPoolResolucionMotivosRRHHPostgreSQL) VincularSello(
	sello *selloPoolResolucionMotivosRRHH,
) bool {
	if o == nil || o.pool == nil || o.sello != nil || sello == nil {
		return false
	}
	o.sello = sello
	return true
}

func (o *origenPoolResolucionMotivosRRHHPostgreSQL) Adquirir(
	ctx context.Context,
) (conexionPoolResolucionMotivosRRHH, error) {
	if o == nil || o.pool == nil || o.sello == nil {
		return nil, ports.ErrMotivoConsultaRRHHNoDisponible
	}
	conexion, err := o.pool.Acquire(ctx)
	if err != nil || conexion == nil {
		return nil, err
	}
	return &conexionPoolResolucionMotivosRRHHPostgreSQL{
		conexion: conexion,
		sello:    o.sello,
	}, nil
}

func (o *origenPoolResolucionMotivosRRHHPostgreSQL) Cerrar() {
	if o != nil && o.pool != nil {
		o.pool.Close()
	}
}

func (c *conexionPoolResolucionMotivosRRHHPostgreSQL) Configuracion() *pgx.ConnConfig {
	if c == nil || c.conexion == nil || c.conexion.Conn() == nil {
		return nil
	}
	return c.conexion.Conn().Config()
}

func (c *conexionPoolResolucionMotivosRRHHPostgreSQL) Sello() *selloPoolResolucionMotivosRRHH {
	if c == nil {
		return nil
	}
	return c.sello
}

func (c *conexionPoolResolucionMotivosRRHHPostgreSQL) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	return c.conexion.QueryRow(ctx, consulta, argumentos...)
}

func (c *conexionPoolResolucionMotivosRRHHPostgreSQL) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (transaccionPoolResolucionMotivosRRHH, error) {
	tx, err := c.conexion.BeginTx(ctx, opciones)
	if err != nil || tx == nil {
		return nil, err
	}
	return &transaccionPoolResolucionMotivosRRHHPostgreSQL{
		Tx: tx, sello: c.sello,
	}, nil
}

func (t *transaccionPoolResolucionMotivosRRHHPostgreSQL) Sello() *selloPoolResolucionMotivosRRHH {
	if t == nil {
		return nil
	}
	return t.sello
}

func (c *conexionPoolResolucionMotivosRRHHPostgreSQL) Liberar() {
	if c != nil && c.conexion != nil {
		c.conexion.Release()
	}
}

func selloPoolResolucionMotivosRRHHValido(
	sello *selloPoolResolucionMotivosRRHH,
	pool *PoolResolucionMotivosRRHHPostgreSQL,
	exigirOID bool,
) bool {
	if sello == nil || pool == nil || sello.dependencia != pool ||
		pool.sello != sello || sello.login != pool.login ||
		sello.modo != pool.modoTLS || !sello.callbacksPredeterminados ||
		!loginResolutorMotivosRRHHValido(sello.login) {
		return false
	}
	if !exigirOID {
		return sello.oidCuadro == 0 && sello.oidDetalle == 0 &&
			pool.oidCuadro == 0 && pool.oidDetalle == 0
	}
	return sello.oidCuadro != 0 && sello.oidDetalle != 0 &&
		sello.oidCuadro != sello.oidDetalle &&
		sello.oidCuadro == pool.oidCuadro &&
		sello.oidDetalle == pool.oidDetalle
}

func loginResolutorMotivosRRHHValido(login string) bool {
	return login != "" && login != rolResolutorMotivosRRHHPostgreSQL &&
		login == strings.TrimSpace(login) &&
		patronLoginResolutorMotivosRRHH.MatchString(login)
}

// Cerrar materializa el ownership del llamador. Es idempotente y una copia
// accidental del wrapper no puede cerrar el pool original.
func (p *PoolResolucionMotivosRRHHPostgreSQL) Cerrar() {
	defer func() { _ = recover() }()
	if p == nil || dependenciaNula(p.origen) || p.cierre == nil ||
		p.cierre.cerrar == nil ||
		!selloPoolResolucionMotivosRRHHValido(p.sello, p, true) {
		return
	}
	p.cierre.unaVez.Do(p.cierre.cerrar)
}

func cerrarOrigenResolucionMotivosRRHH(origen origenPoolResolucionMotivosRRHH) {
	defer func() { _ = recover() }()
	if !dependenciaNula(origen) {
		origen.Cerrar()
	}
}

func errorPoolResolucionMotivosRRHH(
	ctx context.Context,
) (resultado error) {
	resultado = ports.ErrMotivoConsultaRRHHNoDisponible
	defer func() {
		if recover() != nil {
			resultado = ports.ErrMotivoConsultaRRHHNoDisponible
		}
	}()
	if !dependenciaNula(ctx) {
		if err := ctx.Err(); err != nil {
			return errors.Join(err, resultado)
		}
	}
	return resultado
}
