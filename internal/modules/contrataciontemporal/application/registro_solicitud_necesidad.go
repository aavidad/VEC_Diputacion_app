package application

import (
	"context"
	"errors"
	"reflect"
	"strconv"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// La entrada v2 fija solo la cantidad declarada en la necesidad sellada.
// Jornada y denominación no se deducen del código RPT ni se inventan.
func completarCantidadAltaNecesidadV2(solicitud *domain.SolicitudCentro, necesidad *domain.DatosNecesidadAlta) error {
	if solicitud == nil || necesidad == nil || necesidad.ValidarInstantanea() != nil {
		return ErrSolicitudRegistroInvalida
	}
	texto := necesidad.Campos["numero_personas"]
	cantidad, err := strconv.ParseUint(texto, 10, 32)
	if err != nil || cantidad == 0 || strconv.FormatUint(cantidad, 10) != texto ||
		solicitud.NumeroPersonas != 0 || solicitud.JornadaMinutos != 0 || solicitud.PuestoSolicitado != "" {
		return ErrSolicitudRegistroInvalida
	}
	solicitud.NumeroPersonas = uint32(cantidad)
	return nil
}

type claveNecesidadAltaValidada struct{}

type necesidadAltaValidada struct {
	organizacionRef string
	material        domain.DatosNecesidadAlta
}

type claveNecesidadPeticionRatificada struct{}

type necesidadPeticionRatificada struct {
	organizacionRef, actorRef, perfilRef, claveAlta, ambitoAltaHMAC string
	peticionRef                                                     string
	necesidad                                                       domain.DatosNecesidadAlta
}

// ConNecesidadRatificadaPeticionCentro sólo acepta la reserva completa que ya
// leyó la entrega autorizada. El marcador no sustituye la autorización del
// alta: liga la instantánea original a la misma clave, actor y ámbito HMAC.
func ConNecesidadRatificadaPeticionCentro(ctx context.Context, organizacionRef string, e ports.EntregaPeticionCentro) (context.Context, error) {
	if ctx == nil || ctx.Err() != nil || !domain.ReferenciaOpacaValida(organizacionRef) ||
		e.ValidarReserva() != nil || e.EstadoEntrega != "preparada" ||
		e.Peticion.Solicitud.Necesidad == nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	necesidad, err := e.Peticion.Solicitud.Necesidad.Clonar()
	if err != nil || necesidad.CausaClave != e.Peticion.Solicitud.MotivoClave ||
		necesidad.Periodo != e.Peticion.Solicitud.Periodo {
		return nil, ErrSolicitudRegistroInvalida
	}
	return context.WithValue(ctx, claveNecesidadPeticionRatificada{}, necesidadPeticionRatificada{
		organizacionRef: organizacionRef,
		actorRef:        e.ActorRef, perfilRef: e.PerfilRef, claveAlta: e.ClaveAlta,
		ambitoAltaHMAC: e.AmbitoAltaHMAC, peticionRef: e.Peticion.Referencia, necesidad: necesidad,
	}), nil
}

// NecesidadAltaValidadaPara permite a la composición distinguir una necesidad
// comprobada por el mismo servicio de una modalidad histórica. La clave del
// contexto es privada y ningún campo HTTP puede crear esta marca.
func NecesidadAltaValidadaPara(ctx context.Context, organizacionRef string, necesidad *domain.DatosNecesidadAlta) bool {
	if ctx == nil || necesidad == nil || necesidad.ValidarInstantanea() != nil {
		return false
	}
	marca, ok := ctx.Value(claveNecesidadAltaValidada{}).(necesidadAltaValidada)
	return ok && marca.organizacionRef == organizacionRef && reflect.DeepEqual(marca.material, *necesidad)
}

func contextoNecesidadAltaValidada(ctx context.Context, organizacionRef string, necesidad *domain.DatosNecesidadAlta) (context.Context, error) {
	if ctx == nil || necesidad == nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	copia, err := necesidad.Clonar()
	if err != nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	return context.WithValue(ctx, claveNecesidadAltaValidada{}, necesidadAltaValidada{organizacionRef, copia}), nil
}

func copiaNecesidadAltaParaPuerto(necesidad *domain.DatosNecesidadAlta) (*domain.DatosNecesidadAlta, error) {
	if necesidad == nil {
		return nil, nil
	}
	copia, err := necesidad.Clonar()
	if err != nil {
		return nil, ErrSolicitudRegistroInvalida
	}
	return &copia, nil
}

// prepararNecesidadAlta usa primero la instantánea confirmada de la misma
// intención. Eso permite repetir un alta después de cambiar la publicación
// activa sin volver a sellar el expediente con otros bytes.
func (s *ServicioRegistroSolicitud) prepararNecesidadAlta(
	ctx context.Context,
	entrada domain.DatosNecesidadAlta,
	ambitos ports.ColeccionSellosHMAC,
	organizacionRef, actorRef, perfilRef string,
) (domain.DatosNecesidadAlta, error) {
	if s == nil || dependenciaNula(s.fuenteNecesidades) || ctx == nil {
		return domain.DatosNecesidadAlta{}, ErrServicioRegistroInvalido
	}
	consulta := ports.ConsultaInstantaneaNecesidadConfirmada{
		AmbitosHMAC: ambitos, OrganizacionRef: organizacionRef,
		ActorRef: actorRef, PerfilRef: perfilRef,
	}
	if consulta.Validar() != nil {
		return domain.DatosNecesidadAlta{}, ErrSolicitudRegistroInvalida
	}
	resultado, err := s.fuenteNecesidades.RecuperarInstantaneaNecesidadConfirmada(ctx, consulta)
	if err != nil {
		return domain.DatosNecesidadAlta{}, errors.Join(ports.ErrPersistenciaNoDisponible, err)
	}
	if resultado.Validar() != nil {
		return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return domain.DatosNecesidadAlta{}, err
	}
	var catalogo domain.CatalogoNecesidadesAlta
	marca, desdePeticion := ctx.Value(claveNecesidadPeticionRatificada{}).(necesidadPeticionRatificada)
	switch resultado.Estado {
	case ports.InstantaneaNecesidadConfirmadaV3:
		catalogo, err = domain.RestaurarCatalogoNecesidadesAlta(resultado.Instantanea)
	case ports.InstantaneaNecesidadLegadoV2:
		return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
	case ports.InstantaneaNecesidadAusente:
		if desdePeticion {
			sinInstantanea := marca.necesidad
			sinInstantanea.CatalogoInstantanea = nil
			if marca.organizacionRef != organizacionRef || marca.actorRef != actorRef ||
				marca.perfilRef != perfilRef || !ports.ClaveIdempotenciaValida(marca.claveAlta) ||
				!domain.ReferenciaOpacaValida(marca.peticionRef) ||
				!ambitos.Contiene(marca.ambitoAltaHMAC) || !reflect.DeepEqual(sinInstantanea, entrada) {
				return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
			}
			catalogo, err = domain.RestaurarCatalogoNecesidadesAlta(marca.necesidad.CatalogoInstantanea)
		} else {
			catalogo, err = s.fuenteNecesidades.ResolverCatalogoNecesidadesAlta(
				ctx, entrada.CatalogoRef, entrada.CatalogoVersion, entrada.CatalogoHuellaSHA256,
			)
		}
	default:
		return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
	}
	if err != nil || catalogo.Validar() != nil {
		if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 {
			return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
		}
		return domain.DatosNecesidadAlta{}, errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.DatosNecesidadAlta{}, err
	}
	sellada, _, err := sellarNecesidadAltaPublicada(ctx, entrada, catalogo, s.verificadorPuestoRPT,
		resultado.Estado == ports.InstantaneaNecesidadAusente && !desdePeticion)
	if resultado.Estado == ports.InstantaneaNecesidadConfirmadaV3 && errors.Is(err, ErrSolicitudRegistroInvalida) {
		return domain.DatosNecesidadAlta{}, ports.ErrClaveIdempotenciaUsada
	}
	if err == nil && desdePeticion && resultado.Estado == ports.InstantaneaNecesidadAusente &&
		!reflect.DeepEqual(sellada, marca.necesidad) {
		return domain.DatosNecesidadAlta{}, ErrResultadoRegistroNoConfiable
	}
	return sellada, err
}

// PrepararNecesidadPeticionCentro usa la misma autoridad y la misma validación
// de Alta v3. No consulta la intención de alta: la petición aún no tiene MOAD.
func (s *ServicioRegistroSolicitud) PrepararNecesidadPeticionCentro(ctx context.Context,
	entrada domain.DatosNecesidadAlta,
) (domain.DatosNecesidadAlta, *ports.PuestoRPTAltaVerificado, error) {
	if s == nil || ctx == nil || dependenciaNula(s.fuenteNecesidades) || len(entrada.CatalogoInstantanea) != 0 {
		return domain.DatosNecesidadAlta{}, nil, ports.ErrFuenteNecesidadesAltaNoDisponible
	}
	catalogo, err := s.fuenteNecesidades.ResolverCatalogoNecesidadesAlta(ctx,
		entrada.CatalogoRef, entrada.CatalogoVersion, entrada.CatalogoHuellaSHA256)
	if err != nil || catalogo.Validar() != nil {
		return domain.DatosNecesidadAlta{}, nil, errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible, err)
	}
	return sellarNecesidadAltaPublicada(ctx, entrada, catalogo, s.verificadorPuestoRPT, true)
}

