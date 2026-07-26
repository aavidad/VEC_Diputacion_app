package postgres

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

type origenAcreditacionPoolO405PostgreSQL struct {
	dependencia *PoolRecuperacionCoberturaO405PostgreSQL
}

type conexionAcreditacionPoolO405PostgreSQL struct {
	conexion *pgxpool.Conn
	sello    *selloFabricaPoolO405
}

type iniciadorTransaccionesAcreditadoO405 struct {
	origen     origenAcreditacionPoolO405
	modo       modoTLSAcreditacionPoolO405
	oidFuncion uint32
}

type transaccionAcreditadaO405 struct {
	pgx.Tx
	conexion        conexionAcreditacionPoolO405
	muFinalizacion  sync.Mutex
	finalizada      bool
	liberarUnaVez   sync.Once
	falloLiberacion bool
	oidFuncion      uint32
	tlsEsperado     bool
}

// NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL
// comprueba readiness y, en cada operación, sella el transporte de la conexión
// física. Catálogos vivos y llamada se acreditan después en una única sentencia
// de la transacción. Nunca cierra el pool recibido.
func NuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
	ctx context.Context,
	dependencia *PoolRecuperacionCoberturaO405PostgreSQL,
) (*EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL, error) {
	return nuevoEjecutorRecuperacionCoberturaO405Acreditado(
		ctx,
		&origenAcreditacionPoolO405PostgreSQL{
			dependencia: dependencia,
		},
		modoTLSAcreditacionPoolO405Produccion,
	)
}

func nuevoEjecutorRecuperacionCoberturaO405Acreditado(
	ctx context.Context,
	origen origenAcreditacionPoolO405,
	modo modoTLSAcreditacionPoolO405,
) (*EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL, error) {
	oidFuncion, err := acreditarPoolRecuperacionCoberturaO405ConManifiesto(
		ctx,
		origen,
		modo,
	)
	if err != nil {
		return nil, err
	}
	return nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
		&iniciadorTransaccionesAcreditadoO405{
			origen: origen, modo: modo, oidFuncion: oidFuncion,
		},
	)
}

func (o *origenAcreditacionPoolO405PostgreSQL) Configuracion() *pgxpool.Config {
	if o == nil || !dependenciaPoolRecuperacionO405Valida(o.dependencia) {
		return nil
	}
	return o.dependencia.pool.Config()
}

func (o *origenAcreditacionPoolO405PostgreSQL) Sello() *selloFabricaPoolO405 {
	if o == nil || !dependenciaPoolRecuperacionO405Valida(o.dependencia) {
		return nil
	}
	return o.dependencia.sello
}

func (o *origenAcreditacionPoolO405PostgreSQL) Adquirir(
	ctx context.Context,
) (conexionAcreditacionPoolO405, error) {
	if o == nil || !dependenciaPoolRecuperacionO405Valida(o.dependencia) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	conexion, err := o.dependencia.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	return &conexionAcreditacionPoolO405PostgreSQL{
		conexion: conexion,
		sello:    o.Sello(),
	}, nil
}

func (c *conexionAcreditacionPoolO405PostgreSQL) Configuracion() *pgx.ConnConfig {
	if c == nil || c.conexion == nil || c.conexion.Conn() == nil {
		return nil
	}
	return c.conexion.Conn().Config()
}

func (c *conexionAcreditacionPoolO405PostgreSQL) Sello() *selloFabricaPoolO405 {
	if c == nil {
		return nil
	}
	return c.sello
}

func (c *conexionAcreditacionPoolO405PostgreSQL) QueryRow(
	ctx context.Context,
	consulta string,
	argumentos ...any,
) pgx.Row {
	return c.conexion.QueryRow(ctx, consulta, argumentos...)
}

func (c *conexionAcreditacionPoolO405PostgreSQL) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (pgx.Tx, error) {
	return c.conexion.BeginTx(ctx, opciones)
}

func (c *conexionAcreditacionPoolO405PostgreSQL) Liberar() {
	if c != nil && c.conexion != nil {
		c.conexion.Release()
	}
}

