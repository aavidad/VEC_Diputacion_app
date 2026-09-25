package application

import (
	"context"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrServicioEventosPlazoInvalido        = errors.New("ct_evento_plazo_servicio_invalido")
	ErrSolicitudEventoPlazoInvalida        = ports.ErrSolicitudEventoPlazoInvalida
	ErrClaveEventoPlazoEnColision          = ports.ErrClaveEventoPlazoUsada
	ErrEventoPlazoEnConflicto              = ports.ErrEventoPlazoEnConflicto
	ErrEventoPlazoDenegado                 = ports.ErrOperacionEventoPlazoDenegada
	ErrEventoPlazoNoDisponible             = ports.ErrEventoPlazoNoDisponible
	ErrResultadoEventoPlazoNoConfiable     = ports.ErrResultadoEventoPlazoNoConfiable
	ErrReglasPlazoLlamamientoNoDisponibles = ports.ErrReglasPlazoNoDisponibles
)

// ServicioEventosPlazoLlamamiento registra los hechos de plazo que declara
// RRHH. Con el contacto efectivo calcula el vencimiento en el servidor con las
// reglas del catálogo; el aviso por correo nunca abre plazo. No resuelve el
// llamamiento ni propone por sí mismo la expiración: la propuesta se deriva
// del recibo y la confirma RRHH con su propia operación.
type ServicioEventosPlazoLlamamiento struct {
	reglas   ports.ReglasPlazoRespuestaLlamamiento
	registro ports.RegistroEventosPlazoLlamamiento
}

func NuevoServicioEventosPlazoLlamamiento(
	reglas ports.ReglasPlazoRespuestaLlamamiento,
	registro ports.RegistroEventosPlazoLlamamiento,
) (*ServicioEventosPlazoLlamamiento, error) {
	if dependenciaNula(reglas) || dependenciaNula(registro) {
		return nil, ErrServicioEventosPlazoInvalido
	}
	return &ServicioEventosPlazoLlamamiento{reglas: reglas, registro: registro}, nil
}

func (s *ServicioEventosPlazoLlamamiento) Registrar(
	ctx context.Context,
	solicitud ports.SolicitudRegistrarEventoPlazoLlamamiento,
) (ports.EventoPlazoLlamamientoRegistrado, error) {
	vacio := ports.EventoPlazoLlamamientoRegistrado{}
	if s == nil || ctx == nil || dependenciaNula(s.reglas) || dependenciaNula(s.registro) {
		return vacio, ErrServicioEventosPlazoInvalido
	}
	if solicitud.Validar() != nil {
		return vacio, ErrSolicitudEventoPlazoInvalida
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	var plazo *ports.PlazoRespuestaGobernado
	if solicitud.Tipo == ports.EventoPlazoContactoEfectivo {
		calculado, err := s.reglas.PlazoRespuesta(ctx, solicitud.InstanteEn)
		if errContexto := ctx.Err(); errContexto != nil {
			return vacio, errContexto
		}
		if err != nil || calculado.ValidarDesde(solicitud.InstanteEn) != nil {
			return vacio, ErrReglasPlazoLlamamientoNoDisponibles
		}
		plazo = &calculado
	}
	resultado, err := s.registro.RegistrarEventoPlazo(ctx, solicitud, plazo)
	if errContexto := ctx.Err(); errContexto != nil {
		return vacio, errContexto
	}
	if err != nil {
		if resultado.Plazo != nil || resultado.EventoRef != "" {
			return vacio, ErrResultadoEventoPlazoNoConfiable
		}
		return vacio, clasificarErrorEventoPlazo(err)
	}
	if resultado.ValidarPara(solicitud) != nil {
		return vacio, ErrResultadoEventoPlazoNoConfiable
	}
	return resultado, nil
}

// No propaga detalles internos ni datos del proveedor.
func clasificarErrorEventoPlazo(err error) error {
	for _, conocido := range []error{
		context.Canceled, context.DeadlineExceeded,
		ErrSolicitudEventoPlazoInvalida, ErrEventoPlazoDenegado, ErrClaveEventoPlazoEnColision,
		ErrEventoPlazoEnConflicto, ErrResultadoEventoPlazoNoConfiable,
	} {
		if errors.Is(err, conocido) {
			return conocido
		}
	}
	return ErrEventoPlazoNoDisponible
}