func sellarNecesidadAltaPublicada(ctx context.Context, entrada domain.DatosNecesidadAlta,
	catalogo domain.CatalogoNecesidadesAlta, verificador ports.VerificadorPuestoRPTAlta, verificarPuesto bool,
) (domain.DatosNecesidadAlta, *ports.PuestoRPTAltaVerificado, error) {
	if ctx == nil {
		return domain.DatosNecesidadAlta{}, nil, ErrSolicitudRegistroInvalida
	}
	if err := ctx.Err(); err != nil {
		return domain.DatosNecesidadAlta{}, nil, err
	}
	if len(entrada.CatalogoInstantanea) != 0 || catalogo.ValidarDatos(entrada) != nil {
		return domain.DatosNecesidadAlta{}, nil, ErrSolicitudRegistroInvalida
	}
	var verificado *ports.PuestoRPTAltaVerificado
	if verificarPuesto && entrada.Campos["puesto_codigo"] != "" {
		if dependenciaNula(verificador) {
			return domain.DatosNecesidadAlta{}, nil, ErrServicioRegistroInvalido
		}
		solicitudPuesto, err := ports.SolicitudPuestoRPTDesdeCampos(entrada.Campos)
		if err != nil {
			return domain.DatosNecesidadAlta{}, nil, ErrSolicitudRegistroInvalida
		}
		puesto, err := verificador.VerificarPuestoRPTAlta(ctx, solicitudPuesto)
		if err != nil {
			return domain.DatosNecesidadAlta{}, nil, errors.Join(ports.ErrFuenteNecesidadesAltaNoDisponible, err)
		}
		if !puesto.ExisteEnPublicacion {
			return domain.DatosNecesidadAlta{}, nil, ErrSolicitudRegistroInvalida
		}
		if puesto.ValidarPara(solicitudPuesto) != nil {
			return domain.DatosNecesidadAlta{}, nil, ErrResultadoRegistroNoConfiable
		}
		if err := ctx.Err(); err != nil {
			return domain.DatosNecesidadAlta{}, nil, err
		}
		verificado = &puesto
	}
	sellada, err := catalogo.SellarDatos(entrada)
	if err != nil {
		return domain.DatosNecesidadAlta{}, nil, ErrSolicitudRegistroInvalida
	}
	return sellada, verificado, nil
}
