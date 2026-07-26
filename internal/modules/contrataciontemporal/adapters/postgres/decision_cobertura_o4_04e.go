package postgres

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

const funcionConfirmarDecisionCoberturaO404E = "" +
	"vec_contratacion_temporal." +
	"confirmar_operacion_decision_cobertura_o404e_v1"

var (
	errAdaptadorDecisionCoberturaO404ENoDisponible = errors.New(
		"contratacion temporal: adaptador PostgreSQL O4-04E no disponible",
	)
	errSesionDecisionCoberturaO404EInvalida = errors.New(
		"contratacion temporal: sesion PostgreSQL O4-04E invalida",
	)
)

var _ cobertura.EjecutorSesionTCBOperacionDecisionCobertura = (*EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL)(nil)

// EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL aporta la única
// transacción física O4-04E. No conoce la orden funcional y nunca reintenta:
// cualquier fallo tras alcanzar Confirmar puede representar un COMMIT
// efectivo y corresponde al núcleo reconciliarlo contra el primario.
type EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(
	pool *pgxpool.Pool,
) (*EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL, error) {
	return nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(pool)
}

func nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(
	pool iniciadorTransacciones,
) (*EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return &EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL{
		pool: pool,
	}, nil
}

func (e *EjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL) EjecutarSesionTCB(
	ctx context.Context,
	usar func(cobertura.SesionTCBOperacionDecisionCobertura) error,
) error {
	if ctx == nil || e == nil || dependenciaNula(e.pool) || usar == nil {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	})
	if err != nil {
		return err
	}
	defer revertirTransaccion(tx)
	if err := configurarSesionDecisionCoberturaO404E(ctx, tx); err != nil {
		return err
	}

	ctxCiclo, cancelarCiclo := context.WithCancel(ctx)
	guardia := nuevaGuardiaCicloDecisionCoberturaO404E()
	sesion := nuevaSesionDecisionCoberturaO404E(tx, ctxCiclo, guardia)
	cerrada := false
	defer func() {
		guardia.cerrar()
		cancelarCiclo()
		if !cerrada {
			sesion.cerrar()
		}
	}()
	errCallback := usar(sesion)
	violacion := guardia.cerrar()
	cancelarCiclo()
	confirmada := sesion.cerrar()
	cerrada = true
	if errCallback != nil {
		return errCallback
	}
	if violacion || !confirmada {
		return errSesionDecisionCoberturaO404EInvalida
	}
	// Un solo intento. En particular, no se repite ante 40001, 40P01,
	// cancelación ni pérdida de la respuesta del servidor.
	return tx.Commit(ctx)
}

