package postgres

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

const (
	esquemaConsultaRecuperacionResultadoCoberturaO405 = "" +
		"vec.contratacion-temporal.consulta-recuperacion-propia-" +
		"decision-cobertura.o4-05.v1"
	maximoBytesConsultaRecuperacionResultadoCoberturaO405 = 8 * 1024
)

var _ cobertura.EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL)(nil)

// EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL
// aporta la transacción física de recuperación O4-05. El pool debe apuntar al
// primario con el rol nominativo de lectura; la función SQL vuelve a comprobar
// ambas condiciones y falla cerrado.
type EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL struct {
	pool iniciadorTransacciones
}

func nuevoEjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL(
	pool iniciadorTransacciones,
) (*EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil,
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	return &EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL{
		pool: pool,
	}, nil
}

func (e *EjecutorLecturaResultadoHistoricoOperacionDecisionCoberturaPostgreSQL) EjecutarLecturaResultadoHistoricoTCB(
	ctx context.Context,
	usar func(
		cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	) error,
) (errResultado error) {
	defer func() {
		if recover() != nil {
			errResultado =
				cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
		}
	}()
	if ctx == nil || e == nil || dependenciaNula(e.pool) || usar == nil {
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return normalizarErrorRecuperacionResultadoCoberturaO405(ctx, err)
	}
	defer revertirTransaccion(tx)
	oidFuncion := uint32(0)
	portadorOID, ok := tx.(interface {
		oidFuncionRecuperacionCoberturaO405() uint32
		tlsEsperadoRecuperacionCoberturaO405() bool
	})
	tlsEsperado := false
	if ok {
		oidFuncion = portadorOID.oidFuncionRecuperacionCoberturaO405()
		tlsEsperado =
			portadorOID.tlsEsperadoRecuperacionCoberturaO405()
	}
	if err := configurarSesionDecisionCoberturaO404E(ctx, tx); err != nil {
		return normalizarErrorRecuperacionResultadoCoberturaO405(ctx, err)
	}

	ctxCiclo, cancelarCiclo := context.WithCancel(ctx)
	guardia := nuevaGuardiaCicloDecisionCoberturaO404E()
	sesion := &sesionRecuperacionResultadoCoberturaO405{
		tx: tx, ctx: ctxCiclo, guardia: guardia,
		oidFuncion: oidFuncion, tlsEsperado: tlsEsperado,
	}
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
	usada := sesion.cerrar()
	cerrada = true
	if errCallback != nil {
		return normalizarErrorCallbackRecuperacionResultadoCoberturaO405(
			ctx,
			errCallback,
		)
	}
	if violacion || !usada {
		return cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return normalizarErrorRecuperacionResultadoCoberturaO405(ctx, err)
	}
	return nil
}

type sesionRecuperacionResultadoCoberturaO405 struct {
	mu          sync.Mutex
	tx          pgx.Tx
	ctx         context.Context
	guardia     *guardiaCicloDecisionCoberturaO404E
	oidFuncion  uint32
	tlsEsperado bool
	usada       bool
	cerrada     bool
}

var _ cobertura.SesionLecturaResultadoHistoricoOperacionDecisionCoberturaTCB = (*sesionRecuperacionResultadoCoberturaO405)(nil)

func (s *sesionRecuperacionResultadoCoberturaO405) cerrar() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	usada := s.usada && !s.cerrada
	s.cerrada = true
	s.tx = nil
	s.ctx = nil
	return usada
}

