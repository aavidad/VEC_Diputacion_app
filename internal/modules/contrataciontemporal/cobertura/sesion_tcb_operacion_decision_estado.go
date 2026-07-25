package cobertura

import (
	"context"
	"strconv"
	"sync/atomic"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type estadoDespliegueSesionTCB uint8

const (
	estadoSesionTCBNueva estadoDespliegueSesionTCB = iota
	estadoSesionTCBAbierta
	estadoSesionTCBGobernada
	estadoSesionTCBDecisionVEC
	estadoSesionTCBConsumosC1
	estadoSesionTCBTerminal
	estadoSesionTCBConsumida
	estadoSesionTCBInvalida
)

type sesionControladaOperacionDecisionCobertura struct {
	destino    SesionTCBOperacionDecisionCobertura
	guardia    *guardiaCicloSesionTCBOperacionDecisionCobertura
	estado     estadoDespliegueSesionTCB
	cabecera   DatosCabeceraSesionTCBOperacionDecisionCobertura
	gobierno   DatosGobiernoSesionTCBOperacionDecisionCobertura
	consumos   uint64
	peticiones map[string]struct{}
	respuestas map[string]struct{}
}

// guardiaCicloSesionTCBOperacionDecisionCobertura invalida el ciclo al
// retornar el ejecutor. No espera a un adaptador infractor que ignore la
// cancelación; rechaza operaciones nuevas y descarta la salida de cualquiera
// que terminase después.
type guardiaCicloSesionTCBOperacionDecisionCobertura struct {
	retornada atomic.Bool
}

func (g *guardiaCicloSesionTCBOperacionDecisionCobertura) marcarRetornoEjecutor() {
	g.retornada.Store(true)
}

func (g *guardiaCicloSesionTCBOperacionDecisionCobertura) delegar(
	operacion func() error,
) error {
	if g.retornada.Load() {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	err := operacion()
	if g.retornada.Load() {
		return ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return err
}

func (g *guardiaCicloSesionTCBOperacionDecisionCobertura) delegarConfirmacion(
	operacion func() (
		DatosReciboSesionTCBOperacionDecisionCobertura,
		error,
	),
) (DatosReciboSesionTCBOperacionDecisionCobertura, error) {
	if g.retornada.Load() {
		return DatosReciboSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	recibo, err := operacion()
	if g.retornada.Load() {
		return DatosReciboSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return recibo, err
}

func nuevaSesionControladaOperacionDecisionCobertura(
	destino SesionTCBOperacionDecisionCobertura,
	guardia *guardiaCicloSesionTCBOperacionDecisionCobertura,
) *sesionControladaOperacionDecisionCobertura {
	return &sesionControladaOperacionDecisionCobertura{
		destino: destino,
		guardia: guardia,
		estado:  estadoSesionTCBNueva,
	}
}

func (s *sesionControladaOperacionDecisionCobertura) invalidar() error {
	s.estado = estadoSesionTCBInvalida
	return ErrSesionTCBOperacionDecisionCoberturaInvalida
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarApertura(
	cabecera CabeceraSesionTCBOperacionDecisionCobertura,
) error {
	datos, err := cabecera.Datos()
	if s == nil || s.estado != estadoSesionTCBNueva ||
		dependenciaGobiernoOperacionCoberturaNula(s.destino) || err != nil {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	s.cabecera = datos
	s.peticiones = make(map[string]struct{}, datos.NumeroConsumosC1)
	s.respuestas = make(map[string]struct{}, datos.NumeroConsumosC1)
	s.estado = estadoSesionTCBAbierta
	if s.guardia.delegar(func() error {
		return s.destino.Abrir(cabecera)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarGobierno(
	gobierno GobiernoSesionTCBOperacionDecisionCobertura,
) error {
	datos, err := gobierno.Datos()
	if s == nil || s.estado != estadoSesionTCBAbierta ||
		s.cabecera.Rama != RamaSesionTCBOperacionDecisionCoberturaConcedida ||
		err != nil ||
		datos.Politica.OrganizacionRef != s.cabecera.OrganizacionRef ||
		datos.PoliticaActuacion.OrganizacionRef !=
			s.cabecera.OrganizacionRef {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	s.gobierno = datos
	s.estado = estadoSesionTCBGobernada
	if s.guardia.delegar(func() error {
		return s.destino.Gobierno(gobierno)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarDecisionVEC(
	decision DecisionVECSesionTCBOperacionDecisionCobertura,
) error {
	datos, err := decision.Datos()
	if s == nil || err != nil {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	estadoEsperado := estadoSesionTCBAbierta
	concedidaEsperada := false
	if s.cabecera.Rama == RamaSesionTCBOperacionDecisionCoberturaConcedida {
		estadoEsperado = estadoSesionTCBGobernada
		concedidaEsperada = true
	}
	if s.estado != estadoEsperado ||
		datos.Concedida != concedidaEsperada ||
		datos.Resumen.DecisionRef != s.cabecera.DecisionVECRef {
		return s.invalidar()
	}
	solicitud, errSolicitud := datos.Orden.Solicitud.Datos()
	correlacion, errCorrelacion :=
		solicitud.Correlacion.ValorCanonico()
	if errSolicitud != nil || errCorrelacion != nil ||
		correlacion != s.cabecera.CorrelacionVECRef ||
		solicitud.Recurso.Referencia != s.cabecera.ReservaRef ||
		solicitud.Recurso.ModuloID !=
			moduloRecursoOperacionDecisionCobertura ||
		solicitud.Recurso.Tipo != tipoRecursoOperacionDecisionCobertura {
		return s.invalidar()
	}
	s.estado = estadoSesionTCBDecisionVEC
	if s.guardia.delegar(func() error {
		return s.destino.DecisionVEC(decision)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarConsumoC1(
	consumo ConsumoC1SesionTCBOperacionDecisionCobertura,
) error {
	datos, err := consumo.Datos()
	if s == nil || err != nil ||
		s.cabecera.Rama != RamaSesionTCBOperacionDecisionCoberturaConcedida ||
		(s.estado != estadoSesionTCBDecisionVEC &&
			s.estado != estadoSesionTCBConsumosC1) ||
		datos.Posicion != s.consumos+1 ||
		datos.Total != s.cabecera.NumeroConsumosC1 ||
		datos.Resumen.OrganizacionRef != s.cabecera.OrganizacionRef ||
		datos.Resumen.ExpedienteRef != s.cabecera.ExpedienteRef ||
		datos.Resumen.VersionExpediente != s.cabecera.VersionExpediente ||
		!datos.Resumen.Catalogo.CoincideExactamente(
			domain.IdentidadCatalogoViasCobertura{
				Referencia:   s.gobierno.Catalogo.Referencia,
				Version:      s.gobierno.Catalogo.Version,
				HuellaSHA256: s.gobierno.Catalogo.HuellaSHA256,
			},
		) {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	claveRespuesta := datos.Resumen.AutoridadRef + "\x00" +
		strconv.FormatUint(uint64(datos.Resumen.Generacion), 10) + "\x00" +
		datos.Resumen.ReciboRespuestaRef
	if !s.registrarIdentidadConsumoC1(
		datos.Posicion,
		datos.Total,
		datos.Resumen.PeticionRef,
		claveRespuesta,
	) {
		return s.invalidar()
	}
	s.estado = estadoSesionTCBConsumosC1
	if s.guardia.delegar(func() error {
		return s.destino.ConsumoC1(consumo)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) registrarIdentidadConsumoC1(
	posicion uint64,
	total uint64,
	peticion string,
	respuesta string,
) bool {
	if s == nil || posicion == 0 || posicion != s.consumos+1 ||
		total == 0 ||
		total > MaximoConsumosC1SesionTCBOperacionDecisionCobertura ||
		total != s.cabecera.NumeroConsumosC1 ||
		posicion > total ||
		peticion == "" ||
		respuesta == "" {
		return false
	}
	if _, repetida := s.peticiones[peticion]; repetida {
		return false
	}
	if _, repetida := s.respuestas[respuesta]; repetida {
		return false
	}
	s.peticiones[peticion] = struct{}{}
	s.respuestas[respuesta] = struct{}{}
	s.consumos++
	return true
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarConcesion(
	efecto EfectoConcedidoSesionTCBOperacionDecisionCobertura,
) error {
	datos, err := efecto.Datos()
	if s == nil || err != nil ||
		s.cabecera.Rama != RamaSesionTCBOperacionDecisionCoberturaConcedida ||
		s.estado != estadoSesionTCBConsumosC1 ||
		s.consumos == 0 ||
		s.consumos != s.cabecera.NumeroConsumosC1 ||
		datos.AgregadoAnterior.OrganizacionRef !=
			s.cabecera.OrganizacionRef ||
		datos.AgregadoAnterior.Referencia != s.cabecera.ExpedienteRef ||
		datos.AgregadoAnterior.Version != s.cabecera.VersionExpediente ||
		datos.Propuesta.AnalisisRef != s.cabecera.AnalisisRef ||
		datos.Propuesta.AnalisisHuellaSHA256 !=
			s.cabecera.AnalisisHuellaSHA256 ||
		datos.Propuesta.PreparacionEvidenciasRef !=
			s.cabecera.PreparacionC1Ref ||
		datos.Propuesta.PreparacionEvidenciasHuellaSHA256 !=
			s.cabecera.PreparacionC1HuellaSHA256 ||
		!datos.Propuesta.Catalogo.CoincideExactamente(
			domain.IdentidadCatalogoViasCobertura{
				Referencia:   s.gobierno.Catalogo.Referencia,
				Version:      s.gobierno.Catalogo.Version,
				HuellaSHA256: s.gobierno.Catalogo.HuellaSHA256,
			},
		) ||
		datos.Propuesta.Politica.Referencia !=
			s.gobierno.Politica.Referencia ||
		datos.Propuesta.Politica.Version != s.gobierno.Politica.Version ||
		datos.Propuesta.Politica.HuellaSHA256 !=
			s.gobierno.Politica.HuellaSHA256 ||
		datos.EfectoEn.Before(s.cabecera.ObservadaEnDB) ||
		!datos.EfectoEn.Before(s.cabecera.ValidaHastaOrden) ||
		!datos.ValidaHasta.Equal(s.cabecera.ValidaHastaOrden) {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	s.estado = estadoSesionTCBTerminal
	if s.guardia.delegar(func() error {
		return s.destino.Concesion(efecto)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarDenegacion(
	terminal TerminalDenegadoSesionTCBOperacionDecisionCobertura,
) error {
	datos, err := terminal.Datos()
	if s == nil || err != nil ||
		s.cabecera.Rama != RamaSesionTCBOperacionDecisionCoberturaDenegada ||
		s.estado != estadoSesionTCBDecisionVEC ||
		s.consumos != 0 ||
		datos.OrganizacionRef != s.cabecera.OrganizacionRef ||
		datos.ExpedienteRef != s.cabecera.ExpedienteRef ||
		datos.VersionExpediente != s.cabecera.VersionExpediente ||
		datos.ReservaRef != s.cabecera.ReservaRef ||
		datos.ReciboRef != s.cabecera.ReciboRef ||
		datos.AuditoriaRef != s.cabecera.AuditoriaRef ||
		datos.CorrelacionVECRef != s.cabecera.CorrelacionVECRef ||
		datos.DecisionVECRef != s.cabecera.DecisionVECRef ||
		datos.RevisionCercado != s.cabecera.RevisionCercado ||
		!datos.ValidaHasta.Equal(s.cabecera.ValidaHastaOrden) {
		if s == nil {
			return ErrSesionTCBOperacionDecisionCoberturaInvalida
		}
		return s.invalidar()
	}
	s.estado = estadoSesionTCBTerminal
	if s.guardia.delegar(func() error {
		return s.destino.Denegacion(terminal)
	}) != nil {
		return s.invalidar()
	}
	return nil
}

func (s *sesionControladaOperacionDecisionCobertura) aplicarConfirmacion(
	ctx context.Context,
) (DatosReciboSesionTCBOperacionDecisionCobertura, error) {
	if s == nil || s.estado != estadoSesionTCBTerminal ||
		dependenciaGobiernoOperacionCoberturaNula(ctx) {
		if s != nil {
			s.invalidar()
		}
		return DatosReciboSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	if err := ctx.Err(); err != nil {
		s.invalidar()
		return DatosReciboSesionTCBOperacionDecisionCobertura{}, err
	}
	// Se consume antes de delegar. Ni un error del adaptador permite una
	// segunda llamada a Confirmar sobre la misma sesión.
	s.estado = estadoSesionTCBConsumida
	recibo, err := s.guardia.delegarConfirmacion(func() (
		DatosReciboSesionTCBOperacionDecisionCobertura,
		error,
	) {
		return s.destino.Confirmar(ctx)
	})
	if err != nil {
		s.estado = estadoSesionTCBInvalida
		return DatosReciboSesionTCBOperacionDecisionCobertura{},
			ErrSesionTCBOperacionDecisionCoberturaInvalida
	}
	return recibo, nil
}
