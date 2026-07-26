package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	funcionLeerTerminalDecisionCoberturaO404E = "" +
		"vec_contratacion_temporal." +
		"leer_terminal_primario_decision_cobertura_o404e_v1"
	esquemaConsultaPrimariaDecisionCoberturaO404E = "" +
		"vec.contratacion-temporal.consulta-primaria-" +
		"decision-cobertura.o4-04e.v1"
	esquemaResultadoPrimarioDecisionCoberturaO404E = "" +
		"vec.contratacion-temporal.resultado-primario-" +
		"decision-cobertura.o4-04e.v1"
)

var _ cobertura.EjecutorLecturaPrimariaTCBOperacionDecisionCobertura = (*EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL)(nil)

type EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
	pool *pgxpool.Pool,
) (*EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL, error) {
	return nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
		pool,
	)
}

func nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
	pool iniciadorTransacciones,
) (*EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return &EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL{
		pool: pool,
	}, nil
}

func (e *EjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL) EjecutarLecturaPrimariaTCB(
	ctx context.Context,
	usar func(cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura) error,
) error {
	if ctx == nil || e == nil || dependenciaNula(e.pool) || usar == nil {
		return errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	tx, err := e.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadOnly,
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
	sesion := &sesionLecturaPrimariaDecisionCoberturaO404E{
		tx: tx, ctx: ctxCiclo, guardia: guardia,
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
		return errCallback
	}
	if violacion || !usada {
		return errSesionDecisionCoberturaO404EInvalida
	}
	return tx.Commit(ctx)
}

type sesionLecturaPrimariaDecisionCoberturaO404E struct {
	mu      sync.Mutex
	tx      pgx.Tx
	ctx     context.Context
	guardia *guardiaCicloDecisionCoberturaO404E
	usada   bool
	cerrada bool
}

var _ cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura = (*sesionLecturaPrimariaDecisionCoberturaO404E)(nil)

func (s *sesionLecturaPrimariaDecisionCoberturaO404E) cerrar() bool {
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

func (s *sesionLecturaPrimariaDecisionCoberturaO404E) LeerTerminalPrimario(
	ctx context.Context,
	consulta cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura,
) (
	cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura,
	error,
) {
	if !s.entrar() {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	defer s.salir()
	if ctx == nil || s.cerrada || s.usada || dependenciaNula(s.tx) ||
		s.ctx == nil || !s.guardia.activa.Load() {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	// Un intento de lectura consume la sesión incluso si PostgreSQL falla.
	s.usada = true
	if err := ctx.Err(); err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			err
	}
	datos, err := consulta.Datos()
	if err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	carga := consultaPrimariaDecisionCoberturaO404E{
		Esquema:           esquemaConsultaPrimariaDecisionCoberturaO404E,
		OrganizacionRef:   datos.Coordenadas.OrganizacionRef,
		ExpedienteRef:     datos.Coordenadas.ExpedienteRef,
		VersionExpediente: datos.Coordenadas.VersionExpediente,
		ReservaRef:        datos.Coordenadas.ReservaRef,
		ReciboRef:         datos.Coordenadas.ReciboRef,
		CorrelacionVECRef: datos.Coordenadas.CorrelacionVECRef,
		DecisionVECRef:    datos.Coordenadas.DecisionVECRef,
		RevisionCercado:   datos.Coordenadas.RevisionCercado,
		HuellaOrdenSHA256: datos.HuellaOrdenSHA256,
	}
	contenido, err := json.Marshal(carga)
	if err != nil || len(contenido) == 0 || len(contenido) > 64*1024 {
		borrarBytes(contenido)
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errSesionDecisionCoberturaO404EInvalida
	}
	defer borrarBytes(contenido)
	ctxOperacion, cancelarOperacion := contextoLigadoDecisionCoberturaO404E(
		s.ctx,
		ctx,
	)
	defer cancelarOperacion()
	var respuesta []byte
	err = s.tx.QueryRow(ctxOperacion, `
		SELECT resultado_json::text
		  FROM `+funcionLeerTerminalDecisionCoberturaO404E+`($1::jsonb)`,
		contenido,
	).Scan(&respuesta)
	if err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			err
	}
	defer borrarBytes(respuesta)
	return decodificarResultadoPrimarioDecisionCoberturaO404E(respuesta)
}

func (s *sesionLecturaPrimariaDecisionCoberturaO404E) entrar() bool {
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

func (s *sesionLecturaPrimariaDecisionCoberturaO404E) salir() {
	s.mu.Unlock()
	s.guardia.salir()
}

type consultaPrimariaDecisionCoberturaO404E struct {
	Esquema           string `json:"esquema"`
	OrganizacionRef   string `json:"organizacion_ref"`
	ExpedienteRef     string `json:"expediente_ref"`
	VersionExpediente uint64 `json:"version_expediente"`
	ReservaRef        string `json:"reserva_ref"`
	ReciboRef         string `json:"recibo_ref"`
	CorrelacionVECRef string `json:"correlacion_vec_ref"`
	DecisionVECRef    string `json:"decision_vec_ref"`
	RevisionCercado   uint64 `json:"revision_cercado"`
	HuellaOrdenSHA256 string `json:"huella_orden_sha256"`
}

type resultadoPrimarioDecisionCoberturaO404E struct {
	Esquema             string                                  `json:"esquema"`
	Encontrado          bool                                    `json:"encontrado"`
	Consulta            *consultaPrimariaDecisionCoberturaO404E `json:"consulta"`
	Recibo              *reciboDecisionCoberturaO404E           `json:"recibo"`
	ObservadaEnPrimario time.Time                               `json:"observada_en_primario"`
}

func decodificarResultadoPrimarioDecisionCoberturaO404E(
	contenido []byte,
) (
	cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura,
	error,
) {
	if len(contenido) == 0 ||
		len(contenido) > maximoBytesReciboDecisionCoberturaO404E {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	var dto resultadoPrimarioDecisionCoberturaO404E
	objeto, err := decodificarObjetoJSONExactoDecisionCoberturaO404E(
		contenido,
		clavesResultadoPrimarioDecisionCoberturaO404E,
		&dto,
	)
	if err != nil ||
		dto.Esquema != esquemaResultadoPrimarioDecisionCoberturaO404E ||
		!domain.InstanteUTCCanonico(dto.ObservadaEnPrimario) ||
		dto.Encontrado != (dto.Consulta != nil && dto.Recibo != nil) ||
		(!dto.Encontrado && (dto.Consulta != nil || dto.Recibo != nil)) {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	if !dto.Encontrado {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{
			ObservadaEnPrimario: dto.ObservadaEnPrimario,
		}, nil
	}
	if validarObjetoCrudoJSONExactoDecisionCoberturaO404E(
		objeto["consulta"],
		clavesConsultaPrimariaDecisionCoberturaO404E,
	) != nil ||
		validarObjetoCrudoJSONExactoDecisionCoberturaO404E(
			objeto["recibo"],
			clavesReciboDecisionCoberturaO404E,
		) != nil ||
		!validarConsultaPrimariaDecisionCoberturaO404E(*dto.Consulta) ||
		!validarReciboDecisionCoberturaO404E(*dto.Recibo) ||
		dto.ObservadaEnPrimario.Before(dto.Recibo.ConfirmadaEn) {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	recibo, err := reciboNominalDecisionCoberturaO404E(*dto.Recibo)
	if err != nil {
		return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{},
			errAdaptadorDecisionCoberturaO404ENoDisponible
	}
	return cobertura.DatosResultadoLecturaPrimariaTCBOperacionDecisionCobertura{
		Encontrado: true,
		Coordenadas: cobertura.DatosConsultaPrimariaOperacionDecisionCobertura{
			OrganizacionRef:   dto.Consulta.OrganizacionRef,
			ExpedienteRef:     dto.Consulta.ExpedienteRef,
			VersionExpediente: dto.Consulta.VersionExpediente,
			ReservaRef:        dto.Consulta.ReservaRef,
			ReciboRef:         dto.Consulta.ReciboRef,
			CorrelacionVECRef: dto.Consulta.CorrelacionVECRef,
			DecisionVECRef:    dto.Consulta.DecisionVECRef,
			RevisionCercado:   dto.Consulta.RevisionCercado,
		},
		HuellaOrdenSHA256: dto.Consulta.HuellaOrdenSHA256,
		Recibo:            recibo, ObservadaEnPrimario: dto.ObservadaEnPrimario,
	}, nil
}

func reciboNominalDecisionCoberturaO404E(
	dto reciboDecisionCoberturaO404E,
) (cobertura.ReciboOperacionDecisionCobertura, error) {
	contenido, err := json.Marshal(dto)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	defer borrarBytes(contenido)
	crudo, err := decodificarReciboDecisionCoberturaO404E(contenido)
	if err != nil {
		return cobertura.ReciboOperacionDecisionCobertura{}, err
	}
	recibo := cobertura.ReciboOperacionDecisionCobertura{
		ReciboRef: crudo.ReciboRef, ReservaRef: crudo.ReservaRef,
		AuditoriaRef:            crudo.AuditoriaRef,
		CorrelacionVECRef:       crudo.CorrelacionVECRef,
		DecisionVECRef:          crudo.DecisionVECRef,
		DecisionVECHuellaSHA256: crudo.DecisionVECHuellaSHA256,
		CodigoProbatorioVEC:     crudo.CodigoProbatorioVEC,
		ConcedidaVEC:            crudo.ConcedidaVEC,
		RevisionCercado:         crudo.RevisionCercado,
		AmbitoIdempotenciaHMAC:  crudo.AmbitoIdempotenciaHMAC,
		HuellaSemanticaHMAC:     crudo.HuellaSemanticaHMAC,
		ConfirmadaEn:            crudo.ConfirmadaEn,
	}
	if crudo.Aplicada {
		recibo.Aplicada =
			&cobertura.ResultadoAplicadoOperacionDecisionCobertura{
				DecisionCoberturaRef:    crudo.DecisionCoberturaRef,
				DecisionCoberturaHuella: crudo.DecisionCoberturaHuella,
				VersionResultante:       crudo.VersionResultante,
				EventoRef:               crudo.EventoRef, ActuacionRef: crudo.ActuacionRef,
			}
	} else {
		recibo.DenegadaVEC =
			&cobertura.ResultadoDenegadoVECOperacionDecisionCobertura{}
	}
	return recibo, nil
}