func (s *sesionRecuperacionResultadoCoberturaO405) LeerResultadoHistoricoTCB(
	ctx context.Context,
	consulta cobertura.ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	if !s.entrar() {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	defer s.salir()
	if ctx == nil || s.cerrada || s.usada || dependenciaNula(s.tx) ||
		s.ctx == nil || !s.guardia.activa.Load() {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	// El primer intento consume la sesión aunque falle antes de llegar al SQL.
	s.usada = true
	if err := ctx.Err(); err != nil {
		return resultadoVacioRecuperacionResultadoCoberturaO405(), err
	}
	carga, err := construirConsultaRecuperacionResultadoCoberturaO405(consulta)
	if err != nil {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	return s.consultarResultadoHistoricoO405(ctx, carga)
}

func (s *sesionRecuperacionResultadoCoberturaO405) consultarResultadoHistoricoO405(
	ctx context.Context,
	carga consultaRecuperacionResultadoCoberturaO405,
) (
	cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
	error,
) {
	contenido, err := json.Marshal(carga)
	if err != nil || len(contenido) == 0 ||
		len(contenido) > maximoBytesConsultaRecuperacionResultadoCoberturaO405 ||
		s.oidFuncion == 0 {
		borrarBytes(contenido)
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			cobertura.ErrLecturaResultadoHistoricoOperacionDecisionCoberturaNoConfiable
	}
	defer borrarBytes(contenido)

	ctxOperacion, cancelarOperacion := contextoLigadoDecisionCoberturaO404E(
		s.ctx,
		ctx,
	)
	defer cancelarOperacion()
	filas, err := s.tx.Query(
		ctxOperacion,
		consultaSQLRecuperacionResultadoCoberturaO405,
		firmaFuncionRecuperacionResultadoCoberturaO405,
		rolLectorResultadoCoberturaO405,
		esquemaFuncionRecuperacionResultadoCoberturaO405,
		nombreFuncionRecuperacionResultadoCoberturaO405,
		propietarioFuncionRecuperacionResultadoCoberturaO405,
		configuracionFuncionRecuperacionResultadoCoberturaO405(),
		argumentosFuncionRecuperacionResultadoCoberturaO405,
		retornoFuncionRecuperacionResultadoCoberturaO405,
		lenguajeFuncionRecuperacionResultadoCoberturaO405,
		huellaProsrcFuncionRecuperacionResultadoCoberturaO405,
		huellaDefinicionFuncionRecuperacionResultadoCoberturaO405,
		s.oidFuncion,
		s.tlsEsperado,
		contenido,
		int64(maximoBytesResultadoRecuperacionResultadoCoberturaO405),
	)
	if err != nil {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			normalizarErrorRecuperacionResultadoCoberturaO405(
				ctxOperacion,
				err,
			)
	}
	defer filas.Close()
	if !filas.Next() {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			normalizarErrorFilasRecuperacionResultadoCoberturaO405(
				ctxOperacion,
				filas.Err(),
			)
	}
	var respuesta []byte
	var longitud int64
	var manifiestoAcreditado bool
	if err := filas.Scan(
		&respuesta,
		&longitud,
		&manifiestoAcreditado,
	); err != nil {
		borrarBytes(respuesta)
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			normalizarErrorFilasRecuperacionResultadoCoberturaO405(
				ctxOperacion,
				err,
			)
	}
	defer borrarBytes(respuesta)
	if filas.Next() {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	if err := filas.Err(); err != nil {
		return resultadoVacioRecuperacionResultadoCoberturaO405(),
			normalizarErrorFilasRecuperacionResultadoCoberturaO405(
				ctxOperacion,
				err,
			)
	}
	if !manifiestoAcreditado || longitud <= 0 ||
		longitud > maximoBytesResultadoRecuperacionResultadoCoberturaO405 ||
		longitud != int64(len(respuesta)) {
		return resultadoNoConfiableRecuperacionResultadoCoberturaO405()
	}
	return decodificarResultadoRecuperacionResultadoCoberturaO405(respuesta)
}

func (s *sesionRecuperacionResultadoCoberturaO405) entrar() bool {
	if s == nil || s.guardia == nil || !s.guardia.entrar() {
		return false
	}
	if !s.mu.TryLock() {
		s.guardia.violacion.Store(true)
		s.guardia.salir()
		return false
	}
	return true
}

func (s *sesionRecuperacionResultadoCoberturaO405) salir() {
	s.mu.Unlock()
	s.guardia.salir()
}

type consultaRecuperacionResultadoCoberturaO405 struct {
	Esquema         string   `json:"esquema"`
	OrganizacionRef string   `json:"organizacion_ref"`
	ExpedienteRef   string   `json:"expediente_ref"`
	AmbitosHMAC     []string `json:"ambitos_idempotencia_hmac"`
}

func construirConsultaRecuperacionResultadoCoberturaO405(
	consulta cobertura.ConsultaLecturaResultadoHistoricoOperacionDecisionCoberturaTCB,
) (consultaRecuperacionResultadoCoberturaO405, error) {
	datos, err := consulta.DatosLectura()
	if err != nil {
		return consultaRecuperacionResultadoCoberturaO405{}, err
	}
	coleccion, err := datos.AmbitosHMAC.Datos()
	if err != nil {
		return consultaRecuperacionResultadoCoberturaO405{}, err
	}
	ambitos := make([]string, 0, len(coleccion.Retenidos)+1)
	ambitos = append(ambitos, coleccion.Activo.Valor)
	for _, retenido := range coleccion.Retenidos {
		ambitos = append(ambitos, retenido.Valor)
	}
	return consultaRecuperacionResultadoCoberturaO405{
		Esquema:         esquemaConsultaRecuperacionResultadoCoberturaO405,
		OrganizacionRef: datos.OrganizacionRef,
		ExpedienteRef:   datos.ExpedienteRef,
		AmbitosHMAC:     ambitos,
	}, nil
}

func resultadoVacioRecuperacionResultadoCoberturaO405() cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB {
	return cobertura.DatosResultadoLecturaResultadoHistoricoOperacionDecisionCoberturaTCB{}
}
