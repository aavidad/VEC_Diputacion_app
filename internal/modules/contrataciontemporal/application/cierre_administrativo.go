package application

import (
	"context"
	"errors"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const tiempoMaximoCierreAdministrativo = 15 * time.Second

var (
	ErrServicioCierreAdministrativoInvalido = errors.New(
		"contratacion temporal: servicio de cierre administrativo invalido",
	)
	ErrSolicitudCierreAdministrativoInvalida = errors.New(
		"contratacion temporal: solicitud de cierre administrativo invalida",
	)
	ErrCierreAdministrativoNoPermitido = errors.New(
		"contratacion temporal: cierre administrativo no permitido",
	)
	ErrVersionCierreAdministrativoEnConflicto = errors.New(
		"contratacion temporal: version de cierre administrativo en conflicto",
	)
	ErrCierreAdministrativoNoDisponible = errors.New(
		"contratacion temporal: cierre administrativo no disponible",
	)
	ErrResultadoCierreAdministrativoInvalido = errors.New(
		"contratacion temporal: resultado de cierre administrativo invalido",
	)
	errDecisionCierreAdministrativoNoPermitida = errors.New(
		"decision de cierre administrativo no permitida",
	)
)

type SolicitudCerrarAdministrativamente struct {
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	VersionEsperada uint64
	TransicionClave domain.ClaveCatalogo
	MotivoClave     domain.ClaveCatalogo
}

type SolicitudReabrirExcepcionalmente struct {
	OrganizacionRef string
	ExpedienteRef   string
	SeguimientoRef  string
	VersionEsperada uint64
	TransicionClave domain.ClaveCatalogo
	MotivoClave     domain.ClaveCatalogo
}

type ServicioCierreAdministrativo struct {
	transaccion ports.TransaccionCierreAdministrativo
}

func NuevoServicioCierreAdministrativo(
	transaccion ports.TransaccionCierreAdministrativo,
) (*ServicioCierreAdministrativo, error) {
	if dependenciaNula(transaccion) {
		return nil, ErrServicioCierreAdministrativoInvalido
	}
	return &ServicioCierreAdministrativo{transaccion: transaccion}, nil
}

func (s *ServicioCierreAdministrativo) Cerrar(
	ctx context.Context,
	solicitud SolicitudCerrarAdministrativamente,
) (ports.ResultadoCierreAdministrativo, error) {
	return s.ejecutar(ctx, ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionCerrarAdministrativamente,
		OrganizacionRef: solicitud.OrganizacionRef,
		ExpedienteRef:   solicitud.ExpedienteRef,
		SeguimientoRef:  solicitud.SeguimientoRef,
		VersionEsperada: solicitud.VersionEsperada,
		TransicionClave: solicitud.TransicionClave,
		MotivoClave:     solicitud.MotivoClave,
	})
}

func (s *ServicioCierreAdministrativo) ReabrirExcepcionalmente(
	ctx context.Context,
	solicitud SolicitudReabrirExcepcionalmente,
) (ports.ResultadoCierreAdministrativo, error) {
	return s.ejecutar(ctx, ports.SolicitudTransaccionCierreAdministrativo{
		Operacion:       ports.OperacionReabrirExcepcionalmente,
		OrganizacionRef: solicitud.OrganizacionRef,
		ExpedienteRef:   solicitud.ExpedienteRef,
		SeguimientoRef:  solicitud.SeguimientoRef,
		VersionEsperada: solicitud.VersionEsperada,
		TransicionClave: solicitud.TransicionClave,
		MotivoClave:     solicitud.MotivoClave,
	})
}