func (i *iniciadorTransaccionesAcreditadoO405) BeginTx(
	ctx context.Context,
	opciones pgx.TxOptions,
) (txResultado pgx.Tx, errResultado error) {
	var conexion conexionAcreditacionPoolO405
	liberar := false
	defer func() {
		panico := recover()
		falloLiberacion := liberar &&
			liberarConexionAcreditacionO405(conexion)
		if panico != nil || falloLiberacion {
			txResultado = nil
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
		}
	}()
	if dependenciaNula(ctx) || i == nil || i.oidFuncion == 0 ||
		dependenciaNula(i.origen) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	selloEsperado := i.origen.Sello()
	if !selloAcreditacionO405Valido(selloEsperado, i.modo) ||
		!configuracionPoolAcreditacionO405Valida(
			i.origen.Configuracion(),
			i.modo,
		) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var err error
	conexion, err = i.origen.Adquirir(ctx)
	if !dependenciaNula(conexion) {
		liberar = true
	}
	if err != nil || dependenciaNula(conexion) {
		return nil, errorAcreditacionPoolO405(ctx)
	}
	if conexion.Sello() != selloEsperado ||
		!configuracionConexionAcreditacionO405Valida(
			conexion.Configuracion(),
			i.modo,
		) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	tx, err := conexion.BeginTx(ctx, opciones)
	if err != nil || dependenciaNula(tx) {
		return nil, errorAcreditacionPoolO405(ctx)
	}
	liberar = false
	return &transaccionAcreditadaO405{
		Tx: tx, conexion: conexion, oidFuncion: i.oidFuncion,
		tlsEsperado: i.modo == modoTLSAcreditacionPoolO405Produccion,
	}, nil
}

func (t *transaccionAcreditadaO405) oidFuncionRecuperacionCoberturaO405() uint32 {
	if t == nil {
		return 0
	}
	return t.oidFuncion
}

func (t *transaccionAcreditadaO405) tlsEsperadoRecuperacionCoberturaO405() bool {
	return t != nil && t.tlsEsperado
}

func dependenciaPoolRecuperacionO405Valida(
	dependencia *PoolRecuperacionCoberturaO405PostgreSQL,
) bool {
	return dependencia != nil && dependencia.pool != nil &&
		dependencia.cierre != nil &&
		dependencia.sello != nil &&
		dependencia.sello.dependencia == dependencia
}

func (t *transaccionAcreditadaO405) Commit(
	ctx context.Context,
) (errResultado error) {
	return t.finalizar(ctx, true)
}

func (t *transaccionAcreditadaO405) Rollback(
	ctx context.Context,
) (errResultado error) {
	return t.finalizar(ctx, false)
}

func (t *transaccionAcreditadaO405) finalizar(
	ctx context.Context,
	confirmar bool,
) (errResultado error) {
	debeLiberar := false
	defer func() {
		panico := recover()
		falloLiberacion := debeLiberar && t != nil && t.liberar()
		if falloLiberacion || panico != nil {
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
		}
	}()
	if t == nil || dependenciaNula(t.Tx) {
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	t.muFinalizacion.Lock()
	if t.finalizada {
		t.muFinalizacion.Unlock()
		return pgx.ErrTxClosed
	}
	t.finalizada = true
	t.muFinalizacion.Unlock()
	debeLiberar = true
	if confirmar {
		return t.Tx.Commit(ctx)
	}
	return t.Tx.Rollback(ctx)
}

func (t *transaccionAcreditadaO405) liberar() bool {
	if t == nil {
		return true
	}
	t.liberarUnaVez.Do(func() {
		t.falloLiberacion =
			liberarConexionAcreditacionO405(t.conexion)
	})
	return t.falloLiberacion
}

func liberarConexionAcreditacionO405(
	conexion conexionAcreditacionPoolO405,
) (fallo bool) {
	defer func() {
		if recover() != nil {
			fallo = true
		}
	}()
	if dependenciaNula(conexion) {
		return true
	}
	conexion.Liberar()
	return false
}