func configurarSesionDecisionCoberturaO404E(
	ctx context.Context,
	tx pgx.Tx,
) error {
	if ctx == nil || dependenciaNula(tx) {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	_, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config(
		           'idle_in_transaction_session_timeout',
		           '20s',
		           true
		       )`)
	return err
}

type estadoSesionDecisionCoberturaO404E uint8

const (
	estadoSesionDecisionCoberturaNueva estadoSesionDecisionCoberturaO404E = iota
	estadoSesionDecisionCoberturaAbierta
	estadoSesionDecisionCoberturaGobernada
	estadoSesionDecisionCoberturaVEC
	estadoSesionDecisionCoberturaC1
	estadoSesionDecisionCoberturaLista
	estadoSesionDecisionCoberturaConsumida
	estadoSesionDecisionCoberturaInvalida
	estadoSesionDecisionCoberturaCerrada
)

// sesionDecisionCoberturaO404E acumula fragmentos defensivos sin I/O hasta
// Confirmar. El mutex hace determinista el rechazo de llamadas concurrentes;
// cerrar invalida además cualquier referencia que escapase del callback.
type sesionDecisionCoberturaO404E struct {
	mu  sync.Mutex
	tx  pgx.Tx
	ctx context.Context

	guardia *guardiaCicloDecisionCoberturaO404E

	estado estadoSesionDecisionCoberturaO404E
	carga  cargaConfirmarDecisionCoberturaO404E

	rama           cobertura.RamaSesionTCBOperacionDecisionCobertura
	totalC1        uint64
	siguienteC1    uint64
	peticionesC1   map[string]struct{}
	respuestasC1   map[claveRespuestaC1DecisionCoberturaO404E]struct{}
	bytesCanonicos int
	confirmada     bool
}

type claveRespuestaC1DecisionCoberturaO404E struct {
	autoridadRef       string
	generacion         uint32
	reciboRespuestaRef string
}

var _ cobertura.SesionTCBOperacionDecisionCobertura = (*sesionDecisionCoberturaO404E)(nil)

func nuevaSesionDecisionCoberturaO404E(
	tx pgx.Tx,
	ctx context.Context,
	guardia *guardiaCicloDecisionCoberturaO404E,
) *sesionDecisionCoberturaO404E {
	return &sesionDecisionCoberturaO404E{
		tx: tx, ctx: ctx, guardia: guardia,
		estado: estadoSesionDecisionCoberturaNueva,
	}
}

func (s *sesionDecisionCoberturaO404E) invalidar() error {
	if s != nil {
		s.estado = estadoSesionDecisionCoberturaInvalida
		s.limpiar()
	}
	return errSesionDecisionCoberturaO404EInvalida
}

func (s *sesionDecisionCoberturaO404E) cerrar() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	confirmada := s.confirmada &&
		s.estado == estadoSesionDecisionCoberturaConsumida
	s.estado = estadoSesionDecisionCoberturaCerrada
	s.tx = nil
	s.ctx = nil
	s.limpiar()
	return confirmada
}

func (s *sesionDecisionCoberturaO404E) limpiar() {
	limpiarCargaConfirmarDecisionCoberturaO404E(&s.carga)
	s.rama = ""
	s.totalC1 = 0
	s.siguienteC1 = 0
	s.peticionesC1 = nil
	s.respuestasC1 = nil
	s.bytesCanonicos = 0
}

// guardiaCicloDecisionCoberturaO404E cierra la ventana de uso exactamente
// cuando retorna el callback. Si una operación seguía viva, o dos intentaron
// entrar a la vez, el ejecutor deniega el COMMIT.
type guardiaCicloDecisionCoberturaO404E struct {
	activa    atomic.Bool
	enCurso   atomic.Bool
	violacion atomic.Bool
}

func nuevaGuardiaCicloDecisionCoberturaO404E() *guardiaCicloDecisionCoberturaO404E {
	g := &guardiaCicloDecisionCoberturaO404E{}
	g.activa.Store(true)
	return g
}

func (g *guardiaCicloDecisionCoberturaO404E) entrar() bool {
	if g == nil || !g.activa.Load() {
		return false
	}
	if !g.enCurso.CompareAndSwap(false, true) {
		g.violacion.Store(true)
		return false
	}
	if !g.activa.Load() {
		g.enCurso.Store(false)
		g.violacion.Store(true)
		return false
	}
	return true
}

func (g *guardiaCicloDecisionCoberturaO404E) salir() {
	if g == nil {
		return
	}
	if !g.activa.Load() {
		g.violacion.Store(true)
	}
	g.enCurso.Store(false)
}

func (g *guardiaCicloDecisionCoberturaO404E) cerrar() bool {
	if g == nil {
		return true
	}
	g.activa.Store(false)
	if g.enCurso.Load() {
		g.violacion.Store(true)
	}
	return g.violacion.Load()
}

func (s *sesionDecisionCoberturaO404E) entrar() bool {
	if s == nil || !s.guardia.entrar() {
		return false
	}
	if !s.mu.TryLock() {
		s.guardia.violacion.Store(true)
		s.guardia.salir()
		return false
	}
	return true
}

func (s *sesionDecisionCoberturaO404E) salir() {
	s.mu.Unlock()
	s.guardia.salir()
}

func contextoLigadoDecisionCoberturaO404E(
	ctxCiclo context.Context,
	ctxInvocacion context.Context,
) (context.Context, context.CancelFunc) {
	ctx, cancelar := context.WithCancel(ctxCiclo)
	detener := context.AfterFunc(ctxInvocacion, cancelar)
	return ctx, func() {
		detener()
		cancelar()
	}
}