func (s *ServicioCierreAdministrativo) ejecutar(
	ctx context.Context,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) (ports.ResultadoCierreAdministrativo, error) {
	if s == nil || ctx == nil || dependenciaNula(s.transaccion) {
		return ports.ResultadoCierreAdministrativo{},
			ErrServicioCierreAdministrativoInvalido
	}
	if solicitud.Validar() != nil {
		return ports.ResultadoCierreAdministrativo{},
			ErrSolicitudCierreAdministrativoInvalida
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoCierreAdministrativo{}, err
	}
	ctxOperacion, cancelar := context.WithTimeout(
		ctx,
		tiempoMaximoCierreAdministrativo,
	)
	defer cancelar()

	var decisionInvocada bool
	var actuacionRef, reciboRef string
	resultado, err := s.transaccion.EjecutarCierreAdministrativo(
		ctxOperacion,
		solicitud,
		func(
			preparacion ports.PreparacionTransaccionCierreAdministrativo,
		) (domain.Seguimiento, error) {
			if decisionInvocada {
				return domain.Seguimiento{},
					errDecisionCierreAdministrativoNoPermitida
			}
			decisionInvocada = true
			if preparacion.ValidarPara(solicitud) != nil ||
				!transicionCierreAdministrativoValida(
					preparacion.Definicion,
					preparacion.Seguimiento.EstadoActual(),
					solicitud,
				) ||
				solicitud.Operacion == ports.OperacionCerrarAdministrativamente &&
					preparacion.Inventario.Pendientes != 0 {
				return domain.Seguimiento{},
					errDecisionCierreAdministrativoNoPermitida
			}
			siguiente, errAplicar := preparacion.Seguimiento.Aplicar(
				preparacion.Definicion,
				solicitud.VersionEsperada,
				domain.DatosTransicionSeguimiento{
					ActuacionRef:    preparacion.ActuacionRef,
					TransicionClave: solicitud.TransicionClave,
					MotivoClave:     solicitud.MotivoClave,
					ActorRef:        preparacion.ActorRef,
					UnidadRef:       preparacion.UnidadRef,
					EfectivoEn:      preparacion.EfectivoEn,
					RegistradaEn:    preparacion.RegistradaEn,
					Documentos:      preparacion.Documentos,
					ReciboRef:       preparacion.ReciboRef,
					CorrelacionRef:  preparacion.CorrelacionRef,
				},
			)
			if errAplicar != nil {
				return domain.Seguimiento{}, errAplicar
			}
			if !actuacionCierreAdministrativoValida(
				siguiente,
				solicitud,
				preparacion,
			) {
				return domain.Seguimiento{},
					errDecisionCierreAdministrativoNoPermitida
			}
			actuacionRef = preparacion.ActuacionRef
			reciboRef = preparacion.ReciboRef
			return siguiente, nil
		},
	)
	if err != nil {
		return ports.ResultadoCierreAdministrativo{},
			clasificarErrorCierreAdministrativo(ctxOperacion, err)
	}
	if !decisionInvocada || resultado.ValidarPara(
		solicitud,
		actuacionRef,
		reciboRef,
	) != nil {
		return ports.ResultadoCierreAdministrativo{},
			ErrResultadoCierreAdministrativoInvalido
	}
	return resultado, nil
}

func transicionCierreAdministrativoValida(
	definicion domain.DefinicionSeguimiento,
	estadoActual domain.ClaveCatalogo,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
) bool {
	publicacion := definicion.Publicacion()
	estados := make(map[domain.ClaveCatalogo]bool, len(publicacion.Estados))
	for _, estado := range publicacion.Estados {
		estados[estado.Clave] = estado.Final
	}
	for _, transicion := range publicacion.Transiciones {
		if transicion.Clave != solicitud.TransicionClave {
			continue
		}
		origenFinal, existeOrigen := estados[transicion.Origen]
		destinoFinal, existeDestino := estados[transicion.Destino]
		if !existeOrigen || !existeDestino ||
			transicion.Origen != estadoActual ||
			!transicion.MotivoObligatorio ||
			!contieneMotivoCierreAdministrativo(
				transicion.MotivosPermitidos,
				solicitud.MotivoClave,
			) {
			return false
		}
		switch solicitud.Operacion {
		case ports.OperacionCerrarAdministrativamente:
			return transicion.Clase == domain.TransicionOrdinaria &&
				!origenFinal && destinoFinal
		case ports.OperacionReabrirExcepcionalmente:
			return transicion.Clase == domain.TransicionReapertura &&
				origenFinal && !destinoFinal
		default:
			return false
		}
	}
	return false
}

func contieneMotivoCierreAdministrativo(
	permitidos []domain.ClaveCatalogo,
	motivo domain.ClaveCatalogo,
) bool {
	for _, permitido := range permitidos {
		if permitido == motivo {
			return true
		}
	}
	return false
}

func actuacionCierreAdministrativoValida(
	seguimiento domain.Seguimiento,
	solicitud ports.SolicitudTransaccionCierreAdministrativo,
	preparacion ports.PreparacionTransaccionCierreAdministrativo,
) bool {
	if seguimiento.Version() != solicitud.VersionEsperada+1 {
		return false
	}
	actuaciones := seguimiento.Actuaciones()
	if len(actuaciones) == 0 {
		return false
	}
	ultima := actuaciones[len(actuaciones)-1]
	return ultima.VersionSeguimiento == seguimiento.Version() &&
		ultima.ActuacionRef == preparacion.ActuacionRef &&
		ultima.TransicionClave == solicitud.TransicionClave &&
		ultima.MotivoClave == solicitud.MotivoClave &&
		ultima.ActorRef == preparacion.ActorRef &&
		ultima.UnidadRef == preparacion.UnidadRef &&
		ultima.ReciboRef == preparacion.ReciboRef &&
		ultima.CorrelacionRef == preparacion.CorrelacionRef
}

func clasificarErrorCierreAdministrativo(
	ctx context.Context,
	err error,
) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(err, domain.ErrVersionEnConflicto):
		return ErrVersionCierreAdministrativoEnConflicto
	case errors.Is(err, errDecisionCierreAdministrativoNoPermitida),
		errors.Is(err, ports.ErrCierreAdministrativoDenegado),
		errors.Is(err, domain.ErrTransicionInvalida),
		errors.Is(err, domain.ErrSeguimientoInvalido),
		errors.Is(err, domain.ErrActuacionSeguimientoEnConflicto):
		return ErrCierreAdministrativoNoPermitido
	default:
		return ErrCierreAdministrativoNoDisponible
	}
}
